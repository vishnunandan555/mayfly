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
	"mayfly/pkg/updater"
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
		fmt.Fprintln(stderr, "usage: mayfly set <NAME> [--clip] [VALUE] or mayfly set <NAME>=<VALUE> [--clip]")
		return 2
	}

	// Parse --clip / -c flag and strip it from positional args.
	clip := false
	var positional []string
	for _, arg := range args {
		if arg == "--clip" || arg == "-c" {
			clip = true
		} else {
			positional = append(positional, arg)
		}
	}
	if len(positional) < 1 {
		fmt.Fprintln(stderr, "usage: mayfly set <NAME> [--clip] [VALUE] or mayfly set <NAME>=<VALUE> [--clip]")
		return 2
	}

	var rawName, val string
	hasInlineVal := false
	if name, inlineVal, ok := strings.Cut(positional[0], "="); ok {
		rawName = name
		val = inlineVal
		hasInlineVal = true
	} else {
		rawName = positional[0]
		if len(positional) >= 2 {
			val = strings.Join(positional[1:], " ")
			hasInlineVal = true
		}
	}

	secName := domain.SecretName(rawName)
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

	if !hasInlineVal && !isTerm && os.Getenv("MAYFLY_FORCE_NONINTERACTIVE") != "1" && !isTesting() {
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

	if !hasInlineVal {
		if isTerm {
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
	}

	if err := svc.SetSecret(ctx, proj.ID, secName, val); err != nil {
		fmt.Fprintf(stderr, "mayfly: failed to save secret: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "[OK] Secret %s saved for project %s\n", secName, proj.ID)

	if clip {
		_ = terminal.CopyToClipboard(val, stdout)
		fmt.Fprintf(stdout, "[OK] Value also copied to clipboard.\n")
	}

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
	jsonOutput := false
	severityFilter := ""

	for i, arg := range args {
		switch {
		case arg == "--json":
			jsonOutput = true
		case arg == "--severity" && i+1 < len(args):
			severityFilter = strings.ToUpper(args[i+1])
		case strings.HasPrefix(arg, "--severity="):
			severityFilter = strings.ToUpper(strings.TrimPrefix(arg, "--severity="))
		case !strings.HasPrefix(arg, "-") && targetDir == ".":
			targetDir = arg
		}
	}

	findings, err := svc.Scan(ctx, targetDir)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: scan failed: %v\n", err)
		return 1
	}

	// Apply severity filter.
	if severityFilter != "" {
		var filtered []domain.ScanFinding
		for _, f := range findings {
			if strings.ToUpper(string(f.Severity)) == severityFilter {
				filtered = append(filtered, f)
			}
		}
		findings = filtered
	}

	if len(findings) == 0 {
		if !jsonOutput {
			fmt.Fprintln(stdout, "No plaintext credentials or .env files detected.")
		} else {
			fmt.Fprintln(stdout, "[]")
		}
		return 0
	}

	if jsonOutput {
		data, jErr := json.MarshalIndent(findings, "", "  ")
		if jErr != nil {
			fmt.Fprintf(stderr, "mayfly: json error: %v\n", jErr)
			return 1
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		for _, f := range findings {
			fmt.Fprintf(stdout, "%s:%d:%d: [%s] %s (%s)\n", f.Path, f.Line, f.Column, f.Severity, f.Message, f.Category)
		}
	}

	// Exit 1 if any CRITICAL findings (breaks CI), exit 2 if only WARNINGs.
	hasCritical := false
	for _, f := range findings {
		if f.Severity == domain.SeverityCritical {
			hasCritical = true
			break
		}
	}
	if hasCritical {
		return 1
	}
	return 2 // warnings only
}

func cmdAudit(ctx context.Context, svc *application.Service, args []string, stdout, stderr io.Writer) int {
	jsonOutput := false
	tailN := 0
	var positional []string

	for i, arg := range args {
		switch {
		case arg == "--json":
			jsonOutput = true
		case arg == "--tail" && i+1 < len(args):
			_, _ = fmt.Sscanf(args[i+1], "%d", &tailN)
		case strings.HasPrefix(arg, "--tail="):
			_, _ = fmt.Sscanf(strings.TrimPrefix(arg, "--tail="), "%d", &tailN)
		case !strings.HasPrefix(arg, "-"):
			positional = append(positional, arg)
		}
	}

	if len(positional) >= 1 && positional[0] == "verify" {
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

	// Apply --tail N filter.
	if tailN > 0 && len(events) > tailN {
		events = events[len(events)-tailN:]
	}

	if jsonOutput {
		data, jErr := json.MarshalIndent(events, "", "  ")
		if jErr != nil {
			fmt.Fprintf(stderr, "mayfly: json error: %v\n", jErr)
			return 1
		}
		fmt.Fprintln(stdout, string(data))
		return 0
	}

	for _, ev := range events {
		fmt.Fprintf(stdout, "#%-4d %-25s %-28s project=%-12s secret=%s hash=%s\n",
			ev.Sequence,
			ev.At.Format("2006-01-02T15:04:05Z07:00"),
			ev.Action,
			truncate(ev.ProjectID, 12),
			ev.Secret,
			ev.Hash[:12])
	}
	return 0
}

// truncate shortens s to maxLen characters with "..." suffix if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
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
	forceDelete := false
	noDelete := false

	for _, arg := range args {
		if arg == "--delete" || arg == "-d" {
			forceDelete = true
		} else if arg == "--no-delete" {
			noDelete = true
		} else if !strings.HasPrefix(arg, "-") {
			targetFile = arg
		}
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

	if count > 0 {
		if forceDelete {
			if rmErr := os.Remove(envPath); rmErr != nil {
				fmt.Fprintf(stderr, "mayfly: warning: failed to delete %s: %v\n", targetFile, rmErr)
			} else {
				fmt.Fprintf(stdout, "[OK] Deleted plaintext %s from disk.\n", targetFile)
			}
		} else if !noDelete {
			f, isFile := stdin.(*os.File)
			isTerm := isFile && terminal.IsTerminal(f)
			if isTerm && !isTesting() {
				fmt.Fprintf(stdout, "\nWould you like to delete the plaintext %s file from disk now? [y/N]: ", targetFile)
				resp, _ := readLine(stdin)
				cleanResp := strings.ToLower(strings.TrimSpace(resp))
				if cleanResp == "y" || cleanResp == "yes" {
					if rmErr := os.Remove(envPath); rmErr != nil {
						fmt.Fprintf(stderr, "mayfly: warning: failed to delete %s: %v\n", targetFile, rmErr)
					} else {
						fmt.Fprintf(stdout, "[OK] Deleted plaintext %s from disk.\n", targetFile)
					}
				} else {
					fmt.Fprintf(stdout, "Kept %s on disk. (Tip: add it to .gitignore or remove it when ready).\n", targetFile)
				}
			}
		}
	}

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

func cmdUpdate(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	checkOnly := false
	forceYes := false

	for _, arg := range args {
		if arg == "--check" || arg == "-c" {
			checkOnly = true
		} else if arg == "--yes" || arg == "-y" {
			forceYes = true
		}
	}

	fmt.Fprintf(stdout, "Checking for MayFly updates (current: v%s)...\n", domain.Version)
	rel, isNewer, err := updater.CheckForUpdates(ctx, "")
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: unable to check for updates: %v\n", err)
		return 1
	}

	if !isNewer {
		fmt.Fprintf(stdout, "[OK] MayFly is up to date (v%s is the latest release).\n", domain.Version)
		return 0
	}

	fmt.Fprintf(stdout, "\nA newer release of MayFly is available:\n")
	fmt.Fprintf(stdout, "  Current: v%s\n", domain.Version)
	fmt.Fprintf(stdout, "  Latest:  %s (%s)\n", rel.TagName, rel.Name)
	if rel.HTMLURL != "" {
		fmt.Fprintf(stdout, "  URL:     %s\n", rel.HTMLURL)
	}

	if strings.TrimSpace(rel.Body) != "" {
		fmt.Fprintf(stdout, "\nRelease Notes:\n%s\n", strings.TrimSpace(rel.Body))
	}

	if checkOnly {
		return 0
	}

	f, isFile := stdin.(*os.File)
	isTerm := isFile && terminal.IsTerminal(f)

	if !forceYes {
		if !isTerm && os.Getenv("MAYFLY_FORCE_NONINTERACTIVE") != "1" && !isTesting() {
			fmt.Fprintln(stdout, "\nTo update, run 'mayfly update' interactively or use 'mayfly update --yes'.")
			return 0
		}

		fmt.Fprintf(stdout, "\nWould you like to update to %s now? [y/N]: ", rel.TagName)
		resp, _ := readLine(stdin)
		clean := strings.ToLower(strings.TrimSpace(resp))
		if clean != "y" && clean != "yes" {
			fmt.Fprintln(stdout, "Update postponed. (Run 'mayfly update' anytime).")
			return 0
		}
	}

	fmt.Fprintf(stdout, "\nDownloading and installing %s...\n", rel.TagName)
	if err := updater.PerformUpdate(ctx); err != nil {
		fmt.Fprintf(stderr, "mayfly: update failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "[OK] MayFly successfully updated to %s!\n", rel.TagName)
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

// cmdEnv exports all project secrets as shell environment variable assignments.
// Usage: eval $(mf env) or eval $(mf env --shell fish) or eval (mf env --shell powershell)
func cmdEnv(ctx context.Context, svc *application.Service, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	shell := "bash"
	for _, arg := range args {
		if strings.HasPrefix(arg, "--shell=") {
			shell = strings.ToLower(strings.TrimPrefix(arg, "--shell="))
		}
	}
	for i, arg := range args {
		if arg == "--shell" && i+1 < len(args) {
			shell = strings.ToLower(args[i+1])
		}
	}

	switch shell {
	case "bash", "zsh", "sh", "posix", "fish", "json", "powershell", "pwsh":
		// valid supported shell
	default:
		fmt.Fprintf(stderr, "mayfly: unsupported shell %q (supported: bash, zsh, fish, powershell, json)\n", shell)
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

	secrets, err := svc.ExportSecrets(proj.ID)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: %v\n", err)
		return 1
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	sortStrings(keys)

	switch shell {
	case "fish":
		for _, k := range keys {
			fmt.Fprintf(stdout, "set -x %s %s;\n", k, shellQuote(secrets[k]))
		}
	case "powershell", "pwsh":
		for _, k := range keys {
			// PowerShell single-quote escaping: replace ' with ''
			escaped := strings.ReplaceAll(secrets[k], "'", "''")
			fmt.Fprintf(stdout, "$env:%s = '%s';\n", k, escaped)
		}
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(secrets)
	default: // bash / zsh / sh / posix
		for _, k := range keys {
			fmt.Fprintf(stdout, "export %s=%s\n", k, shellQuote(secrets[k]))
		}
	}
	return 0
}

// shellQuote wraps a string in single quotes, escaping internal single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// sortStrings sorts a string slice in place (stdlib sort.Strings requires sort import).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// cmdStatus shows vault health summary without requiring unlock.
func cmdStatus(ctx context.Context, svc *application.Service, stdout, stderr io.Writer) int {
	info := svc.VaultStatus()

	fmt.Fprintln(stdout, "=== MayFly Status ===")
	if v, ok := info["vault_exists"]; ok && v == "true" {
		fmt.Fprintf(stdout, "  Vault file:     %s\n", info["vault_file"])
		if info["vault_locked"] == "false" {
			fmt.Fprintf(stdout, "  Vault:          unlocked (total secrets: %s)\n", info["total_secrets"])
		} else {
			fmt.Fprintf(stdout, "  Vault:          locked\n")
		}
	} else {
		fmt.Fprintln(stdout, "  Vault:          not initialized (run 'mf init' first)")
	}

	if c, ok := info["project_count"]; ok {
		fmt.Fprintf(stdout, "  Projects:       %s registered\n", c)
	}

	return 0
}

// cmdCheck verifies vault, audit log, and project registry integrity.
func cmdCheck(ctx context.Context, svc *application.Service, stdout, stderr io.Writer) int {
	results := svc.CheckIntegrity(ctx)
	exitCode := 0
	for _, r := range results {
		fmt.Fprintln(stdout, r)
		if strings.HasPrefix(r, "FAIL") {
			exitCode = 1
		}
	}
	return exitCode
}

// cmdTemplate renders a config template file with project secrets injected.
// Usage: mf template config.template.yaml [--output config.yaml]
func cmdTemplate(ctx context.Context, svc *application.Service, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: mayfly template <FILE> [--output FILE]")
		return 2
	}

	templateFile := ""
	outputFile := ""
	for i, arg := range args {
		if arg == "--output" || arg == "-o" {
			if i+1 < len(args) {
				outputFile = args[i+1]
			}
		} else if !strings.HasPrefix(arg, "--") && !strings.HasPrefix(arg, "-") && templateFile == "" {
			templateFile = arg
		}
	}

	if templateFile == "" {
		fmt.Fprintln(stderr, "usage: mayfly template <FILE> [--output FILE]")
		return 2
	}

	cwd, _ := os.Getwd()
	proj, err := svc.ResolveCurrentProject(cwd)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: current directory is not an initialized project\n")
		return 1
	}

	content, err := os.ReadFile(templateFile)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: failed to read template file: %v\n", err)
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

	rendered, err := svc.RenderTemplate(proj.ID, string(content))
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: template rendering failed: %v\n", err)
		return 1
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(rendered), 0600); err != nil {
			fmt.Fprintf(stderr, "mayfly: failed to write output file: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "[OK] Rendered template written to %s\n", outputFile)
	} else {
		fmt.Fprint(stdout, rendered)
	}
	return 0
}

// cmdDiff compares secret keys between two projects.
// Usage: mf diff [PATH_B] (compares current project vs PATH_B)
//        mf diff <PATH_A> <PATH_B> (compares PATH_A vs PATH_B)
func cmdDiff(ctx context.Context, svc *application.Service, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: mayfly diff [PATH_A] <PATH_B>")
		return 2
	}

	cwd, _ := os.Getwd()
	var pathA, pathB string

	if len(args) >= 2 {
		pathA = args[0]
		pathB = args[1]
	} else {
		pathA = cwd
		pathB = args[0]
	}

	projA, err := svc.ResolveCurrentProject(pathA)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: %s is not an initialized project\n", pathA)
		return 1
	}

	projB, err := svc.ResolveCurrentProject(pathB)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: %s is not an initialized project\n", pathB)
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

	onlyA, onlyB, inBoth, err := svc.DiffSecrets(projA.ID, projB.ID)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: %v\n", err)
		return 1
	}

	sortStrings(onlyA)
	sortStrings(onlyB)
	sortStrings(inBoth)

	fmt.Fprintf(stdout, "=== Secret Key Diff: %s vs %s ===\n", projA.CanonicalPath, projB.CanonicalPath)

	if len(inBoth) > 0 {
		fmt.Fprintf(stdout, "\n  In both (%d):\n", len(inBoth))
		for _, k := range inBoth {
			fmt.Fprintf(stdout, "    = %s\n", k)
		}
	}
	if len(onlyA) > 0 {
		fmt.Fprintf(stdout, "\n  Only in A (%d):\n", len(onlyA))
		for _, k := range onlyA {
			fmt.Fprintf(stdout, "    < %s\n", k)
		}
	}
	if len(onlyB) > 0 {
		fmt.Fprintf(stdout, "\n  Only in B (%d):\n", len(onlyB))
		for _, k := range onlyB {
			fmt.Fprintf(stdout, "    > %s\n", k)
		}
	}

	if len(onlyA)+len(onlyB) == 0 {
		fmt.Fprintln(stdout, "\n  Projects have identical secret key sets.")
	}

	return 0
}

// resolveGitHooksDir locates the .git/hooks directory even if .git is a worktree / submodule file.
func resolveGitHooksDir(dir string) (string, error) {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return filepath.Join(gitPath, "hooks"), nil
	}

	// In git worktrees and submodules, .git is a file containing "gitdir: <path>"
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "gitdir:") {
		target := strings.TrimSpace(strings.TrimPrefix(content, "gitdir:"))
		if !filepath.IsAbs(target) {
			target = filepath.Join(dir, target)
		}
		return filepath.Join(target, "hooks"), nil
	}

	return "", fmt.Errorf("invalid .git metadata file in %s", dir)
}

// cmdInstallHook installs a mayfly-aware git pre-commit hook in the current repo.
func cmdInstallHook(args []string, stdout, stderr io.Writer) int {
	cwd, _ := os.Getwd()
	hookDir, err := resolveGitHooksDir(cwd)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: no .git repository found in %s (run from inside a git repository)\n", cwd)
		return 1
	}

	if err := os.MkdirAll(hookDir, 0755); err != nil {
		fmt.Fprintf(stderr, "mayfly: failed to create git hooks directory: %v\n", err)
		return 1
	}

	hookPath := filepath.Join(hookDir, "pre-commit")

	const hookMarker = "# mayfly-managed-hook"
	const hookScript = `#!/bin/sh
# mayfly-managed-hook
# Installed by: mayfly install-hook
# This hook runs 'mf scan' before every commit to catch plaintext secrets.
# Remove with: mf uninstall-hook

set -e

if command -v mayfly >/dev/null 2>&1; then
    MF_BIN="mayfly"
elif command -v mf >/dev/null 2>&1; then
    MF_BIN="mf"
else
    exit 0
fi

if ! OUTPUT=$("$MF_BIN" scan . --severity CRITICAL 2>&1); then
    echo "=== MayFly Pre-Commit Scan FAILED ==="
    echo "$OUTPUT"
    echo ""
    echo "Commit blocked: CRITICAL plaintext secrets detected."
    echo "Fix the findings above, then commit again."
    exit 1
fi

exit 0
`

	// Check if hook already exists and was not installed by mayfly.
	if data, err := os.ReadFile(hookPath); err == nil {
		if !strings.Contains(string(data), hookMarker) {
			fmt.Fprintf(stderr, "mayfly: a custom pre-commit hook already exists at %s\n", hookPath)
			fmt.Fprintln(stderr, "  Back it up and remove it, then run 'mf install-hook' again.")
			return 1
		}
		// Already installed by mayfly — overwrite/update.
	}

	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		fmt.Fprintf(stderr, "mayfly: failed to write hook: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "[OK] Pre-commit hook installed at %s\n", hookPath)
	fmt.Fprintln(stdout, "     Every 'git commit' will now run 'mf scan' and block CRITICAL findings.")
	return 0
}

// cmdUninstallHook removes the mayfly pre-commit hook if it was installed by mayfly.
func cmdUninstallHook(stdout, stderr io.Writer) int {
	cwd, _ := os.Getwd()
	hookDir, err := resolveGitHooksDir(cwd)
	if err != nil {
		fmt.Fprintln(stdout, "No .git repository found.")
		return 0
	}

	hookPath := filepath.Join(hookDir, "pre-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(stdout, "No pre-commit hook found.")
			return 0
		}
		fmt.Fprintf(stderr, "mayfly: failed to read hook: %v\n", err)
		return 1
	}

	if !strings.Contains(string(data), "# mayfly-managed-hook") {
		fmt.Fprintln(stderr, "mayfly: the existing pre-commit hook was not installed by mayfly — not removing it.")
		return 1
	}

	if err := os.Remove(hookPath); err != nil {
		fmt.Fprintf(stderr, "mayfly: failed to remove hook: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "[OK] Pre-commit hook removed from %s\n", hookPath)
	return 0
}

