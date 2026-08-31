package executor

import (
	"bytes"
	"context"
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
