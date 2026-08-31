package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mayfly/pkg/application"
	"mayfly/pkg/audit"
	"mayfly/pkg/domain"
	"mayfly/pkg/executor"
	"mayfly/pkg/project"
	"mayfly/pkg/scanner"
	"mayfly/pkg/tui"
	"mayfly/pkg/tui/terminal"
	"mayfly/pkg/vault"
)

func main() {
	code := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	ctx := context.Background()

	// Initialize dependencies
	reg, err := project.NewRegistry("")
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: failed to initialize project registry: %v\n", err)
		return 1
	}

	storage, err := vault.NewStorage("", 0)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: failed to initialize vault storage: %v\n", err)
		return 1
	}

	execEngine := executor.NewProcessExecutor(stdin, stdout, stderr)
	auditLog, aErr := audit.New("")
	if aErr != nil {
		fmt.Fprintf(stderr, "mayfly: warning: audit log initialization failed: %v\n", aErr)
	}
	leakScanner, sErr := scanner.New(scanner.Options{})
	if sErr != nil {
		fmt.Fprintf(stderr, "mayfly: warning: scanner initialization failed: %v\n", sErr)
	}

	svc := application.NewService(application.Dependencies{
		Projects: reg,
		Vault:    storage,
		Executor: execEngine,
		Auditor:  auditLog,
		Scanner:  leakScanner,
	})

	// 1. If no args provided, launch Global TUI directly!
	if len(args) == 0 {
		if err := tui.Run(svc, tui.Options{}); err != nil {
			fmt.Fprintf(stderr, "mayfly: tui error: %v\n", err)
			return 1
		}
		return 0
	}

	subcmd := args[0]
	subArgs := args[1:]

	switch subcmd {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0

	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "mayfly v%s (zero-dependency)\n", domain.Version)
		return 0

	case "c", "current", "tui":
		// Launch Project-Scoped TUI
		cwd, _ := os.Getwd()
		proj, err := svc.ResolveCurrentProject(cwd)
		opts := tui.Options{CurrentDir: cwd}
		if err == nil {
			opts.ProjectScoped = &proj
		}
		if err := tui.Run(svc, opts); err != nil {
			fmt.Fprintf(stderr, "mayfly: tui error: %v\n", err)
			return 1
		}
		return 0

	case "init":
		fs := flag.NewFlagSet("init", flag.ContinueOnError)
		fs.SetOutput(stderr)
		targetPath := fs.String("path", ".", "Target project directory to register")
		if err := fs.Parse(subArgs); err != nil {
			return 2
		}

		proj, err := svc.RegisterProject(ctx, *targetPath)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: init failed: %v\n", err)
			return 1
		}

		fmt.Fprintf(stdout, "Initialized project in %s\nProject ID: %s\n", proj.CanonicalPath, proj.ID)
		return 0

	case "set":
		if len(subArgs) < 1 {
			fmt.Fprintln(stderr, "usage: mayfly set <NAME> [VALUE]")
			return 2
		}
		secName := domain.SecretName(subArgs[0])
		if err := secName.Validate(); err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		cwd, _ := os.Getwd()
		proj, err := svc.ResolveCurrentProject(cwd)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: current directory is not an initialized project (run 'mayfly init' first)\n")
			return 1
		}

		f, isFile := stdin.(*os.File)
		isTerm := isFile && terminal.IsTerminal(f)

		if len(subArgs) < 2 && !isTerm && os.Getenv("MAYFLY_FORCE_NONINTERACTIVE") != "1" && !isTesting() {
			fmt.Fprintln(stderr, "mayfly: 'set' requires an interactive terminal.")
			fmt.Fprintln(stderr, "Secrets must be entered interactively by a human, or via the TUI ('mf c').")
			return 1
		}

		// Read password & unlock vault
		password, err := getMasterPassword(svc, stdin, stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		if err := svc.UnlockVault(ctx, password); err != nil {
			fmt.Fprintf(stderr, "mayfly: failed to unlock vault: %v\n", err)
			return 1
		}

		var val string
		if len(subArgs) >= 2 {
			val = strings.Join(subArgs[1:], " ")
		} else if isTerm {
			// Clean ephemeral alt-screen input
			term := terminal.NewTerminal(stdout, terminal.Size{Rows: 24, Columns: 80})
			term.EnterAltScreen()
			term.ClearScreen()
			fmt.Fprintf(stdout, "\n\n  Enter value for %s: ", secName)

			val, err = readLine(stdin)
			term.ExitAltScreen()

			if err != nil {
				fmt.Fprintf(stderr, "mayfly: failed to read secret value: %v\n", err)
				return 1
			}

			if strings.TrimSpace(val) == "" {
				fmt.Fprintln(stdout, "Key not created: value was empty.")
				return 0
			}
		} else {
			// Non-interactive fallback (e.g. testing readers)
			fmt.Fprintf(stdout, "Enter value for %s: ", secName)
			val, err = readLine(stdin)
			if err != nil {
				fmt.Fprintf(stderr, "mayfly: failed to read secret value: %v\n", err)
				return 1
			}
			if strings.TrimSpace(val) == "" {
				fmt.Fprintln(stdout, "Key not created: value was empty.")
				return 0
			}
		}

		if err := svc.SetSecret(ctx, proj.ID, secName, val); err != nil {
			fmt.Fprintf(stderr, "mayfly: failed to save secret: %v\n", err)
			return 1
		}

		fmt.Fprintf(stdout, "✓ Secret %s saved for project %s\n", secName, proj.ID)
		return 0

	case "get":
		if len(subArgs) < 1 {
			fmt.Fprintln(stderr, "usage: mayfly get <NAME> [--clip]")
			return 2
		}

		clip := false
		var rawName string
		for _, arg := range subArgs {
			if arg == "--clip" || arg == "-c" {
				clip = true
			} else if rawName == "" {
				rawName = arg
			}
		}

		if rawName == "" {
			fmt.Fprintln(stderr, "usage: mayfly get <NAME> [--clip]")
			return 2
		}

		secName := domain.SecretName(rawName)
		if err := secName.Validate(); err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		cwd, _ := os.Getwd()
		proj, err := svc.ResolveCurrentProject(cwd)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: current directory is not an initialized project\n")
			return 1
		}

		password, err := getMasterPassword(svc, stdin, stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		if err := svc.UnlockVault(ctx, password); err != nil {
			fmt.Fprintf(stderr, "mayfly: failed to unlock vault: %v\n", err)
			return 1
		}

		val, err := svc.GetSecret(ctx, proj.ID, secName)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		if clip {
			_ = terminal.CopyToClipboard(val, stdout)
			fmt.Fprintf(stdout, "✓ Secret %s copied to clipboard.\n", secName)
			return 0
		}

		fmt.Fprintln(stdout, val)
		return 0

	case "list":
		jsonOutput := false
		for _, arg := range subArgs {
			if arg == "--json" {
				jsonOutput = true
			}
		}

		cwd, _ := os.Getwd()
		proj, err := svc.ResolveCurrentProject(cwd)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: current directory is not an initialized project\n")
			return 1
		}

		password, err := getMasterPassword(svc, stdin, stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		if err := svc.UnlockVault(ctx, password); err != nil {
			fmt.Fprintf(stderr, "mayfly: failed to unlock vault: %v\n", err)
			return 1
		}

		list, err := svc.ListSecrets(proj.ID)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		if jsonOutput {
			data, jErr := json.MarshalIndent(list, "", "  ")
			if jErr != nil {
				fmt.Fprintf(stderr, "mayfly: json error: %v\n", jErr)
				return 1
			}
			fmt.Fprintln(stdout, string(data))
			return 0
		}

		if len(list) == 0 {
			fmt.Fprintln(stdout, "No secrets configured for this project.")
			return 0
		}

		for _, s := range list {
			fmt.Fprintln(stdout, s.Name)
		}
		return 0

	case "delete":
		if len(subArgs) < 1 {
			fmt.Fprintln(stderr, "usage: mayfly delete <NAME>")
			return 2
		}
		secName := domain.SecretName(subArgs[0])
		if err := secName.Validate(); err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		cwd, _ := os.Getwd()
		proj, err := svc.ResolveCurrentProject(cwd)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: current directory is not an initialized project\n")
			return 1
		}

		password, err := getMasterPassword(svc, stdin, stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		if err := svc.UnlockVault(ctx, password); err != nil {
			fmt.Fprintf(stderr, "mayfly: failed to unlock vault: %v\n", err)
			return 1
		}

		if err := svc.DeleteSecret(ctx, proj.ID, secName); err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		fmt.Fprintf(stdout, "Deleted secret %s from project %s\n", secName, proj.ID)
		return 0

	case "run":
		if len(subArgs) < 1 {
			fmt.Fprintln(stderr, "usage: mayfly run <COMMAND> [ARGS...]")
			return 2
		}
		return executeTargetProcess(ctx, svc, subArgs, stdin, stdout, stderr)

	case "scan":
		targetDir := "."
		if len(subArgs) >= 1 {
			targetDir = subArgs[0]
		}

		findings, err := svc.Scan(ctx, targetDir)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: scan failed: %v\n", err)
			return 1
		}

		if len(findings) == 0 {
			fmt.Fprintln(stdout, "No plaintext credentials or .env files detected.")
			return 0
		}

		for _, f := range findings {
			fmt.Fprintf(stdout, "%s:%d:%d: [%s] %s (%s)\n", f.Path, f.Line, f.Column, f.Severity, f.Message, f.Category)
		}
		return 0

	case "audit":
		if len(subArgs) >= 1 && subArgs[0] == "verify" {
			if err := svc.VerifyAudit(ctx); err != nil {
				fmt.Fprintf(stderr, "mayfly: audit log verification FAILED: %v\n", err)
				return 1
			}
			fmt.Fprintln(stdout, "Audit log hash chain verified successfully.")
			return 0
		}

		events, err := svc.AuditTrail(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: failed to read audit trail: %v\n", err)
			return 1
		}

		for _, ev := range events {
			fmt.Fprintf(stdout, "#%d %s %s project=%s secret=%s hash=%s\n",
				ev.Sequence, ev.At.Format("2006-01-02T15:04:05Z07:00"), ev.Action, ev.ProjectID, ev.Secret, ev.Hash[:12])
		}
		return 0

	case "backup":
		targetFile := "mayfly-backup.json"
		if len(subArgs) >= 1 {
			targetFile = subArgs[0]
		}
		if err := svc.ExportBackup(ctx, targetFile); err != nil {
			fmt.Fprintf(stderr, "mayfly: backup failed: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Encrypted vault backup exported to %s\n", targetFile)
		return 0

	case "restore":
		if len(subArgs) < 1 {
			fmt.Fprintln(stderr, "usage: mayfly restore <BACKUP_FILE>")
			return 2
		}
		if err := svc.RestoreBackup(ctx, subArgs[0]); err != nil {
			fmt.Fprintf(stderr, "mayfly: restore failed: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Vault and projects restored from %s\n", subArgs[0])
		return 0

	case "migrate":
		if len(subArgs) < 2 {
			fmt.Fprintln(stderr, "usage: mayfly migrate <OLD_PATH> <NEW_PATH>")
			return 2
		}
		oldP, newP, err := svc.MigrateProject(ctx, subArgs[0], subArgs[1])
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: migration failed: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Migrated project %s (%s) -> %s (%s)\n", oldP.ID, oldP.CanonicalPath, newP.ID, newP.CanonicalPath)
		return 0

	case "import":
		targetFile := ".env"
		if len(subArgs) >= 1 {
			targetFile = subArgs[0]
		}

		cwd, _ := os.Getwd()
		proj, err := svc.ResolveCurrentProject(cwd)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: current directory is not an initialized project (run 'mayfly init' first)\n")
			return 1
		}

		envPath := targetFile
		if !filepath.IsAbs(envPath) {
			envPath = filepath.Join(cwd, targetFile)
		}

		content, err := os.ReadFile(envPath)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: failed to read %s: %v\n", targetFile, err)
			return 1
		}

		password, err := getMasterPassword(svc, stdin, stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		if err := svc.UnlockVault(ctx, password); err != nil {
			fmt.Fprintf(stderr, "mayfly: failed to unlock vault: %v\n", err)
			return 1
		}

		count, err := svc.ImportEnv(ctx, proj.ID, string(content))
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: import failed: %v\n", err)
			return 1
		}

		fmt.Fprintf(stdout, "✓ Imported %d secrets from %s into project %s\n", count, targetFile, proj.ID)
		return 0

	case "rotate-password":
		if !svc.VaultExists() {
			fmt.Fprintf(stderr, "mayfly: vault has not been initialized yet\n")
			return 1
		}

		f, isFile := stdin.(*os.File)
		isTerm := isFile && terminal.IsTerminal(f)

		fmt.Fprint(stderr, "Current Master Password: ")
		var oldPass string
		if isTerm {
			passBytes, err := terminal.ReadPassword(f)
			if err != nil {
				fmt.Fprintf(stderr, "\nmayfly: %v\n", err)
				return 1
			}
			fmt.Fprintln(stderr)
			oldPass = string(passBytes)
		} else {
			var err error
			oldPass, err = readLine(stdin)
			if err != nil {
				fmt.Fprintf(stderr, "mayfly: %v\n", err)
				return 1
			}
		}

		fmt.Fprint(stderr, "New Master Password: ")
		var newPass string
		if isTerm {
			passBytes, err := terminal.ReadPassword(f)
			if err != nil {
				fmt.Fprintf(stderr, "\nmayfly: %v\n", err)
				return 1
			}
			fmt.Fprintln(stderr)
			newPass = string(passBytes)

			fmt.Fprint(stderr, "Confirm New Master Password: ")
			confirmBytes, err := terminal.ReadPassword(f)
			if err != nil {
				fmt.Fprintf(stderr, "\nmayfly: %v\n", err)
				return 1
			}
			fmt.Fprintln(stderr)
			if newPass != string(confirmBytes) {
				fmt.Fprintln(stderr, "mayfly: new passwords do not match")
				return 1
			}
		} else {
			var err error
			newPass, err = readLine(stdin)
			if err != nil {
				fmt.Fprintf(stderr, "mayfly: %v\n", err)
				return 1
			}
		}

		if strings.TrimSpace(newPass) == "" {
			fmt.Fprintln(stderr, "mayfly: new password cannot be empty")
			return 1
		}

		if err := svc.RotatePassword(ctx, []byte(oldPass), []byte(newPass)); err != nil {
			fmt.Fprintf(stderr, "mayfly: password rotation failed: %v\n", err)
			return 1
		}

		fmt.Fprintln(stdout, "✓ Vault master password rotated successfully. All secrets re-encrypted with fresh salt.")
		return 0

	case "completion":
		shell := "bash"
		if len(subArgs) >= 1 {
			shell = strings.ToLower(subArgs[0])
		}
		switch shell {
		case "bash":
			printBashCompletion(stdout)
		case "zsh":
			printZshCompletion(stdout)
		case "fish":
			printFishCompletion(stdout)
		default:
			fmt.Fprintf(stderr, "mayfly: unsupported shell %q (supported: bash, zsh, fish)\n", shell)
			return 1
		}
		return 0

	case "uninstall":
		fmt.Fprintln(stdout, "=================================================")
		fmt.Fprintln(stdout, "  MayFly Complete Uninstaller")
		fmt.Fprintln(stdout, "=================================================")
		fmt.Fprintln(stdout, "WARNING: This will completely remove the 'mayfly' and 'mf'")
		fmt.Fprintln(stdout, "binaries, clean your shell PATH, and PERMANENTLY DELETE")
		fmt.Fprintln(stdout, "all encrypted secrets in ~/.mayfly.")
		fmt.Fprintln(stdout, "")
		fmt.Fprint(stdout, "Are you sure you want to completely uninstall MayFly? [y/N]: ")

		resp, _ := readLine(stdin)
		if strings.ToLower(strings.TrimSpace(resp)) != "y" {
			fmt.Fprintln(stdout, "Uninstallation canceled.")
			return 0
		}

		home, _ := os.UserHomeDir()
		binPaths := []string{
			filepath.Join(home, ".local", "bin", "mayfly"),
			filepath.Join(home, ".local", "bin", "mf"),
			"/usr/local/bin/mayfly",
			"/usr/local/bin/mf",
		}
		for _, bp := range binPaths {
			_ = os.Remove(bp)
		}

		_ = os.RemoveAll(filepath.Join(home, ".mayfly"))
		fmt.Fprintln(stdout, "✓ Removed mayfly and mf binaries.")
		fmt.Fprintln(stdout, "✓ Removed ~/.mayfly directory and all encrypted vaults.")
		fmt.Fprintln(stdout, "MayFly has been completely and cleanly uninstalled from your system.")
		return 0

	default:
		// Transparent execution: 'mayfly npm run dev' or 'mf npm run dev'
		return executeTargetProcess(ctx, svc, args, stdin, stdout, stderr)
	}
}

