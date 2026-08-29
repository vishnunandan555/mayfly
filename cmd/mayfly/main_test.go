package main

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"mayfly/application"
	"mayfly/domain"
)

type commandTestProjects struct{ project domain.Project }

func (p commandTestProjects) Current(context.Context) (domain.Project, error) { return p.project, nil }
func (p commandTestProjects) Get(context.Context, domain.ProjectID) (domain.Project, error) {
	return p.project, nil
}
func (p commandTestProjects) Discover(context.Context, string) (domain.Project, error) {
	return p.project, nil
}

type commandTestSecrets struct {
	material domain.SecretMaterial
	puts     []domain.SecretInput
	deletes  []domain.SecretName
}

func (s *commandTestSecrets) List(context.Context, domain.ProjectID) ([]domain.Secret, error) {
	if s.material.Name == "" {
		return nil, nil
	}
	return []domain.Secret{{Name: s.material.Name}}, nil
}
func (s *commandTestSecrets) Get(context.Context, domain.ProjectID, domain.SecretName) (domain.SecretMaterial, error) {
	if s.material.Name == "" {
		return domain.SecretMaterial{}, errors.New("secret not found")
	}
	return s.material, nil
}
func (s *commandTestSecrets) Put(_ context.Context, input domain.SecretInput) error {
	s.puts = append(s.puts, input)
	s.material = domain.SecretMaterial{Name: input.Name, Value: input.Value}
	return nil
}
func (s *commandTestSecrets) Delete(_ context.Context, _ domain.ProjectID, name domain.SecretName) error {
	s.deletes = append(s.deletes, name)
	s.material = domain.SecretMaterial{}
	return nil
}

type commandTestVault struct{ secrets application.SecretService }

func (v commandTestVault) Open(context.Context, []byte) (application.SecretService, error) {
	return v.secrets, nil
}

type commandTestExecutor struct {
	request domain.ExecutionRequest
	env     application.Environment
}

func (e *commandTestExecutor) Execute(_ context.Context, request domain.ExecutionRequest, environment application.Environment) (application.ExecutionResult, error) {
	e.request = request
	e.env = append(application.Environment(nil), environment...)
	return application.ExecutionResult{ExitCode: 0}, nil
}

func newCommandTestRuntime(secrets application.SecretService) *commandRuntime {
	executor := &commandTestExecutor{}
	return &commandRuntime{
		service: application.NewService(application.Dependencies{
			Projects: commandTestProjects{project: domain.Project{ID: "project-1", Name: "Demo"}},
			Vault:    commandTestVault{secrets: secrets},
			Executor: executor,
		}),
	}
}

func TestExecuteSetUsesBufferedInputAndDoesNotPrintValue(t *testing.T) {
	secrets := &commandTestSecrets{}
	runtime := newCommandTestRuntime(secrets)
	var output, errorOutput strings.Builder
	_, err := runtime.execute(context.Background(), []string{"set", "TOKEN"}, strings.NewReader("master\nsecret-value\n"), &output, &errorOutput)
	if err != nil {
		t.Fatal(err)
	}
	if len(secrets.puts) != 1 || secrets.puts[0].Value != "secret-value" {
		t.Fatalf("puts = %#v", secrets.puts)
	}
	if strings.Contains(output.String(), "secret-value") || strings.Contains(errorOutput.String(), "secret-value") {
		t.Fatal("set output leaked the secret value")
	}
	if !strings.Contains(output.String(), "Set TOKEN") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestExecuteGetIsExplicitValueOutput(t *testing.T) {
	runtime := newCommandTestRuntime(&commandTestSecrets{material: domain.SecretMaterial{Name: "TOKEN", Value: "secret-value"}})
	var output, errorOutput strings.Builder
	if _, err := runtime.execute(context.Background(), []string{"get", "TOKEN"}, strings.NewReader("master\n"), &output, &errorOutput); err != nil {
		t.Fatal(err)
	}
	if output.String() != "secret-value\n" {
		t.Fatalf("get output = %q", output.String())
	}
	if errorOutput.String() != "Vault password: " {
		t.Fatalf("prompt output = %q", errorOutput.String())
	}
}

func TestExecuteListNamesOnlyAndDeleteConfirmation(t *testing.T) {
	secrets := &commandTestSecrets{material: domain.SecretMaterial{Name: "TOKEN", Value: "secret-value"}}
	runtime := newCommandTestRuntime(secrets)
	var listOutput, listErrors strings.Builder
	if _, err := runtime.execute(context.Background(), []string{"list"}, strings.NewReader("master\n"), &listOutput, &listErrors); err != nil {
		t.Fatal(err)
	}
	if listOutput.String() != "TOKEN\n" || strings.Contains(listOutput.String(), "secret-value") {
		t.Fatalf("list output = %q", listOutput.String())
	}
	var deleteOutput, deleteErrors strings.Builder
	if _, err := runtime.execute(context.Background(), []string{"delete", "TOKEN"}, strings.NewReader("master\nn\n"), &deleteOutput, &deleteErrors); err != nil {
		t.Fatal(err)
	}
	if len(secrets.deletes) != 0 || deleteOutput.String() != "Delete cancelled\n" {
		t.Fatalf("delete cancellation = deletes %#v output %q", secrets.deletes, deleteOutput.String())
	}
}

func TestExecuteRunDispatchesExactCommandAndDoesNotPrintEnvironment(t *testing.T) {
	secrets := &commandTestSecrets{material: domain.SecretMaterial{Name: "TOKEN", Value: "secret-value"}}
	executor := &commandTestExecutor{}
	runtime := &commandRuntime{service: application.NewService(application.Dependencies{
		Projects: commandTestProjects{project: domain.Project{ID: "project-1", Name: "Demo"}},
		Vault:    commandTestVault{secrets: secrets},
		Executor: executor,
	})}
	var output, errorOutput strings.Builder
	result, err := runtime.execute(context.Background(), []string{"run", "program", "argument with spaces", "ユニコード"}, strings.NewReader("master\n"), &output, &errorOutput)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("run = %#v, %v", result, err)
	}
	if !reflect.DeepEqual(executor.request.Command, []string{"program", "argument with spaces", "ユニコード"}) {
		t.Fatalf("command = %#v", executor.request.Command)
	}
	if len(executor.env) != 1 || executor.env[0].Value != "secret-value" {
		t.Fatalf("environment = %#v", executor.env)
	}
	if strings.Contains(output.String(), "secret-value") || strings.Contains(errorOutput.String(), "secret-value") {
		t.Fatal("run output leaked secret")
	}
}

func TestRunCommandValidationReturnsUsageExitCode(t *testing.T) {
	var output, errorOutput strings.Builder
	if got := run([]string{"get"}, strings.NewReader(""), &output, &errorOutput); got != 2 {
		t.Fatalf("run exit code = %d, want 2 for command validation", got)
	}
	if strings.Contains(errorOutput.String(), "secret-value") {
		t.Fatal("validation output leaked a secret")
	}
}
