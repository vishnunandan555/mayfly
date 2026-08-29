// Package application contains MayFly's use-case contracts and orchestration.
// It is usable by a CLI, the TUI, or tests and has no terminal or filesystem
// side effects by itself.
package application

import (
	"context"
	"errors"
	"time"

	"mayfly/domain"
)

var (
	ErrMissingVaultStorage   = errors.New("application: vault storage is not configured")
	ErrMissingSecrets        = errors.New("application: secret service is not configured")
	ErrMissingExecutor       = errors.New("application: command executor is not configured")
	ErrMissingProject        = errors.New("application: project lookup is not configured")
	ErrProjectNotInitialized = errors.New("application: project is not initialized")
	ErrVaultMissing          = errors.New("application: vault is missing")
	ErrWrongPassword         = errors.New("application: wrong password")
	ErrSecretNotFound        = errors.New("application: secret not found")
	ErrPersistenceFailed     = errors.New("application: persistence failed")
	ErrAuditFailed           = errors.New("application: audit failed")
	ErrInvalidSecretName     = domain.ErrInvalidSecretName
)

// VaultStorage opens a vault and returns a project-aware secret service. The
// password is borrowed for the call and is not retained by this contract.
type VaultStorage interface {
	Open(ctx context.Context, password []byte) (SecretService, error)
}

// ProjectLookup resolves project identity. Implementations may use the
// current directory, an explicit project registry, or another repository-owned
// strategy; the application layer does not choose one.
type ProjectLookup interface {
	Current(ctx context.Context) (domain.Project, error)
	Get(ctx context.Context, id domain.ProjectID) (domain.Project, error)
	Discover(ctx context.Context, path string) (domain.Project, error)
}

// SecretService is the project-scoped secret operation boundary. List returns
// metadata only; Get is the explicit value-bearing operation.
type SecretService interface {
	List(ctx context.Context, projectID domain.ProjectID) ([]domain.Secret, error)
	Get(ctx context.Context, projectID domain.ProjectID, name domain.SecretName) (domain.SecretMaterial, error)
	Put(ctx context.Context, input domain.SecretInput) error
	Delete(ctx context.Context, projectID domain.ProjectID, name domain.SecretName) error
}

// EnvironmentEntry is transient execution input. It must never be logged or
// included in an error. String redacts the value to reduce accidental leaks.
type EnvironmentEntry struct {
	Name  string
	Value string
}

func (EnvironmentEntry) String() string { return "[REDACTED ENVIRONMENT ENTRY]" }

type Environment []EnvironmentEntry

func (Environment) String() string { return "[REDACTED ENVIRONMENT]" }

// Clear releases references held by transient environment entries as soon as
// the executor returns. Go's garbage collector does not guarantee erasure of
// previously allocated string data, so this is reference cleanup only.
func (e Environment) Clear() {
	for index := range e {
		e[index] = EnvironmentEntry{}
	}
}

// ExecutionResult contains process outcome without retaining its environment.
type ExecutionResult struct {
	ExitCode int
}

// CommandExecutor is the only boundary allowed to launch a child process.
// A concrete implementation will use os/exec; this package does not launch
// anything itself.
type CommandExecutor interface {
	Execute(ctx context.Context, request domain.ExecutionRequest, environment Environment) (ExecutionResult, error)
}

// AuditService receives metadata-only events. It must not be passed secret
// values or the transient Environment.
type AuditService interface {
	Record(ctx context.Context, event domain.AuditEvent) error
}

// Scanner is the optional leak-scanning boundary. Findings must contain
// locations and safe descriptions, never the matched secret itself.
type Scanner interface {
	Scan(ctx context.Context, project domain.Project) ([]domain.ScanFinding, error)
}

// Dependencies are explicit so services can be assembled by a CLI, TUI, or
// test without process-global state or a real terminal.
type Dependencies struct {
	Projects ProjectLookup
	Vault    VaultStorage
	Secrets  SecretService
	Executor CommandExecutor
	Auditor  AuditService
	Scanner  Scanner
}

// Service coordinates application use cases. It does not know how to draw a
// frame, read keyboard input, encrypt bytes, access files, or start processes.
type Service struct {
	projects ProjectLookup
	vault    VaultStorage
	secrets  SecretService
	executor CommandExecutor
	auditor  AuditService
	scanner  Scanner
}

func NewService(dependencies Dependencies) *Service {
	return &Service{
		projects: dependencies.Projects,
		vault:    dependencies.Vault,
		secrets:  dependencies.Secrets,
		executor: dependencies.Executor,
		auditor:  dependencies.Auditor,
		scanner:  dependencies.Scanner,
	}
}