func executeTargetProcess(ctx context.Context, svc *application.Service, cmdArgs []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(cmdArgs) == 0 {
		printUsage(stderr)
		return 2
	}

	cwd, _ := os.Getwd()
	proj, err := svc.ResolveCurrentProject(cwd)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: current directory is not an initialized project (run 'mayfly init' first)\n")
		return 1
	}

	password, err := getMasterPassword(svc, stdin, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: %v\n", err)
		return 1
	}

	if err := svc.UnlockVault(ctx, password); err != nil {
		fmt.Fprintf(stderr, "mayfly: failed to unlock vault: %v\n", err)
		return 1
	}

	req := domain.ExecutionRequest{
		ProjectID: proj.ID,
		Command:   cmdArgs,
		Dir:       cwd,
	}

	res, err := svc.Run(ctx, req)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: execution error: %v\n", err)
		return res.ExitCode
	}
	return res.ExitCode
}

func getMasterPassword(svc *application.Service, in io.Reader, out, errOut io.Writer) ([]byte, error) {
	if envPass := os.Getenv("MAYFLY_VAULT_PASSWORD"); envPass != "" {
		return []byte(envPass), nil
	}

	f, isFile := in.(*os.File)
	isTerm := isFile && terminal.IsTerminal(f)

	if !svc.VaultExists() {
		fmt.Fprint(errOut, "Create Master Password: ")
		var p1 string
		var err error
		if isTerm {
			passBytes, rErr := terminal.ReadPassword(f)
			if rErr != nil {
				return nil, rErr
			}
			p1 = string(passBytes)
			fmt.Fprintln(errOut)
		} else {
			p1, err = readLine(in)
			if err != nil {
				return nil, err
			}
		}
		if p1 == "" {
			return nil, errors.New("password cannot be empty")
		}

		if isTerm {
			fmt.Fprint(errOut, "Confirm Master Password: ")
			passBytes2, rErr := terminal.ReadPassword(f)
			if rErr != nil {
				return nil, rErr
			}
			fmt.Fprintln(errOut)
			p2 := string(passBytes2)
			if p1 != p2 {
				return nil, errors.New("passwords do not match")
			}
		}

		if err := svc.InitializeVault(context.Background(), []byte(p1)); err != nil {
			return nil, err
		}
		return []byte(p1), nil
	}

	fmt.Fprint(errOut, "Vault password: ")
	var p string
	var err error
	if isTerm {
		passBytes, rErr := terminal.ReadPassword(f)
		if rErr != nil {
			return nil, rErr
		}
		fmt.Fprintln(errOut)
		p = string(passBytes)
	} else {
		p, err = readLine(in)
		if err != nil {
			return nil, err
		}
	}
	return []byte(p), nil
}

