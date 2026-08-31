package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mayfly/pkg/application"
	"mayfly/pkg/domain"
	"mayfly/pkg/tui/terminal"
)

func cmdInit(ctx context.Context, svc *application.Service, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetPath := fs.String("path", ".", "Target project directory to register")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	proj, err := svc.RegisterProject(ctx, *targetPath)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: init failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Initialized project in %s\nProject ID: %s\n", proj.CanonicalPath, proj.ID)
	return 0
}

func cmdSet(ctx context.Context, svc *application.Service, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: mayfly set <NAME> [VALUE]")
		return 2
	}
	secName := domain.SecretName(args[0])
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

	if len(args) < 2 && !isTerm && os.Getenv("MAYFLY_FORCE_NONINTERACTIVE") != "1" && !isTesting() {
		fmt.Fprintln(stderr, "mayfly: 'set' requires an interactive terminal.")
		fmt.Fprintln(stderr, "Secrets must be entered interactively by a human, or via the TUI ('mf c').")
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

	var val string
	if len(args) >= 2 {
		val = strings.Join(args[1:], " ")
	} else if isTerm {
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

	fmt.Fprintf(stdout, "[OK] Secret %s saved for project %s\n", secName, proj.ID)
	return 0
}

func cmdGet(ctx context.Context, svc *application.Service, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: mayfly get <NAME> [--clip]")
		return 2
	}

	clip := false
	var rawName string
	for _, arg := range args {
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
		fmt.Fprintf(stdout, "[OK] Secret %s copied to clipboard.\n", secName)
		return 0
	}

	fmt.Fprintln(stdout, val)
	return 0
}

func cmdList(ctx context.Context, svc *application.Service, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	jsonOutput := false
	for _, arg := range args {
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
}

func cmdDelete(ctx context.Context, svc *application.Service, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: mayfly delete <NAME>")
		return 2
	}
	secName := domain.SecretName(args[0])
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
}

func cmdRun(ctx context.Context, svc *application.Service, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: mayfly run <COMMAND> [ARGS...]")
		return 2
	}
	return executeTargetProcess(ctx, svc, args, stdin, stdout, stderr)
}

func cmdScan(ctx context.Context, svc *application.Service, args []string, stdout, stderr io.Writer) int {
	targetDir := "."
	if len(args) >= 1 {
		targetDir = args[0]
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
}

func cmdAudit(ctx context.Context, svc *application.Service, args []string, stdout, stderr io.Writer) int {
	if len(args) >= 1 && args[0] == "verify" {
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
}

func cmdBackup(ctx context.Context, svc *application.Service, args []string, stdout, stderr io.Writer) int {
	targetFile := "mayfly-backup.json"
	if len(args) >= 1 {
		targetFile = args[0]
	}
	if err := svc.ExportBackup(ctx, targetFile); err != nil {
		fmt.Fprintf(stderr, "mayfly: backup failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Encrypted vault backup exported to %s\n", targetFile)
	return 0
}

func cmdRestore(ctx context.Context, svc *application.Service, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: mayfly restore <BACKUP_FILE>")
		return 2
	}
	if err := svc.RestoreBackup(ctx, args[0]); err != nil {
		fmt.Fprintf(stderr, "mayfly: restore failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Vault and projects restored from %s\n", args[0])
	return 0
}

func cmdMigrate(ctx context.Context, svc *application.Service, args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: mayfly migrate <OLD_PATH> <NEW_PATH>")
		return 2
	}
	oldP, newP, err := svc.MigrateProject(ctx, args[0], args[1])
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: migration failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Migrated project %s (%s) -> %s (%s)\n", oldP.ID, oldP.CanonicalPath, newP.ID, newP.CanonicalPath)
	return 0
}

func cmdImport(ctx context.Context, svc *application.Service, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	targetFile := ".env"
	if len(args) >= 1 {
		targetFile = args[0]
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

	fmt.Fprintf(stdout, "[OK] Imported %d secrets from %s into project %s\n", count, targetFile, proj.ID)
	return 0
}

func cmdRotatePassword(ctx context.Context, svc *application.Service, stdin io.Reader, stdout, stderr io.Writer) int {
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

	fmt.Fprintln(stdout, "[OK] Vault master password rotated successfully. All secrets re-encrypted with fresh salt.")
	return 0
}

func cmdUninstall(stdin io.Reader, stdout io.Writer) int {
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
	fmt.Fprintln(stdout, "[OK] Removed mayfly and mf binaries.")
	fmt.Fprintln(stdout, "[OK] Removed ~/.mayfly directory and all encrypted vaults.")
	fmt.Fprintln(stdout, "MayFly has been completely and cleanly uninstalled from your system.")
	return 0
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
