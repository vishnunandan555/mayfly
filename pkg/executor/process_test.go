package executor

import (
	"bytes"
	"context"
	"os"
	"os/exec"
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

func TestProcessExecutorEnvironmentOverlay(t *testing.T) {
	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	exec := NewProcessExecutor(nil, &stdout, &stderr)

	secrets := map[domain.SecretName]string{
		"MAYFLY_TEST_SECRET": "hello_from_ram",
	}

	cmdScript := "echo VAL=$MAYFLY_TEST_SECRET"
	if runtime.GOOS == "windows" {
		cmdScript = "echo VAL=%MAYFLY_TEST_SECRET%"
	}
	req := domain.ExecutionRequest{
		Command: shellCmd(cmdScript),
	}

	res, err := exec.Execute(ctx, req, secrets)
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", res.ExitCode)
	}

	if !strings.Contains(stdout.String(), "VAL=hello_from_ram") {
		t.Fatalf("expected injected secret in stdout, got: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "MAYFLY_TEST_SECRET") {
		t.Fatalf("expected injection notification in stderr, got: %s", stderr.String())
	}
}

func TestTerminateChildInterruptKillsWindowsChild(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only signal regression test")
	}

	cmd := exec.Command("ping", "-n", "30", "127.0.0.1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start ping: %v", err)
	}

	if err := terminateChild(cmd, os.Interrupt); err != nil && !strings.Contains(err.Error(), "process already finished") {
		t.Fatalf("terminateChild returned unexpected error: %v", err)
	}

	if err := cmd.Wait(); err == nil {
		if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
			t.Fatal("expected ping process to be terminated by interrupt")
		}
	}
}