func isTesting() bool {
	return os.Getenv("GO_TESTING") == "1" || flag.Lookup("test.v") != nil
}

func readLine(in io.Reader) (string, error) {
	var buf [1]byte
	var line []byte
	for {
		n, err := in.Read(buf[:])
		if n > 0 {
			b := buf[0]
			if b == '\n' {
				break
			}
			if b != '\r' {
				line = append(line, b)
			}
		}
		if err != nil {
			if len(line) > 0 && err == io.EOF {
				return string(line), nil
			}
			if len(line) == 0 && err == io.EOF {
				return "", io.EOF
			}
			return string(line), err
		}
	}
	return string(line), nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `MayFly — Zero-Dependency Secrets Workspace & Process Injector

Usage:
  mayfly <COMMAND> [ARGS...]          Inject secrets in memory and execute process directly
  mayfly                              Launch interactive Global TUI Dashboard (Project Grid)
  mayfly c, mayfly current            Launch TUI scoped directly to current project
  mayfly init [-path DIR]             Initialize project in current directory or target path
  mayfly set <NAME> [VALUE]           Add or update an encrypted secret (alt-screen if interactive)
  mayfly get <NAME> [--clip]          Output decrypted secret or copy to clipboard (-c)
  mayfly list [--json]                List secret keys for the current project
  mayfly delete <NAME>                Remove a secret from the vault
  mayfly import [FILE]                Import secrets from .env file into vault (default: .env)
  mayfly rotate-password              Re-encrypt vault with a new master password
  mayfly run <COMMAND> [ARGS...]      Explicit process execution alias
  mayfly scan [DIR]                   Scan codebase for plaintext secret leaks (.mayflyignore supported)
  mayfly audit [verify]               View or cryptographically verify audit log
  mayfly backup [FILE]                Export encrypted vault backup snapshot
  mayfly restore <FILE>               Restore vault and projects from backup snapshot
  mayfly migrate <OLD> <NEW>          Update project identity when directory moves
  mayfly completion <SHELL>           Generate autocompletion script (bash, zsh, fish)
  mayfly uninstall                    Cleanly uninstall binaries and remove data
  mayfly version                      Show version information
  mayfly help                         Show this help message

Short alias:
  All commands work with 'mf' as well:
    mf npm run dev                    Run dev server with injected secrets
    mf set STRIPE_KEY                 Enter secret value in clean ephemeral alt-screen
    mf get STRIPE_KEY --clip          Copy secret directly to clipboard
    mf import .env                    Migrate existing .env file into vault
    mf rotate-password                Rotate vault encryption master password
    mf                                Open TUI dashboard`)
}

