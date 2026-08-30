package executor

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"mayfly/pkg/domain"
)

func TestProcessExecutorEnvironmentOverlay(t *testing.T) {
	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	exec := NewProcessExecutor(nil, &stdout, &stderr)

	secrets := map[domain.SecretName]string{
		"MAYFLY_TEST_SECRET": "hello_from_ram",
	}

	req := domain.ExecutionRequest{
		Command: []string{"sh", "-c", "echo VAL=$MAYFLY_TEST_SECRET"},
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
}
