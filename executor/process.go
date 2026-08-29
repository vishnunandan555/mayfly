// Package executor launches MayFly child processes.
//
// Commands are invoked directly through os/exec. No shell is inserted, no
// dotenv or temporary environment file is created, and environment values are
// never included in executor errors.
package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"mayfly/application"
	"mayfly/domain"
)

// ProcessExecutor runs commands with injectable standard streams. Nil streams
// select the process's current stdin, stdout, or stderr at construction time.
type ProcessExecutor struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// NewProcessExecutor creates a process executor. The standard streams are
// injected so tests and embedding applications do not need a real terminal.
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
	return &ProcessExecutor{stdin: stdin, stdout: stdout, stderr: stderr}
}

// Execute invokes request.Command directly and passes the resulting
// environment to the child. Parent variables are retained in their original
// order except for names overridden by an explicit MayFly secret. Secret
// entries are then appended, so the explicit vault value wins collisions.
func (e *ProcessExecutor) Execute(ctx context.Context, request domain.ExecutionRequest, environment application.Environment) (application.ExecutionResult, error) {
	if err := request.Validate(); err != nil {
		return application.ExecutionResult{}, err
	}
	merged, err := mergeEnvironment(os.Environ(), environment)
	if err != nil {
		return application.ExecutionResult{}, err
	}
	defer clearEnvironment(merged)

	command := exec.CommandContext(ctx, request.Command[0], request.Command[1:]...)
	command.Env = merged
	command.Stdin = e.input()
	command.Stdout = e.output()
	command.Stderr = e.errorOutput()
	defer func() { command.Env = nil }()

	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return application.ExecutionResult{ExitCode: processExitCode(exitError.ProcessState)}, err
		}
		return application.ExecutionResult{}, fmt.Errorf("executor: start or wait for command: %w", err)
	}
	return application.ExecutionResult{ExitCode: 0}, nil
}

func (e *ProcessExecutor) input() io.Reader {
	if e == nil || e.stdin == nil {
		return os.Stdin
	}
	return e.stdin
}

func (e *ProcessExecutor) output() io.Writer {
	if e == nil || e.stdout == nil {
		return os.Stdout
	}
	return e.stdout
}

func (e *ProcessExecutor) errorOutput() io.Writer {
	if e == nil || e.stderr == nil {
		return os.Stderr
	}
	return e.stderr
}

func mergeEnvironment(parent []string, secrets application.Environment) ([]string, error) {
	overrides := make(map[string]string, len(secrets))
	order := make([]string, 0, len(secrets))
	for _, entry := range secrets {
		if err := validateEnvironmentEntry(entry); err != nil {
			return nil, err
		}
		if _, exists := overrides[entry.Name]; !exists {
			order = append(order, entry.Name)
		}
		overrides[entry.Name] = entry.Value
	}

	merged := make([]string, 0, len(parent)+len(order))
	for _, item := range parent {
		name, _, hasValue := strings.Cut(item, "=")
		if hasValue {
			if _, overridden := overrides[name]; overridden {
				continue
			}
		}
		merged = append(merged, item)
	}
	for _, name := range order {
		merged = append(merged, name+"="+overrides[name])
	}
	return merged, nil
}

func validateEnvironmentEntry(entry application.EnvironmentEntry) error {
	if strings.TrimSpace(entry.Name) == "" || strings.ContainsRune(entry.Name, '=') || strings.ContainsRune(entry.Name, '\x00') || !utf8.ValidString(entry.Name) {
		return errors.New("executor: invalid environment name")
	}
	if strings.ContainsRune(entry.Value, '\x00') || !utf8.ValidString(entry.Value) {
		return errors.New("executor: invalid environment value")
	}
	return nil
}

func clearEnvironment(environment []string) {
	for index := range environment {
		environment[index] = ""
	}
}