func printBashCompletion(w io.Writer) {
	fmt.Fprintln(w, `_mayfly() {
    local cur prev words cword
    _init_completion || return

    local commands="init set get list delete run scan audit backup restore migrate import rotate-password completion uninstall version help c current"

    if [[ ${cword} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi

    case "${prev}" in
        get|delete|set)
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
            return 0
            ;;
        audit)
            COMPREPLY=( $(compgen -W "verify" -- ${cur}) )
            return 0
            ;;
    esac
}
complete -F _mayfly mayfly mf`)
}

func printZshCompletion(w io.Writer) {
	fmt.Fprintln(w, `#compdef mayfly mf

_mayfly() {
    local -a commands
    commands=(
        'init:Initialize project in current directory'
        'set:Add or update an encrypted secret'
        'get:Output decrypted secret to stdout or clipboard'
        'list:List all secret keys for current project'
        'delete:Remove a secret from the vault'
        'import:Import secrets from .env file into vault'
        'rotate-password:Re-encrypt vault with new master password'
        'run:Inject secrets in memory and execute process'
        'scan:Scan codebase for plaintext secret leaks'
        'audit:View or cryptographically verify audit log'
        'backup:Export encrypted vault backup snapshot'
        'restore:Restore vault and projects from backup snapshot'
        'migrate:Update project identity when directory moves'
        'completion:Generate shell autocompletion script'
        'uninstall:Uninstall mayfly and remove data'
        'version:Show version information'
        'help:Show help message'
        'c:Launch TUI scoped to current project'
    )

    if (( CURRENT == 2 )); then
        _describe -t commands 'mayfly commands' commands
    fi
}

compdef _mayfly mayfly mf`)
}

