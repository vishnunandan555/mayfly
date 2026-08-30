package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"mayfly/pkg/domain"
)

var ErrNoCommandProvided = errors.New("executor: command cannot be empty")

// ProcessExecutor executes child processes directly via os/exec with in-memory environment overlay.
type ProcessExecutor struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func NewProcessExecutor(stdin io.Reader, stdout, stderr io.Writer) *ProcessExecutor {
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	return &ProcessExecutor{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
}

// Execute runs the requested command with secrets overlaid onto its environment in RAM.
func (e *ProcessExecutor) Execute(ctx context.Context, req domain.ExecutionRequest, secrets map[domain.SecretName]string) (domain.ExecutionResult, error) {
	if len(req.Command) == 0 || strings.TrimSpace(req.Command[0]) == "" {
		return domain.ExecutionResult{ExitCode: 1}, ErrNoCommandProvided
	}

	cmdName := req.Command[0]
	cmdArgs := req.Command[1:]

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Stdin = e.stdin
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr
	if req.Dir != "" {
		cmd.Dir = req.Dir
	}

	// Build in-memory environment overlay
	envMap := make(map[string]string)
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Overlay project secrets
	for k, v := range secrets {
		envMap[string(k)] = v
	}

	envSlice := make([]string, 0, len(envMap))
	for k, v := range envMap {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = envSlice

	err := cmd.Run()

	// Clear memory references
	for i := range envSlice {
		envSlice[i] = ""
	}
	runtime.KeepAlive(envSlice)

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return domain.ExecutionResult{ExitCode: extractExitCode(exitErr)}, nil
		}
		return domain.ExecutionResult{ExitCode: 1}, err
	}

	return domain.ExecutionResult{ExitCode: 0}, nil
}
