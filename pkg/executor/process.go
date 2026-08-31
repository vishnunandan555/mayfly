package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"syscall"

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
	configureProcess(cmd)
	cmd.Stdin = e.stdin
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr
	if req.Dir != "" {
		cmd.Dir = req.Dir
	}

	// Inform user of in-memory injection status on stderr
	if len(secrets) > 0 {
		keys := make([]string, 0, len(secrets))
		for k := range secrets {
			keys = append(keys, string(k))
		}
		sort.Strings(keys)
		fmt.Fprintf(e.stderr, "[mayfly] Injected %d secret(s) into process RAM [%s]\n", len(secrets), strings.Join(keys, ", "))
	} else {
		fmt.Fprintf(e.stderr, "[mayfly] No secrets configured for this project\n")
	}

	// Build in-memory environment overlay
	envMap := make(map[string]string)
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Overlay project secrets into RAM table
	for k, v := range secrets {
		envMap[string(k)] = v
	}

	envSlice := make([]string, 0, len(envMap))
	for k, v := range envMap {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = envSlice

	// Deferred memory wipe to prevent credentials lingering in process heap
	defer func() {
		for i := range envSlice {
			envSlice[i] = ""
		}
		for k := range envMap {
			delete(envMap, k)
		}
		runtime.GC()
	}()

	// Signal forwarding channel to guarantee child process terminates on Ctrl+C / SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	if err := cmd.Start(); err != nil {
		return domain.ExecutionResult{ExitCode: 1}, err
	}

	// Forward kill/interrupt signals to the child process
	doneChan := make(chan struct{})
	go func() {
		select {
		case sig, ok := <-sigChan:
			if ok && cmd.Process != nil {
				_ = terminateChild(cmd, sig)
			}
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		case <-doneChan:
		}
	}()

	err := cmd.Wait()
	close(doneChan)

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return domain.ExecutionResult{ExitCode: extractExitCode(exitErr)}, nil
		}
		return domain.ExecutionResult{ExitCode: 1}, err
	}

	return domain.ExecutionResult{ExitCode: 0}, nil
}
