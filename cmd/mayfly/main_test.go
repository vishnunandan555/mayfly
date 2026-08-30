package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	// 5. Run command with injected secret
	code, stdout, stderr = executeMayfly(t, []string{"run", "sh", "-c", "echo DB=$DATABASE_URL"}, "masterpass\n", projDir)
	if code != 0 {
		t.Fatalf("run failed: code=%d, err=%s", code, stderr)
	}
	if !strings.Contains(stdout, "DB=postgres://localhost/db") {
		t.Fatalf("unexpected run output: %s", stdout)
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

	// 10. Version flag
	code, stdout, stderr = executeMayfly(t, []string{"version"}, "", projDir)
	if code != 0 || !strings.Contains(stdout, "mayfly v1.0.0") {
		t.Fatalf("version failed: code=%d, err=%s, out=%s", code, stderr, stdout)
	}
}
