package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"mayfly/pkg/domain"
)

func shellCmd(script string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", script}
	}
	return []string{"sh", "-c", script}
}

func executeMayfly(t *testing.T, args []string, input string, dir string) (int, string, string) {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(oldDir) }()
	}

	var stdout, stderr bytes.Buffer
	in := strings.NewReader(input)
	code := run(args, in, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestCompleteCLIWorkflow(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tempHome)
		t.Setenv("HOMEDRIVE", "")
		t.Setenv("HOMEPATH", "")
	} else {
		t.Setenv("USERPROFILE", tempHome)
	}

	projDir := filepath.Join(t.TempDir(), "test-app")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 1. Initialize project
	code, stdout, stderr := executeMayfly(t, []string{"init", "-path", projDir}, "", "")
	if code != 0 {
		t.Fatalf("init failed: code=%d, err=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Initialized project") {
		t.Fatalf("unexpected init output: %s", stdout)
	}

	// 2. Set secret (creates master password)
	code, stdout, stderr = executeMayfly(t, []string{"set", "DATABASE_URL", "postgres://localhost/db"}, "masterpass\n", projDir)
	if code != 0 {
		t.Fatalf("set failed: code=%d, err=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Secret DATABASE_URL saved") {
		t.Fatalf("unexpected set output: %s", stdout)
	}

	// 2b. Set secret with empty value
	code, stdout, stderr = executeMayfly(t, []string{"set", "EMPTY_VAR"}, "masterpass\n\n", projDir)
	if code != 0 {
		t.Fatalf("empty set failed: code=%d, err=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Key not created: value was empty") {
		t.Fatalf("unexpected empty set output: %s", stdout)
	}

	// 3. Get secret
	code, stdout, stderr = executeMayfly(t, []string{"get", "DATABASE_URL"}, "masterpass\n", projDir)
	if code != 0 {
		t.Fatalf("get failed: code=%d, err=%s", code, stderr)
	}
	if strings.TrimSpace(stdout) != "postgres://localhost/db" {
		t.Fatalf("unexpected get output: %s", stdout)
	}

	// 3b. Get secret with invalid name
	code, _, stderr = executeMayfly(t, []string{"get", "123-INVALID"}, "masterpass\n", projDir)
	if code == 0 || !strings.Contains(stderr, "invalid secret name") {
		t.Fatalf("expected validation error for invalid name on get: code=%d, err=%s", code, stderr)
	}

	// 4. List secrets
	code, stdout, stderr = executeMayfly(t, []string{"list"}, "masterpass\n", projDir)
	if code != 0 {
		t.Fatalf("list failed: code=%d, err=%s", code, stderr)
	}
	if !strings.Contains(stdout, "DATABASE_URL") {
		t.Fatalf("unexpected list output: %s", stdout)
	}

<<<<<<< HEAD
	// 5. Run command with injected secret (explicit 'run')
	runScript := "echo DB=$DATABASE_URL"
	if runtime.GOOS == "windows" {
		runScript = "echo DB=%DATABASE_URL%"
	}
	code, stdout, stderr = executeMayfly(t, append([]string{"run"}, shellCmd(runScript)...), "masterpass\n", projDir)
	if code != 0 {
		t.Fatalf("run failed: code=%d, err=%s", code, stderr)
	}
	if !strings.Contains(stdout, "DB=postgres://localhost/db") {
		t.Fatalf("unexpected run output: %s", stdout)
	}

	// 5b. Direct command execution without 'run' (e.g. 'mayfly <cmd>' / 'mf <cmd>')
	directScript := "echo DIRECT_DB=$DATABASE_URL"
	if runtime.GOOS == "windows" {
		directScript = "echo DIRECT_DB=%DATABASE_URL%"
	}
	code, stdout, stderr = executeMayfly(t, shellCmd(directScript), "masterpass\n", projDir)
=======
	// 5. Direct transparent command execution (e.g. 'mayfly <cmd>' / 'mf <cmd>')
	var execArgs []string
	if runtime.GOOS == "windows" {
		execArgs = []string{"cmd.exe", "/c", "echo DIRECT_DB=%DATABASE_URL%"}
	} else {
		execArgs = []string{"sh", "-c", "echo DIRECT_DB=$DATABASE_URL"}
	}
	code, stdout, stderr = executeMayfly(t, execArgs, "masterpass\n", projDir)
>>>>>>> 2449f154ba575ec0bb8803642cbb78f38d9b5e45
	if code != 0 {
		t.Fatalf("direct execution failed: code=%d, err=%s", code, stderr)
	}
	if !strings.Contains(stdout, "DIRECT_DB=postgres://localhost/db") {
		t.Fatalf("unexpected direct execution output: %s", stdout)
	}

	// 5b. Explicit command execution via 'mayfly run <cmd>'
	var runArgs []string
	if runtime.GOOS == "windows" {
		runArgs = append([]string{"run", "cmd.exe", "/c"}, "echo EXPLICIT_DB=%DATABASE_URL%")
	} else {
		runArgs = []string{"run", "sh", "-c", "echo EXPLICIT_DB=$DATABASE_URL"}
	}
	code, stdout, stderr = executeMayfly(t, runArgs, "masterpass\n", projDir)
	if code != 0 {
		t.Fatalf("explicit run execution failed: code=%d, err=%s", code, stderr)
	}
	if !strings.Contains(stdout, "EXPLICIT_DB=postgres://localhost/db") {
		t.Fatalf("unexpected explicit run output: %s", stdout)
	}


	// 6. Plaintext scanner
	code, stdout, stderr = executeMayfly(t, []string{"scan"}, "", projDir)
	if code != 0 {
		t.Fatalf("scan failed: code=%d, err=%s", code, stderr)
	}

	// 7. Audit log & verification
	code, stdout, stderr = executeMayfly(t, []string{"audit", "verify"}, "", projDir)
	if code != 0 {
		t.Fatalf("audit verify failed: code=%d, err=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Audit log hash chain verified successfully") {
		t.Fatalf("unexpected audit output: %s", stdout)
	}

	// 8. Backup & Restore
	backupFile := filepath.Join(tempHome, "backup.json")
	code, stdout, stderr = executeMayfly(t, []string{"backup", backupFile}, "", projDir)
	if code != 0 {
		t.Fatalf("backup failed: code=%d, err=%s", code, stderr)
	}

	code, stdout, stderr = executeMayfly(t, []string{"restore", backupFile}, "", projDir)
	if code != 0 {
		t.Fatalf("restore failed: code=%d, err=%s", code, stderr)
	}

	// 9. Delete secret
	code, stdout, stderr = executeMayfly(t, []string{"delete", "DATABASE_URL"}, "masterpass\n", projDir)
	if code != 0 {
		t.Fatalf("delete failed: code=%d, err=%s", code, stderr)
	}

	// 10. Import .env file
	envFile := filepath.Join(projDir, ".env")
	mockKey := "sk_" + "live_" + "1234567890"
	if err := os.WriteFile(envFile, []byte("STRIPE_KEY=\""+mockKey+"\"\nexport REDIS_PORT=6379\n# comment\n"), 0644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = executeMayfly(t, []string{"import", envFile}, "masterpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "Imported 2 secrets") {
		t.Fatalf("import failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 10b. Import .env with --delete
	envFile2 := filepath.Join(projDir, ".env.staging")
	if err := os.WriteFile(envFile2, []byte("STAGING_KEY=secret123\n"), 0644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = executeMayfly(t, []string{"import", envFile2, "--delete"}, "masterpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "Deleted plaintext") {
		t.Fatalf("import --delete failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}
	if _, err := os.Stat(envFile2); !os.IsNotExist(err) {
		t.Fatalf("expected .env.staging to be deleted after import --delete")
	}

	// 11. List secrets as JSON
	code, stdout, stderr = executeMayfly(t, []string{"list", "--json"}, "masterpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "STRIPE_KEY") || !strings.Contains(stdout, "REDIS_PORT") {
		t.Fatalf("list --json failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 12. Get secret with --clip
	code, stdout, stderr = executeMayfly(t, []string{"get", "STRIPE_KEY", "--clip"}, "masterpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "copied to clipboard") {
		t.Fatalf("get --clip failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 13. Rotate Master Password
	code, stdout, stderr = executeMayfly(t, []string{"rotate-password"}, "masterpass\nnewsecretpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "rotated successfully") {
		t.Fatalf("rotate-password failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// Verify old password fails
	code, _, _ = executeMayfly(t, []string{"list"}, "masterpass\n", projDir)
	if code == 0 {
		t.Fatalf("expected old password to fail after rotation")
	}

	// Verify new password succeeds
	code, stdout, stderr = executeMayfly(t, []string{"list"}, "newsecretpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "STRIPE_KEY") {
		t.Fatalf("list with new password failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 14. Shell completion
	code, stdout, stderr = executeMayfly(t, []string{"completion", "bash"}, "", projDir)
	if code != 0 || !strings.Contains(stdout, "_mayfly") {
		t.Fatalf("completion bash failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 15. Version flag
	code, stdout, stderr = executeMayfly(t, []string{"version"}, "", projDir)
	if code != 0 || !strings.Contains(stdout, "mayfly v"+domain.Version) {
		t.Fatalf("version failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 16. Help text contains update
	code, stdout, stderr = executeMayfly(t, []string{"help"}, "", projDir)
	if code != 0 || !strings.Contains(stdout, "mayfly update") {
		t.Fatalf("help failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 17. Test 'set --clip'
	code, stdout, stderr = executeMayfly(t, []string{"set", "CLIP_VAR", "--clip", "clipval123"}, "newsecretpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "Secret CLIP_VAR saved") {
		t.Fatalf("set --clip failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 18. Test 'env' in various shells (bash, fish, powershell, json)
	code, stdout, stderr = executeMayfly(t, []string{"env"}, "newsecretpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "export STRIPE_KEY=") {
		t.Fatalf("env bash failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	code, stdout, stderr = executeMayfly(t, []string{"env", "--shell", "fish"}, "newsecretpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "set -x STRIPE_KEY") {
		t.Fatalf("env fish failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	code, stdout, stderr = executeMayfly(t, []string{"env", "--shell", "powershell"}, "newsecretpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "$env:STRIPE_KEY =") {
		t.Fatalf("env powershell failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	code, stdout, stderr = executeMayfly(t, []string{"env", "--shell", "json"}, "newsecretpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "\"STRIPE_KEY\":") {
		t.Fatalf("env json failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	code, _, stderr = executeMayfly(t, []string{"env", "--shell", "invalid_shell"}, "newsecretpass\n", projDir)
	if code == 0 || !strings.Contains(stderr, "unsupported shell") {
		t.Fatalf("expected error on invalid shell: code=%d, err=%s", code, stderr)
	}

	// 19. Test 'status' and 'doctor'
	code, stdout, stderr = executeMayfly(t, []string{"status"}, "", projDir)
	if code != 0 || !strings.Contains(stdout, "MayFly Status") {
		t.Fatalf("status failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	code, stdout, stderr = executeMayfly(t, []string{"doctor"}, "", projDir)
	if code != 0 || !strings.Contains(stdout, "MayFly Status") {
		t.Fatalf("doctor failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 20. Test 'check' (integrity verification)
	code, stdout, stderr = executeMayfly(t, []string{"check"}, "", projDir)
	if code != 0 || !strings.Contains(stdout, "audit log hash chain verified") {
		t.Fatalf("check failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 21. Test 'template' rendering (including values with '{{' syntax)
	tplFile := filepath.Join(projDir, "app.config.template")
	if err := os.WriteFile(tplFile, []byte("key = \"{{ STRIPE_KEY }}\"\nclip = \"{{ CLIP_VAR }}\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	outTplFile := filepath.Join(projDir, "app.config")
	code, stdout, stderr = executeMayfly(t, []string{"template", tplFile, "--output", outTplFile}, "newsecretpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "Rendered template written") {
		t.Fatalf("template failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}
	tplData, err := os.ReadFile(outTplFile)
	if err != nil || !strings.Contains(string(tplData), "key = \"sk_live_1234567890\"") {
		t.Fatalf("template content unexpected: %s", string(tplData))
	}

	// 22. Test 'diff' between two projects
	projDir2 := filepath.Join(t.TempDir(), "test-app-2")
	_ = os.MkdirAll(projDir2, 0755)
	executeMayfly(t, []string{"init", "-path", projDir2}, "", "")
	executeMayfly(t, []string{"set", "STRIPE_KEY", "different_val"}, "newsecretpass\n", projDir2)
	executeMayfly(t, []string{"set", "PROJ2_ONLY_KEY", "val"}, "newsecretpass\n", projDir2)

	// 1-arg diff: current vs projDir2
	code, stdout, stderr = executeMayfly(t, []string{"diff", projDir2}, "newsecretpass\n", projDir)
	if code != 0 || !strings.Contains(stdout, "In both") || !strings.Contains(stdout, "Only in B") {
		t.Fatalf("diff 1-arg failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 2-arg diff: projDir vs projDir2
	code, stdout, stderr = executeMayfly(t, []string{"diff", projDir, projDir2}, "newsecretpass\n", tempHome)
	if code != 0 || !strings.Contains(stdout, "In both") {
		t.Fatalf("diff 2-arg failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 23. Test git hook installation and uninstallation
	gitDir := filepath.Join(projDir, ".git")
	_ = os.MkdirAll(filepath.Join(gitDir, "hooks"), 0755)
	code, stdout, stderr = executeMayfly(t, []string{"install-hook"}, "", projDir)
	if code != 0 || !strings.Contains(stdout, "Pre-commit hook installed") {
		t.Fatalf("install-hook failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	code, stdout, stderr = executeMayfly(t, []string{"uninstall-hook"}, "", projDir)
	if code != 0 || !strings.Contains(stdout, "Pre-commit hook removed") {
		t.Fatalf("uninstall-hook failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 24. Test --password-stdin flag (both before and after subcommand)
	code, stdout, stderr = executeMayfly(t, []string{"--password-stdin", "get", "STRIPE_KEY"}, "newsecretpass\n", projDir)
	if code != 0 || strings.TrimSpace(stdout) != "sk_live_1234567890" {
		t.Fatalf("--password-stdin before subcmd failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	code, stdout, stderr = executeMayfly(t, []string{"get", "STRIPE_KEY", "--password-stdin"}, "newsecretpass\n", projDir)
	if code != 0 || strings.TrimSpace(stdout) != "sk_live_1234567890" {
		t.Fatalf("--password-stdin after subcmd failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 25. Test scan --json (detects remaining .env file with exit code 1)
	code, stdout, stderr = executeMayfly(t, []string{"scan", "--json"}, "", projDir)
	if code != 1 || !strings.Contains(stdout, "plaintext-env-file") {
		t.Fatalf("expected scan --json to detect .env with code 1, got code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// Remove .env and verify clean scan returns code 0 and "[]"
	_ = os.Remove(envFile)
	code, stdout, stderr = executeMayfly(t, []string{"scan", "--json"}, "", projDir)
	if code != 0 || !strings.Contains(stdout, "[]") {
		t.Fatalf("scan --json after cleanup failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}

	// 26. Test audit --json and audit --tail
	code, stdout, stderr = executeMayfly(t, []string{"audit", "--json", "--tail", "5"}, "", projDir)
	if code != 0 || !strings.Contains(stdout, "\"action\":") {
		t.Fatalf("audit --json failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}
}