func printFishCompletion(w io.Writer) {
	fmt.Fprintln(w, `complete -c mayfly -f
complete -c mf -f

complete -c mayfly -n "__fish_use_subcommand" -a init -d "Initialize project in current directory"
complete -c mayfly -n "__fish_use_subcommand" -a set -d "Add or update an encrypted secret"
complete -c mayfly -n "__fish_use_subcommand" -a get -d "Output decrypted secret"
complete -c mayfly -n "__fish_use_subcommand" -a list -d "List secret keys"
complete -c mayfly -n "__fish_use_subcommand" -a delete -d "Remove a secret"
complete -c mayfly -n "__fish_use_subcommand" -a import -d "Import .env file into vault"
complete -c mayfly -n "__fish_use_subcommand" -a rotate-password -d "Re-encrypt vault with new password"
complete -c mayfly -n "__fish_use_subcommand" -a run -d "Inject secrets in memory and execute process"
complete -c mayfly -n "__fish_use_subcommand" -a scan -d "Scan codebase for plaintext secret leaks"
complete -c mayfly -n "__fish_use_subcommand" -a audit -d "View or verify audit log"
complete -c mayfly -n "__fish_use_subcommand" -a backup -d "Export encrypted backup"
complete -c mayfly -n "__fish_use_subcommand" -a restore -d "Restore vault backup"
complete -c mayfly -n "__fish_use_subcommand" -a migrate -d "Migrate project directory"
complete -c mayfly -n "__fish_use_subcommand" -a completion -d "Generate shell completion"
complete -c mayfly -n "__fish_use_subcommand" -a version -d "Show version"

complete -c mf -w mayfly`)
}

