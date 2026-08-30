package executor

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"mayfly/application"
	"mayfly/domain"
	"mayfly/internal/childdemo"
)

func TestMain(main *testing.M) {
	if os.Getenv("MAYFLY_EXECUTOR_CHILD") == "1" {
		os.Exit(runTestChild())
	}
	os.Exit(main.Run())
}

func runTestChild() int {
	mode := os.Getenv("MAYFLY_EXECUTOR_CHILD_MODE")
	switch mode {
	case "env":
		return childdemo.Run([]string{"env", "MAYFLY_TEST_SECRET", "MAYFLY_TEST_PARENT", "MAYFLY_TEST_EMPTY"}, os.Stdout)
	case "args":
		args := []string{"args"}
		for _, argument := range os.Args[1:] {
			if strings.HasPrefix(argument, "-test.run=TestProcessHelper/") {
				args = append(args, strings.TrimPrefix(argument, "-test.run=TestProcessHelper/"))
			}
			if strings.HasPrefix(argument, "-test.bench=TestProcessHelper/") {
				args = append(args, strings.TrimPrefix(argument, "-test.bench=TestProcessHelper/"))
			}
		}
		return childdemo.Run(args, os.Stdout)
	case "exit":
		return childdemo.Run([]string{"exit", os.Getenv("MAYFLY_EXECUTOR_CHILD_EXIT")}, os.Stdout)
	default:
		return 2
	}
}

func processRequest(command ...string) domain.ExecutionRequest {
	return domain.ExecutionRequest{ProjectID: "project-1", Command: command}
}

func childCommand(t *testing.T, extra ...string) []string {
	t.Helper()
	args := []string{os.Args[0], "-test.run=TestProcessHelper/TestProcessHelper"}
	if len(extra) > 0 {
		args[1] = "-test.run=TestProcessHelper/" + extra[0]
	}
	if len(extra) > 1 {
		args = append(args, "-test.bench=TestProcessHelper/"+extra[1])
	}
	return args
}

func TestProcessExecutorInjectsSecretsAndPreservesParentEnvironment(t *testing.T) {
	const parentValue = "ordinary-parent-value"
	t.Setenv("MAYFLY_TEST_PARENT", parentValue)
	var output, stderr strings.Builder
	executor := NewProcessExecutor(strings.NewReader(""), &output, &stderr)
	// The helper process is the current test binary. Its environment is
	// controlled by ProcessExecutor itself, so use the executable path directly
	// and pass the helper flags as the request command.
	request := domain.ExecutionRequest{ProjectID: "project-1", Command: childCommand(t)}
	// The executor inherits the test process environment. The helper marker is
	// explicitly supplied as an environment entry so it is not a hidden global
	// process mode in production.
	environment := application.Environment{
		{Name: "MAYFLY_EXECUTOR_CHILD", Value: "1"},
		{Name: "MAYFLY_EXECUTOR_CHILD_MODE", Value: "env"},
		{Name: "MAYFLY_TEST_SECRET", Value: "secret value\nwith-specials"},
		{Name: "MAYFLY_TEST_EMPTY", Value: ""},
	}
	result, err := executor.Execute(context.Background(), request, environment)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("Execute = %#v, %v; stderr=%q", result, err, stderr.String())
	}
	var values []string
	if err := json.Unmarshal([]byte(output.String()), &values); err != nil {
		t.Fatalf("child output = %q: %v", output.String(), err)
	}
	if !reflect.DeepEqual(values, []string{"secret value\nwith-specials", parentValue, ""}) {
		t.Fatalf("child environment = %#v", values)
	}
	if strings.Contains(stderr.String(), "secret value") {
		t.Fatal("executor wrote secret to stderr")
	}
}

