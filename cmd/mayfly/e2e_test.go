package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mayfly/internal/childdemo"
)

func TestMain(m *testing.M) {
	if os.Getenv("MAYFLY_CHILD_TEST_HELPER") == "1" {
		os.Exit(childdemo.Run(os.Args[1:], os.Stdout))
	}
	os.Exit(m.Run())
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

func TestEndToEndCompleteFlow(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("MAYFLY_CHILD_TEST_HELPER", "1")

	projectDir := filepath.Join(t.TempDir(), "my-cool-app")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 1 & 2: Initialize project
	code, stdout, stderr := executeMayfly(t, []string{"init", "-path", projectDir}, "", "")
	if code != 0 {
		t.Fatalf("mayfly init failed with code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Initialized project") {
		t.Fatalf("unexpected init stdout: %s", stdout)
	}

	// 3: Store a known fake secret
	const (
		masterPass = "SuperSecretMaster123!"
		secretName = "FAKESECRET_KEY_12345"
		secretVal  = "fake_secret_value_xyz987"
	)
	setInput := masterPass + "\n" + secretVal + "\n"
	code, stdout, stderr = executeMayfly(t, []string{"set", secretName}, setInput, projectDir)
	if code != 0 {
		t.Fatalf("mayfly set failed with code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Set "+secretName) {
		t.Fatalf("unexpected set stdout: %s", stdout)
	}
	if strings.Contains(stdout, secretVal) || strings.Contains(stderr, secretVal) {
		t.Fatal("mayfly set leaked the plaintext secret value in stdout/stderr")
	}

	// 4: Verify the encrypted vault file does NOT contain the plaintext secret or password
	vaultPath := filepath.Join(tempHome, ".mayfly", "vault.enc")
	vaultBytes, err := os.ReadFile(vaultPath)
	if err != nil {
		t.Fatalf("failed to read vault file: %v", err)
	}
	if !strings.HasPrefix(string(vaultBytes), "MFVAUL") {
		t.Fatal("vault file missing MFVAUL header magic")
	}
	if strings.Contains(string(vaultBytes), secretVal) {
		t.Fatal("vault file contains plaintext secret value!")
	}
	if strings.Contains(string(vaultBytes), masterPass) {
		t.Fatal("vault file contains plaintext master password!")
	}

	// Verify vault permissions are 0600
	info, err := os.Stat(vaultPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("vault file permissions = %o, want 0600", info.Mode().Perm())
	}

	// Verify list command lists names only
	code, stdout, stderr = executeMayfly(t, []string{"list"}, masterPass+"\n", projectDir)
	if code != 0 {
		t.Fatalf("mayfly list failed with code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, secretName) {
		t.Fatalf("mayfly list missing secret name, stdout: %s", stdout)
	}
	if strings.Contains(stdout, secretVal) {
		t.Fatal("mayfly list leaked plaintext secret value in output")
	}

	// 5, 6, 7: Run demo child process through mayfly run
	// The child is this test binary invoked with helper flag
	childExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = executeMayfly(t, []string{"run", childExecutable, "env", secretName}, masterPass+"\n", projectDir)
	if code != 0 {
		t.Fatalf("mayfly run failed with code %d, stderr: %s", code, stderr)
	}

	// Verify child received the secret
	var envOutput []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &envOutput); err != nil {
		t.Fatalf("failed to parse child JSON output: %v, stdout: %q", err, stdout)
	}
	if len(envOutput) != 1 || envOutput[0] != secretVal {
		t.Fatalf("child received env = %#v, want [%q]", envOutput, secretVal)
	}

	// Verify MayFly output (stderr) does not contain the secret
	if strings.Contains(stderr, secretVal) || strings.Contains(stderr, masterPass) {
		t.Fatal("mayfly stderr leaked secret value or password")
	}

	// 8: Verify audit log contains safe metadata only
	code, stdout, stderr = executeMayfly(t, []string{"audit"}, "", projectDir)
	if code != 0 {
		t.Fatalf("mayfly audit failed with code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "PROJECT_INITIALIZED") ||
		!strings.Contains(stdout, "VAULT_UNLOCKED") ||
		(!strings.Contains(stdout, "SECRET_CREATED") && !strings.Contains(stdout, "SECRET_UPDATED")) ||
		!strings.Contains(stdout, "COMMAND_STARTED") ||
		!strings.Contains(stdout, "SECRET_INJECTED") ||
		!strings.Contains(stdout, "COMMAND_EXITED") {
		t.Fatalf("audit log missing expected events, stdout: %s", stdout)
	}
	if strings.Contains(stdout, secretVal) || strings.Contains(stdout, masterPass) {
		t.Fatal("audit log output contains secret value or password!")
	}

	// Verify raw audit file on disk does not contain plaintext secret value
	auditPath := filepath.Join(tempHome, ".mayfly", "audit.log")
	auditBytes, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(auditBytes), secretVal) || strings.Contains(string(auditBytes), masterPass) {
		t.Fatal("audit log file contains plaintext secret value or master password!")
	}

	// Verify audit log verification succeeds
	code, stdout, stderr = executeMayfly(t, []string{"audit", "verify"}, "", projectDir)
	if code != 0 || !strings.Contains(stdout, "Audit verified") {
		t.Fatalf("audit verify failed with code %d, stdout: %s, stderr: %s", code, stdout, stderr)
	}

	// 9: Modify audit log and verify tampering detection
	corrupted := append([]byte(nil), auditBytes...)
	// Flip a byte in the second line (an event record)
	lineBreak := bytes.IndexByte(corrupted, '\n')
	if lineBreak > 0 && lineBreak+20 < len(corrupted) {
		corrupted[lineBreak+15] ^= 0x55
	}
	if err := os.WriteFile(auditPath, corrupted, 0600); err != nil {
		t.Fatal(err)
	}
	code, _, _ = executeMayfly(t, []string{"audit", "verify"}, "", projectDir)
	if code == 0 {
		t.Fatal("mayfly audit verify succeeded on tampered audit log, expected failure")
	}

	// Restore audit log so subsequent operations succeed
	if err := os.WriteFile(auditPath, auditBytes, 0600); err != nil {
		t.Fatal(err)
	}

	// 10: Create a fake plaintext .env in the project directory
	fakeEnvPath := filepath.Join(projectDir, ".env")
	if err := os.WriteFile(fakeEnvPath, []byte("STRIPE_KEY=sk_test_fake_plaintext_key\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 11: Verify scanner detects it and returns exit code 3
	code, stdout, _ = executeMayfly(t, []string{"scan"}, "", projectDir)
	if code != 3 {
		t.Fatalf("mayfly scan exit code = %d, want 3 for findings", code)
	}
	if !strings.Contains(stdout, ".env") || !strings.Contains(stdout, "high-risk-filename") {
		t.Fatalf("scan output missing .env finding, stdout: %s", stdout)
	}
	if strings.Contains(stdout, "sk_test_fake_plaintext_key") {
		t.Fatal("scan output leaked matched secret value!")
	}
}

func TestSecurityHardeningFailurePaths(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("MAYFLY_CHILD_TEST_HELPER", "1")

	uninitializedDir := t.TempDir()

	// 1. Uninitialized project rejection
	for _, cmd := range [][]string{
		{"set", "TOKEN"},
		{"get", "TOKEN"},
		{"list"},
		{"run", "echo", "hello"},
		{"scan"},
	} {
		code, _, stderr := executeMayfly(t, cmd, "master\nval\n", uninitializedDir)
		if code == 0 {
			t.Fatalf("command %v unexpectedly succeeded in uninitialized directory", cmd)
		}
		if !strings.Contains(stderr, "project is not initialized") {
			t.Fatalf("command %v error = %q, want 'project is not initialized'", cmd, stderr)
		}
	}

	// 2. Initialize project
	projectDirA := filepath.Join(t.TempDir(), "project-a")
	_ = os.MkdirAll(projectDirA, 0755)
	code, _, stderr := executeMayfly(t, []string{"init", "-path", projectDirA}, "", "")
	if code != 0 {
		t.Fatalf("init project A failed: %s", stderr)
	}

	// 3. Set secret in Project A
	code, _, stderr = executeMayfly(t, []string{"set", "SECRET_A"}, "passwordA\nvalueA\n", projectDirA)
	if code != 0 {
		t.Fatalf("set in project A failed: %s", stderr)
	}

	// 4. Wrong password rejection on get/list
	code, stdout, stderr := executeMayfly(t, []string{"get", "SECRET_A"}, "wrong-password\n", projectDirA)
	if code == 0 {
		t.Fatal("get with wrong password unexpectedly succeeded")
	}
	if strings.Contains(stdout, "valueA") || strings.Contains(stderr, "valueA") {
		t.Fatal("wrong password attempt leaked secret value")
	}

	code, stdout, stderr = executeMayfly(t, []string{"list"}, "wrong-password\n", projectDirA)
	if code == 0 {
		t.Fatal("list with wrong password unexpectedly succeeded")
	}
	if strings.Contains(stdout, "SECRET_A") {
		t.Fatal("list with wrong password showed secrets")
	}

	// 5. Invalid secret names with '=' or control characters
	code, _, stderr = executeMayfly(t, []string{"set", "BAD=NAME"}, "passwordA\nval\n", projectDirA)
	if code == 0 || !strings.Contains(stderr, "invalid secret name") {
		t.Fatalf("setting name with '=' should fail, got code %d, stderr: %s", code, stderr)
	}

	// 6. Project isolation test
	projectDirB := filepath.Join(t.TempDir(), "project-b")
	_ = os.MkdirAll(projectDirB, 0755)
	code, _, stderr = executeMayfly(t, []string{"init", "-path", projectDirB}, "", "")
	if code != 0 {
		t.Fatalf("init project B failed: %s", stderr)
	}
	code, _, stderr = executeMayfly(t, []string{"set", "SECRET_B"}, "passwordA\nvalueB\n", projectDirB)
	if code != 0 {
		t.Fatalf("set in project B failed: %s", stderr)
	}

	// List in Project A shows ONLY SECRET_A
	code, stdout, _ = executeMayfly(t, []string{"list"}, "passwordA\n", projectDirA)
	if code != 0 || !strings.Contains(stdout, "SECRET_A") || strings.Contains(stdout, "SECRET_B") {
		t.Fatalf("project A list = %q, want only SECRET_A", stdout)
	}

	// List in Project B shows ONLY SECRET_B
	code, stdout, _ = executeMayfly(t, []string{"list"}, "passwordA\n", projectDirB)
	if code != 0 || !strings.Contains(stdout, "SECRET_B") || strings.Contains(stdout, "SECRET_A") {
		t.Fatalf("project B list = %q, want only SECRET_B", stdout)
	}

	// Child execution in Project A receives ONLY SECRET_A
	childExecutable, _ := os.Executable()
	code, stdout, _ = executeMayfly(t, []string{"run", childExecutable, "env", "SECRET_A", "SECRET_B"}, "passwordA\n", projectDirA)
	if code != 0 {
		t.Fatalf("run in project A failed: %s", stdout)
	}
	var envA []string
	_ = json.Unmarshal([]byte(strings.TrimSpace(stdout)), &envA)
	if len(envA) != 2 || envA[0] != "valueA" || envA[1] != "" {
		t.Fatalf("project A child env = %#v, want ['valueA', '']", envA)
	}

	// 7. Child process exit code propagation
	code, _, _ = executeMayfly(t, []string{"run", childExecutable, "exit", "42"}, "passwordA\n", projectDirA)
	if code != 42 {
		t.Fatalf("run exit code = %d, want 42", code)
	}
}
