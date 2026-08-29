package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"mayfly/domain"
)

type fakeSecrets struct {
	list      []domain.Secret
	materials map[domain.SecretName]domain.SecretMaterial
	puts      []domain.SecretInput
	deletes   []domain.SecretName
}

func (f *fakeSecrets) List(context.Context, domain.ProjectID) ([]domain.Secret, error) {
	return append([]domain.Secret(nil), f.list...), nil
}
func (f *fakeSecrets) Get(_ context.Context, _ domain.ProjectID, name domain.SecretName) (domain.SecretMaterial, error) {
	material, ok := f.materials[name]
	if !ok {
		return domain.SecretMaterial{}, errors.New("missing secret")
	}
	return material, nil
}
func (f *fakeSecrets) Put(_ context.Context, input domain.SecretInput) error {
	f.puts = append(f.puts, input)
	return nil
}
func (f *fakeSecrets) Delete(_ context.Context, _ domain.ProjectID, name domain.SecretName) error {
	f.deletes = append(f.deletes, name)
	return nil
}

type fakeExecutor struct {
	request domain.ExecutionRequest
	env     Environment
}

func (f *fakeExecutor) Execute(_ context.Context, request domain.ExecutionRequest, environment Environment) (ExecutionResult, error) {
	f.request = request
	f.env = append(Environment(nil), environment...)
	return ExecutionResult{ExitCode: 7}, nil
}

type fakeAudit struct{ events []domain.AuditEvent }

func (f *fakeAudit) Record(_ context.Context, event domain.AuditEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	f.events = append(f.events, event)
	return nil
}

type fakeProjects struct{ project domain.Project }

func (f fakeProjects) Current(context.Context) (domain.Project, error) { return f.project, nil }
func (f fakeProjects) Get(_ context.Context, id domain.ProjectID) (domain.Project, error) {
	if id != f.project.ID {
		return domain.Project{}, errors.New("project not found")
	}
	return f.project, nil
}

type fakeVault struct{ secrets SecretService }

func (f fakeVault) Open(context.Context, []byte) (SecretService, error) { return f.secrets, nil }

func TestServiceCanBeConstructedWithHandWrittenFakes(t *testing.T) {
	secrets := &fakeSecrets{}
	service := NewService(Dependencies{Secrets: secrets})
	input := domain.SecretInput{ProjectID: "project-1", Name: "TOKEN", Value: "value"}
	if err := service.SaveSecret(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(secrets.puts, []domain.SecretInput{input}) {
		t.Fatalf("puts = %#v", secrets.puts)
	}
	if _, err := service.ListSecrets(context.Background(), "project-1"); err != nil {
		t.Fatal(err)
	}
}

func TestRunLoadsSelectedSecretsAndAuditsWithoutValues(t *testing.T) {
	secrets := &fakeSecrets{
		list: []domain.Secret{{ProjectID: "project-1", Name: "TOKEN"}},
		materials: map[domain.SecretName]domain.SecretMaterial{
			"TOKEN": {Name: "TOKEN", Value: "do-not-log"},
		},
	}
	executor := &fakeExecutor{}
	auditor := &fakeAudit{}
	service := NewService(Dependencies{
		Projects: fakeProjects{project: domain.Project{ID: "project-1", Name: "Demo"}},
		Secrets:  secrets,
		Executor: executor,
		Auditor:  auditor,
	})
	request := domain.ExecutionRequest{ProjectID: "project-1", Command: []string{"demo"}, SecretNames: []domain.SecretName{"TOKEN"}}
	result, err := service.Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 7 || len(executor.env) != 1 || executor.env[0].Value != "do-not-log" {
		t.Fatalf("run result/environment = %#v/%#v", result, executor.env)
	}
	if len(auditor.events) != 1 || auditor.events[0].Secret != "" {
		t.Fatalf("audit events = %#v", auditor.events)
	}
}

func TestRunRejectsInvalidRequestBeforeCallingDependencies(t *testing.T) {
	executor := &fakeExecutor{}
	service := NewService(Dependencies{Secrets: &fakeSecrets{}, Executor: executor})
	if _, err := service.Run(context.Background(), domain.ExecutionRequest{Command: []string{"demo"}}); err == nil {
		t.Fatal("invalid execution request was accepted")
	}
	if executor.request.Command != nil {
		t.Fatal("executor was called for invalid request")
	}
}

func TestRunRequiresProjectLookup(t *testing.T) {
	service := NewService(Dependencies{Secrets: &fakeSecrets{}, Executor: &fakeExecutor{}})
	request := domain.ExecutionRequest{ProjectID: "project-1", Command: []string{"demo"}}
	if _, err := service.Run(context.Background(), request); !errors.Is(err, ErrMissingProject) {
		t.Fatalf("missing project lookup error = %v", err)
	}
}

func TestOpenVaultUsesExplicitStorageBoundary(t *testing.T) {
	secrets := &fakeSecrets{}
	service := NewService(Dependencies{Vault: fakeVault{secrets: secrets}})
	got, err := service.OpenVault(context.Background(), []byte("password"))
	if err != nil || got == nil {
		t.Fatalf("OpenVault = %v, %v", got, err)
	}
	if _, err := got.ListSecrets(context.Background(), "project-1"); err != nil {
		t.Fatalf("opened service did not bind secret service: %v", err)
	}
	if _, err := NewService(Dependencies{}).OpenVault(context.Background(), nil); !errors.Is(err, ErrMissingVaultStorage) {
		t.Fatalf("missing vault error = %v", err)
	}
}
