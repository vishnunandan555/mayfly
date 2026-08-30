package main

import (
	"context"
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
		fmt.Fprintln(stdout, "mayfly v1.0.0 (zero-dependency)")
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
			fmt.Fprintln(stderr, "usage: mayfly get <NAME>")
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

		val, err := svc.GetSecret(ctx, proj.ID, secName)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: %v\n", err)
			return 1
		}

		fmt.Fprintln(stdout, val)
		return 0

	case "list":
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
			Command:   subArgs,
			Dir:       cwd,
		}

		res, err := svc.Run(ctx, req)
		if err != nil {
			fmt.Fprintf(stderr, "mayfly: execution error: %v\n", err)
			return res.ExitCode
		}
		return res.ExitCode

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
		fmt.Fprintf(stderr, "mayfly: unknown command %q\n\n", subcmd)
		printUsage(stderr)
		return 2
	}
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
  mayfly                              Launch interactive Global TUI Dashboard (Project Grid)
  mayfly c, mayfly current            Launch TUI scoped directly to current project
  mayfly init [-path DIR]             Initialize project in current directory or target path
  mayfly set <NAME> [VALUE]           Add or update an encrypted secret
  mayfly get <NAME>                   Output decrypted secret to stdout
  mayfly list                         List all secret keys for the current project
  mayfly delete <NAME>                Remove a secret from the vault
  mayfly run <COMMAND> [ARGS...]      Inject secrets in memory and execute process
  mayfly scan [DIR]                   Scan codebase for plaintext secret leaks
  mayfly audit [verify]               View or cryptographically verify audit log
  mayfly backup [FILE]                Export encrypted vault backup snapshot
  mayfly restore <FILE>               Restore vault and projects from backup snapshot
  mayfly migrate <OLD> <NEW>          Update project identity when directory moves
  mayfly uninstall                    Cleanly uninstall binaries and remove data
  mayfly version                      Show version information
  mayfly help                         Show this help message

Short alias:
  All commands work with 'mf' as well (e.g. 'mf', 'mf c', 'mf run npm start').`)
}

