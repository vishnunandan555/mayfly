package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"mayfly/domain"
)

type fakeSecrets struct {
	list      []domain.Secret
	materials map[domain.SecretName]domain.SecretMaterial
	puts      []domain.SecretInput
	deletes   []domain.SecretName
	putErr    error
	deleteErr error
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
	if f.putErr != nil {
		return f.putErr
	}
	f.puts = append(f.puts, input)
	return nil
}
func (f *fakeSecrets) Delete(_ context.Context, _ domain.ProjectID, name domain.SecretName) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
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

type fakeAudit struct {
	events []domain.AuditEvent
	err    error
}

func (f *fakeAudit) Record(_ context.Context, event domain.AuditEvent) error {
	if f.err != nil {
		return f.err
	}
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
func (f fakeProjects) Discover(context.Context, string) (domain.Project, error) {
	return f.project, nil
}

type fakeVault struct{ secrets SecretService }

func (f fakeVault) Open(context.Context, []byte) (SecretService, error) { return f.secrets, nil }

type fakeScanner struct{ findings []domain.ScanFinding }

func (f fakeScanner) Scan(context.Context, domain.Project) ([]domain.ScanFinding, error) {
	return append([]domain.ScanFinding(nil), f.findings...), nil
}

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
	if len(auditor.events) != 3 || auditor.events[0].Action != domain.AuditCommandStarted || auditor.events[0].Secret != "" || auditor.events[1].Action != domain.AuditSecretInjected || auditor.events[1].Secret != "TOKEN" || auditor.events[2].Action != domain.AuditCommandExited || auditor.events[2].ExitStatus == nil || *auditor.events[2].ExitStatus != 7 {
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
	defer got.Close()
	if _, err := got.ListSecrets(context.Background(), "project-1"); err != nil {
		t.Fatalf("opened service did not bind secret service: %v", err)
	}
	if _, err := NewService(Dependencies{}).OpenVault(context.Background(), nil); !errors.Is(err, ErrMissingVaultStorage) {
		t.Fatalf("missing vault error = %v", err)
	}
}

func TestServiceCurrentSecretCRUDAndAudit(t *testing.T) {
	project := domain.Project{ID: "project-1", Name: "Demo"}
	secrets := &fakeSecrets{materials: map[domain.SecretName]domain.SecretMaterial{
		"TOKEN": {Name: "TOKEN", Value: "value"},
	}}
	auditor := &fakeAudit{}
	service := NewService(Dependencies{
		Projects: fakeProjects{project: project},
		Secrets:  secrets,
		Auditor:  auditor,
	})

	if err := service.SetCurrentSecret(context.Background(), "TOKEN", "new-value"); err != nil {
		t.Fatal(err)
	}
	if got := secrets.puts; len(got) != 1 || got[0].ProjectID != project.ID || got[0].Name != "TOKEN" || got[0].Value != "new-value" {
		t.Fatalf("set inputs = %#v", got)
	}

	material, err := service.GetCurrentSecret(context.Background(), "TOKEN")
	if err != nil || material.Value != "value" {
		t.Fatalf("get = %#v, %v", material, err)
	}
	if err := service.DeleteCurrentSecret(context.Background(), "TOKEN"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(secrets.deletes, []domain.SecretName{"TOKEN"}) {
		t.Fatalf("deletes = %#v", secrets.deletes)
	}
	if got := len(auditor.events); got != 3 {
		t.Fatalf("audit event count = %d, want 3", got)
	}
	if auditor.events[0].Action != domain.AuditSecretWritten || auditor.events[1].Action != domain.AuditSecretRead || auditor.events[2].Action != domain.AuditSecretDeleted {
		t.Fatalf("audit events = %#v", auditor.events)
	}
}

func TestServiceAuditFailureIsStableAndDoesNotLeakValue(t *testing.T) {
	secrets := &fakeSecrets{}
	service := NewService(Dependencies{
		Projects: fakeProjects{project: domain.Project{ID: "project-1", Name: "Demo"}},
		Secrets:  secrets,
		Auditor:  &fakeAudit{err: errors.New("audit failed: secret-value-must-not-escape")},
	})
	if err := service.SetCurrentSecret(context.Background(), "TOKEN", "secret-value-must-not-escape"); !errors.Is(err, ErrAuditFailed) {
		t.Fatalf("set audit error = %v", err)
	} else if strings.Contains(err.Error(), "secret-value-must-not-escape") {
		t.Fatalf("audit error leaked secret: %v", err)
	}
	if len(secrets.puts) != 1 {
		t.Fatalf("successful backend transition was not recorded: %#v", secrets.puts)
	}
}

func TestServiceValidatesNamesBeforeSecretBackend(t *testing.T) {
	secrets := &fakeSecrets{}
	service := NewService(Dependencies{
		Projects: fakeProjects{project: domain.Project{ID: "project-1", Name: "Demo"}},
		Secrets:  secrets,
	})
	if err := service.SetCurrentSecret(context.Background(), "bad=name", "value"); !errors.Is(err, ErrInvalidSecretName) {
		t.Fatalf("invalid name error = %v", err)
	}
	if len(secrets.puts) != 0 {
		t.Fatal("invalid name reached secret backend")
	}
	if err := NewService(Dependencies{Secrets: secrets}).SetCurrentSecret(context.Background(), "TOKEN", "value"); !errors.Is(err, ErrMissingProject) {
		t.Fatalf("missing project error = %v", err)
	}
}

func TestServiceScansCurrentProjectAndAuditsCompletion(t *testing.T) {
	auditor := &fakeAudit{}
	want := []domain.ScanFinding{{
		Path: "config.txt", Line: 2, Column: 1, Category: "password-assignment",
		Severity: domain.SeverityWarning, Message: "password-like assignment found",
	}}
	service := NewService(Dependencies{
		Projects: fakeProjects{project: domain.Project{ID: "project-1", Name: "Demo", Path: "/project"}},
		Scanner:  fakeScanner{findings: want},
		Auditor:  auditor,
	})
	got, err := service.ScanCurrentProject(context.Background())
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("scan = %#v, %v", got, err)
	}
	if len(auditor.events) != 1 || auditor.events[0].Action != domain.AuditScanCompleted || auditor.events[0].ProjectID != "project-1" {
		t.Fatalf("scan audit events = %#v", auditor.events)
	}
}

func TestScreenServiceAdapterLockLifecycle(t *testing.T) {
	project := domain.Project{ID: "project-1", Name: "Demo"}
	secrets := &fakeSecrets{
		list: []domain.Secret{{ProjectID: "project-1", Name: "TOKEN"}},
	}
	vault := fakeVault{secrets: secrets}

	// 1. Pre-unlocked service
	unlockedService := NewService(Dependencies{
		Projects: fakeProjects{project: project},
		Vault:    vault,
		Secrets:  secrets,
	})
	adapter := NewScreenService(unlockedService)
	if !adapter.IsUnlocked() {
		t.Fatal("pre-unlocked screen service adapter should report IsUnlocked() == true")
	}
	items, err := adapter.ListSecrets(context.Background())
	if err != nil || len(items) != 1 {
		t.Fatalf("ListSecrets before close = %v, %v", items, err)
	}

	// Close adapter and verify it is locked
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
	if adapter.IsUnlocked() {
		t.Fatal("adapter should report IsUnlocked() == false after Close()")
	}
	if _, err := adapter.ListSecrets(context.Background()); !errors.Is(err, ErrMissingSecrets) {
		t.Fatalf("ListSecrets after close error = %v, want ErrMissingSecrets", err)
	}

	// Re-unlocking via adapter
	if err := adapter.Unlock(context.Background(), "pass"); err != nil {
		t.Fatal(err)
	}
	if !adapter.IsUnlocked() {
		t.Fatal("adapter should report IsUnlocked() == true after Unlock()")
	}
	if items, err := adapter.ListSecrets(context.Background()); err != nil || len(items) != 1 {
		t.Fatalf("ListSecrets after re-unlock = %v, %v", items, err)
	}

	// 2. Initially locked service
	lockedService := NewService(Dependencies{
		Projects: fakeProjects{project: project},
		Vault:    vault,
	})
	adapterLocked := NewScreenService(lockedService)
	if adapterLocked.IsUnlocked() {
		t.Fatal("locked screen service adapter should report IsUnlocked() == false")
	}
	if err := adapterLocked.Unlock(context.Background(), "pass"); err != nil {
		t.Fatal(err)
	}
	if !adapterLocked.IsUnlocked() {
		t.Fatal("adapter should report IsUnlocked() == true after Unlock()")
	}
	if err := adapterLocked.Close(); err != nil {
		t.Fatal(err)
	}
	if adapterLocked.IsUnlocked() {
		t.Fatal("adapter should report IsUnlocked() == false after Close()")
	}
}