func TestProcessExecutorExplicitSecretWinsCollision(t *testing.T) {
	t.Setenv("MAYFLY_TEST_PARENT", "inherited")
	merged, err := mergeEnvironment([]string{"MAYFLY_TEST_PARENT=inherited", "KEEP=one"}, application.Environment{{Name: "MAYFLY_TEST_PARENT", Value: "explicit"}})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(merged, []string{"MAYFLY_TEST_PARENT=inherited", "KEEP=one"}) {
		t.Fatal("collision was not overridden")
	}
	if got := strings.Join(merged, "\n"); !strings.Contains(got, "MAYFLY_TEST_PARENT=explicit") || strings.Contains(got, "MAYFLY_TEST_PARENT=inherited") {
		t.Fatalf("merged environment = %q", got)
	}
}

func TestProcessExecutorPreservesArgumentBoundaries(t *testing.T) {
	var output strings.Builder
	executor := NewProcessExecutor(strings.NewReader(""), &output, &strings.Builder{})
	request := domain.ExecutionRequest{ProjectID: "project-1", Command: childCommand(t, "hello world", "ユニコード", "")}
	// Two test flags are accepted by the Go test binary and remain distinct
	// os.Args entries in the child helper.
	result, err := executor.Execute(context.Background(), request, application.Environment{
		{Name: "MAYFLY_EXECUTOR_CHILD", Value: "1"},
		{Name: "MAYFLY_EXECUTOR_CHILD_MODE", Value: "args"},
	})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("Execute = %#v, %v", result, err)
	}
	var args []string
	if err := json.Unmarshal([]byte(output.String()), &args); err != nil {
		t.Fatalf("child args output = %q: %v", output.String(), err)
	}
	if !reflect.DeepEqual(args, []string{"hello world", "ユニコード"}) {
		t.Fatalf("child args = %#v", args)
	}
}

func TestProcessExecutorPropagatesExitStatus(t *testing.T) {
	executor := NewProcessExecutor(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})
	request := domain.ExecutionRequest{ProjectID: "project-1", Command: childCommand(t)}
	result, err := executor.Execute(context.Background(), request, application.Environment{
		{Name: "MAYFLY_EXECUTOR_CHILD", Value: "1"},
		{Name: "MAYFLY_EXECUTOR_CHILD_MODE", Value: "exit"},
		{Name: "MAYFLY_EXECUTOR_CHILD_EXIT", Value: "23"},
	})
	if err == nil || result.ExitCode != 23 {
		t.Fatalf("exit result = %#v, %v", result, err)
	}
}

func TestProcessExecutorDoesNotInvokeShell(t *testing.T) {
	var output strings.Builder
	executor := NewProcessExecutor(strings.NewReader(""), &output, &strings.Builder{})
	request := domain.ExecutionRequest{ProjectID: "project-1", Command: []string{"definitely-not-a-real-command; echo forged"}}
	if _, err := executor.Execute(context.Background(), request, nil); err == nil {
		t.Fatal("shell-like command unexpectedly executed")
	}
	if output.Len() != 0 {
		t.Fatalf("unexpected child output = %q", output.String())
	}
}

func TestProcessExecutorDoesNotCreateProjectFiles(t *testing.T) {
	projectDir := t.TempDir()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(workingDirectory) }()
	before, err := os.ReadDir(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	executor := NewProcessExecutor(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})
	request := domain.ExecutionRequest{ProjectID: "project-1", Command: childCommand(t)}
	_, err = executor.Execute(context.Background(), request, application.Environment{
		{Name: "MAYFLY_EXECUTOR_CHILD", Value: "1"},
		{Name: "MAYFLY_EXECUTOR_CHILD_MODE", Value: "exit"},
		{Name: "MAYFLY_EXECUTOR_CHILD_EXIT", Value: "0"},
		{Name: "MAYFLY_SECRET_FOR_FILE_CHECK", Value: "secret-value"},
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadDir(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatalf("executor created project files: before=%v after=%v", before, after)
	}
}

func TestProcessExecutorRejectsInvalidEnvironmentWithoutSecretInError(t *testing.T) {
	secret := "secret-value-must-not-appear"
	executor := NewProcessExecutor(nil, &strings.Builder{}, &strings.Builder{})
	_, err := executor.Execute(context.Background(), processRequest("true"), application.Environment{{Name: "BAD=NAME", Value: secret}})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("invalid environment error = %v", err)
	}
}

func TestProcessHelper(t *testing.T) {}