// OpenVault delegates unlocking to the configured vault boundary and returns
// a new service bound to that opened vault. The original service is unchanged,
// which avoids hidden mutable session state and allows separate callers to
// hold separate vault sessions. The password is copied neither by Service nor
// by this method.
func (s *Service) OpenVault(ctx context.Context, password []byte) (*Service, error) {
	if s == nil || s.vault == nil {
		return nil, ErrMissingVaultStorage
	}
	secrets, err := s.vault.Open(ctx, password)
	if err != nil {
		return nil, err
	}
	opened := NewService(Dependencies{
		Projects: s.projects,
		Vault:    s.vault,
		Secrets:  secrets,
		Executor: s.executor,
		Auditor:  s.auditor,
		Scanner:  s.scanner,
	})
	if err := opened.audit(ctx, domain.AuditEvent{At: time.Now(), Action: domain.AuditVaultUnlocked}); err != nil {
		_ = opened.Close()
		return nil, err
	}
	return opened, nil
}

// Close releases an opened vault session when its SecretService supplies a
// closer. It is safe for services backed by non-owning test doubles and does
// not make storage lifetime a process-global concern.
func (s *Service) Close() error {
	if s == nil || s.secrets == nil {
		return nil
	}
	if closer, ok := s.secrets.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

func (s *Service) ListSecrets(ctx context.Context, projectID domain.ProjectID) ([]domain.Secret, error) {
	if err := projectID.Validate(); err != nil {
		return nil, err
	}
	if s == nil || s.secrets == nil {
		return nil, ErrMissingSecrets
	}
	secrets, err := s.secrets.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := s.audit(ctx, domain.AuditEvent{At: time.Now(), Action: domain.AuditSecretListed, ProjectID: projectID}); err != nil {
		return nil, err
	}
	return secrets, nil
}

// CurrentProject resolves the project containing the current working
// directory through the injected project lookup boundary.
func (s *Service) CurrentProject(ctx context.Context) (domain.Project, error) {
	if s == nil || s.projects == nil {
		return domain.Project{}, ErrMissingProject
	}
	return s.projects.Current(ctx)
}

// GetSecret performs an explicit value-bearing read and audits only its safe
// metadata after the read succeeds. The value is returned only to the caller.
func (s *Service) GetSecret(ctx context.Context, projectID domain.ProjectID, name domain.SecretName) (domain.SecretMaterial, error) {
	if err := projectID.Validate(); err != nil {
		return domain.SecretMaterial{}, err
	}
	if err := name.Validate(); err != nil {
		return domain.SecretMaterial{}, err
	}
	if s == nil || s.secrets == nil {
		return domain.SecretMaterial{}, ErrMissingSecrets
	}
	material, err := s.secrets.Get(ctx, projectID, name)
	if err != nil {
		return domain.SecretMaterial{}, err
	}
	if err := s.audit(ctx, domain.AuditEvent{At: time.Now(), Action: domain.AuditSecretRead, ProjectID: projectID, Secret: name}); err != nil {
		return domain.SecretMaterial{}, err
	}
	return material, nil
}

// ListCurrentSecrets lists metadata for the initialized current project.
func (s *Service) ListCurrentSecrets(ctx context.Context) ([]domain.Secret, error) {
	project, err := s.CurrentProject(ctx)
	if err != nil {
		return nil, err
	}
	return s.ListSecrets(ctx, project.ID)
}

// GetCurrentSecret reads one explicitly requested secret from the current
// project.
func (s *Service) GetCurrentSecret(ctx context.Context, name domain.SecretName) (domain.SecretMaterial, error) {
	project, err := s.CurrentProject(ctx)
	if err != nil {
		return domain.SecretMaterial{}, err
	}
	return s.GetSecret(ctx, project.ID, name)
}

// DiscoverProject resolves a path to an initialized project without exposing
// filesystem or registry details to callers such as the CLI or TUI.
func (s *Service) DiscoverProject(ctx context.Context, path string) (domain.Project, error) {
	if s == nil || s.projects == nil {
		return domain.Project{}, ErrMissingProject
	}
	return s.projects.Discover(ctx, path)
}

// ListSecretsAt discovers the initialized project containing path, then lists
// only that project's metadata. Secret isolation is enforced by carrying the
// discovered ProjectID into the SecretService call.
func (s *Service) ListSecretsAt(ctx context.Context, path string) ([]domain.Secret, error) {
	project, err := s.DiscoverProject(ctx, path)
	if err != nil {
		return nil, err
	}
	return s.ListSecrets(ctx, project.ID)
}

func (s *Service) SaveSecret(ctx context.Context, input domain.SecretInput) error {
	if err := input.ProjectID.Validate(); err != nil {
		return err
	}
	if err := input.Name.Validate(); err != nil {
		return err
	}
	if err := input.Validate(); err != nil {
		return err
	}
	if s == nil || s.secrets == nil {
		return ErrMissingSecrets
	}
	if err := s.secrets.Put(ctx, input); err != nil {
		return err
	}
	return s.audit(ctx, domain.AuditEvent{At: time.Now(), Action: domain.AuditSecretWritten, ProjectID: input.ProjectID, Secret: input.Name})
}

// SetSecret is the application-level name for the create-or-overwrite
// operation. SaveSecret remains as a compatibility alias.
func (s *Service) SetSecret(ctx context.Context, input domain.SecretInput) error {
	return s.SaveSecret(ctx, input)
}

func (s *Service) SetCurrentSecret(ctx context.Context, name domain.SecretName, value string) error {
	project, err := s.CurrentProject(ctx)
	if err != nil {
		return err
	}
	return s.SetSecret(ctx, domain.SecretInput{ProjectID: project.ID, Name: name, Value: value})
}

func (s *Service) DeleteSecret(ctx context.Context, projectID domain.ProjectID, name domain.SecretName) error {
	if err := projectID.Validate(); err != nil {
		return err
	}
	if err := name.Validate(); err != nil {
		return err
	}
	if s == nil || s.secrets == nil {
		return ErrMissingSecrets
	}
	if err := s.secrets.Delete(ctx, projectID, name); err != nil {
		return err
	}
	return s.audit(ctx, domain.AuditEvent{At: time.Now(), Action: domain.AuditSecretDeleted, ProjectID: projectID, Secret: name})
}

func (s *Service) DeleteCurrentSecret(ctx context.Context, name domain.SecretName) error {
	project, err := s.CurrentProject(ctx)
	if err != nil {
		return err
	}
	return s.DeleteSecret(ctx, project.ID, name)
}

func (s *Service) audit(ctx context.Context, event domain.AuditEvent) error {
	if s == nil || s.auditor == nil {
		return nil
	}
	if err := event.Validate(); err != nil {
		return ErrAuditFailed
	}
	if err := s.auditor.Record(ctx, event); err != nil {
		// Do not return an implementation error: a faulty auditor must not
		// leak a secret value that it may have embedded in its error.
		return ErrAuditFailed
	}
	return nil
}

// Run resolves selected secrets only in memory, passes them to the executor,
// and records metadata after execution. An empty SecretNames list means all
// secrets in the project. Secret values never enter ExecutionRequest,
// ExecutionResult, AuditEvent, or any service error created here.
func (s *Service) Run(ctx context.Context, request domain.ExecutionRequest) (ExecutionResult, error) {
	if err := request.Validate(); err != nil {
		return ExecutionResult{}, err
	}
	if s == nil || s.secrets == nil {
		return ExecutionResult{}, ErrMissingSecrets
	}
	if s.executor == nil {
		return ExecutionResult{}, ErrMissingExecutor
	}
	if s.projects == nil {
		return ExecutionResult{}, ErrMissingProject
	}
	if _, err := s.projects.Get(ctx, request.ProjectID); err != nil {
		return ExecutionResult{}, err
	}
	if err := s.audit(ctx, domain.AuditEvent{
		At: time.Now(), Action: domain.AuditCommandStarted,
		ProjectID: request.ProjectID, Command: request.Command[0],
	}); err != nil {
		return ExecutionResult{}, err
	}

	names := append([]domain.SecretName(nil), request.SecretNames...)
	if len(names) == 0 {
		metadata, err := s.secrets.List(ctx, request.ProjectID)
		if err != nil {
			return ExecutionResult{}, err
		}
		for _, secret := range metadata {
			names = append(names, secret.Name)
		}
	}

	environment := make(Environment, 0, len(names))
	for _, name := range names {
		material, err := s.secrets.Get(ctx, request.ProjectID, name)
		if err != nil {
			// Do not decorate backend errors: a faulty backend must not turn a
			// secret value embedded in its error into an application log leak.
			return ExecutionResult{}, err
		}
		environment = append(environment, EnvironmentEntry{Name: string(name), Value: material.Value})
		if err := s.audit(ctx, domain.AuditEvent{
			At: time.Now(), Action: domain.AuditSecretInjected,
			ProjectID: request.ProjectID, Secret: name,
		}); err != nil {
			return ExecutionResult{}, err
		}
	}

	defer environment.Clear()
	result, err := s.executor.Execute(ctx, request, environment)
	if s.auditor != nil {
		status := result.ExitCode
		auditErr := s.auditor.Record(ctx, domain.AuditEvent{
			At:         time.Now(),
			Action:     domain.AuditCommandExited,
			ProjectID:  request.ProjectID,
			Command:    request.Command[0],
			ExitStatus: &status,
		})
		if auditErr != nil {
			auditErr = ErrAuditFailed
			if err == nil {
				err = auditErr
			} else {
				err = errors.Join(err, auditErr)
			}
		}
	}
	return result, err
}
