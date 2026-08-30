package application

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"mayfly/pkg/audit"
	"mayfly/pkg/domain"
	"mayfly/pkg/executor"
	"mayfly/pkg/project"
	"mayfly/pkg/scanner"
	"mayfly/pkg/vault"
)

var (
	ErrMissingVaultStorage = errors.New("application: vault storage dependency is missing")
	ErrVaultMissing        = vault.ErrVaultMissing
	ErrVaultExists         = vault.ErrVaultExists
	ErrWrongPassword       = domain.ErrWrongPassword
	ErrVaultLocked         = domain.ErrVaultLocked
	ErrProjectNotFound     = domain.ErrProjectNotFound
	ErrSecretNotFound      = domain.ErrSecretNotFound
	ErrInvalidSecretName   = domain.ErrInvalidSecretName
	ErrAuditFailed         = audit.ErrAuditFailed
)

type Dependencies struct {
	Projects *project.Registry
	Vault    *vault.Storage
	Executor *executor.ProcessExecutor
	Auditor  *audit.Log
	Scanner  *scanner.Scanner
}

// Service is the main orchestration layer.
type Service struct {
	mu           sync.RWMutex
	projects     *project.Registry
	vault        *vault.Storage
	executor     *executor.ProcessExecutor
	auditor      *audit.Log
	scanner      *scanner.Scanner
	activeSecret vault.StorageRecord
	password     []byte
	isUnlocked   bool
}

func NewService(deps Dependencies) *Service {
	return &Service{
		projects: deps.Projects,
		vault:    deps.Vault,
		executor: deps.Executor,
		auditor:  deps.Auditor,
		scanner:  deps.Scanner,
	}
}

func (s *Service) VaultExists() bool {
	if s.vault == nil {
		return false
	}
	return s.vault.Exists()
}

func (s *Service) IsUnlocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isUnlocked
}

// InitializeVault creates a new encrypted vault with master password.
func (s *Service) InitializeVault(ctx context.Context, password []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vault == nil {
		return ErrMissingVaultStorage
	}

	if err := s.vault.Initialize(password); err != nil {
		return err
	}

	record, err := s.vault.Open(password)
	if err != nil {
		return err
	}

	s.activeSecret = record
	s.password = append([]byte(nil), password...)
	s.isUnlocked = true

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionVaultInitialized, "", "", "", nil)
	}

	return nil
}

// UnlockVault decrypts the vault into memory.
func (s *Service) UnlockVault(ctx context.Context, password []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vault == nil {
		return ErrMissingVaultStorage
	}

	record, err := s.vault.Open(password)
	if err != nil {
		return err
	}

	s.activeSecret = record
	s.password = append([]byte(nil), password...)
	s.isUnlocked = true

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionVaultUnlocked, "", "", "", nil)
	}

	return nil
}

// LockVault zeroes out memory buffers and locks the vault.
func (s *Service) LockVault() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.password {
		s.password[i] = 0
	}
	s.password = nil
	s.activeSecret = vault.StorageRecord{}
	s.isUnlocked = false
}

// Projects returns all registered projects.
func (s *Service) Projects() ([]domain.Project, error) {
	if s.projects == nil {
		return nil, ErrProjectNotFound
	}
	return s.projects.List()
}

// ResolveCurrentProject identifies the project in the current working directory.
func (s *Service) ResolveCurrentProject(dir string) (domain.Project, error) {
	if s.projects == nil {
		return domain.Project{}, ErrProjectNotFound
	}
	return s.projects.Resolve(dir)
}

// RegisterProject registers a new project folder.
func (s *Service) RegisterProject(ctx context.Context, dir string) (domain.Project, error) {
	if s.projects == nil {
		return domain.Project{}, ErrProjectNotFound
	}
	proj, err := s.projects.Register(dir)
	if err != nil {
		return domain.Project{}, err
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionProjectInit, proj.ID, "", "", nil)
	}

	return proj, nil
}

// DeleteProject unregisters a project and removes its secrets.
func (s *Service) DeleteProject(ctx context.Context, projectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.projects != nil {
		_ = s.projects.Delete(projectID)
	}

	if s.isUnlocked && s.vault != nil {
		delete(s.activeSecret.Projects, projectID)
		_ = s.vault.Save(s.activeSecret, s.password)
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionProjectDeleted, projectID, "", "", nil)
	}

	return nil
}

// MigrateProject updates project paths when a directory moves.
func (s *Service) MigrateProject(ctx context.Context, oldDir, newDir string) (domain.Project, domain.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.projects == nil {
		return domain.Project{}, domain.Project{}, ErrProjectNotFound
	}

	oldProj, newProj, err := s.projects.MigrateProject(oldDir, newDir)
	if err != nil {
		return domain.Project{}, domain.Project{}, err
	}

	// Migrate secrets map key
	if s.isUnlocked && s.vault != nil {
		if secrets, ok := s.activeSecret.Projects[oldProj.ID]; ok {
			s.activeSecret.Projects[newProj.ID] = secrets
			delete(s.activeSecret.Projects, oldProj.ID)
			_ = s.vault.Save(s.activeSecret, s.password)
		}
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionProjectMigrated, newProj.ID, "", fmt.Sprintf("%s -> %s", oldDir, newDir), nil)
	}

	return oldProj, newProj, nil
}

// ListSecrets returns secret keys for a project.
func (s *Service) ListSecrets(projectID string) ([]domain.Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isUnlocked {
		return nil, ErrVaultLocked
	}

	projMap, ok := s.activeSecret.Projects[projectID]
	if !ok {
		return []domain.Secret{}, nil
	}

	var list []domain.Secret
	for k, v := range projMap {
		list = append(list, domain.Secret{
			Name:  k,
			Value: v,
		})
	}
	return list, nil
}

// GetSecret retrieves a single secret value.
func (s *Service) GetSecret(ctx context.Context, projectID string, name domain.SecretName) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isUnlocked {
		return "", ErrVaultLocked
	}

	projMap, ok := s.activeSecret.Projects[projectID]
	if !ok {
		return "", ErrSecretNotFound
	}

	val, found := projMap[name]
	if !found {
		return "", ErrSecretNotFound
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionSecretGet, projectID, string(name), "", nil)
	}

	return val, nil
}

// SetSecret adds or updates a secret in the encrypted vault.
func (s *Service) SetSecret(ctx context.Context, projectID string, name domain.SecretName, value string) error {
	if err := name.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isUnlocked {
		return ErrVaultLocked
	}

	if s.activeSecret.Projects == nil {
		s.activeSecret.Projects = make(map[string]map[domain.SecretName]string)
	}

	if _, ok := s.activeSecret.Projects[projectID]; !ok {
		s.activeSecret.Projects[projectID] = make(map[domain.SecretName]string)
	}

	s.activeSecret.Projects[projectID][name] = value

	if err := s.vault.Save(s.activeSecret, s.password); err != nil {
		return err
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionSecretSet, projectID, string(name), "", nil)
	}

	return nil
}

// DeleteSecret removes a secret from the vault.
func (s *Service) DeleteSecret(ctx context.Context, projectID string, name domain.SecretName) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isUnlocked {
		return ErrVaultLocked
	}

	projMap, ok := s.activeSecret.Projects[projectID]
	if !ok {
		return ErrSecretNotFound
	}

	if _, found := projMap[name]; !found {
		return ErrSecretNotFound
	}

	delete(projMap, name)
	if err := s.vault.Save(s.activeSecret, s.password); err != nil {
		return err
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionSecretDeleted, projectID, string(name), "", nil)
	}

	return nil
}

// Run executes a command with in-memory secret injection.
func (s *Service) Run(ctx context.Context, req domain.ExecutionRequest) (domain.ExecutionResult, error) {
	s.mu.RLock()
	if !s.isUnlocked {
		s.mu.RUnlock()
		return domain.ExecutionResult{ExitCode: 1}, ErrVaultLocked
	}

	secrets := s.activeSecret.Projects[req.ProjectID]
	s.mu.RUnlock()

	if s.auditor != nil {
		for k := range secrets {
			_ = s.auditor.Record(ctx, domain.ActionSecretInjected, req.ProjectID, string(k), "", nil)
		}
	}

	res, err := s.executor.Execute(ctx, req, secrets)

	if s.auditor != nil && len(req.Command) > 0 {
		code := res.ExitCode
		_ = s.auditor.Record(ctx, domain.ActionCommandExited, req.ProjectID, "", req.Command[0], &code)
	}

	return res, err
}

// Scan scans a directory for plaintext credential exposures.
func (s *Service) Scan(ctx context.Context, dir string) ([]domain.ScanFinding, error) {
	if s.scanner == nil {
		var err error
		s.scanner, err = scanner.New(scanner.Options{})
		if err != nil {
			return nil, err
		}
	}

	findings, err := s.scanner.Scan(ctx, dir)
	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionScanCompleted, "", "", "", nil)
	}
	return findings, err
}

// AuditTrail returns chronological access logs.
func (s *Service) AuditTrail(ctx context.Context) ([]domain.AuditEvent, error) {
	if s.auditor == nil {
		return nil, nil
	}
	return s.auditor.Events(ctx)
}

// VerifyAudit verifies the SHA-256 hash chain of the audit log.
func (s *Service) VerifyAudit(ctx context.Context) error {
	if s.auditor == nil {
		return nil
	}
	return s.auditor.Verify(ctx)
}

// ExportBackup creates an encrypted backup snapshot.
func (s *Service) ExportBackup(ctx context.Context, targetFile string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.vault == nil {
		return ErrMissingVaultStorage
	}

	projects, err := s.projects.AllMap()
	if err != nil {
		return err
	}

	if err := s.vault.ExportSnapshot(targetFile, projects); err != nil {
		return err
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionBackupCreated, "", "", targetFile, nil)
	}

	return nil
}

// RestoreBackup imports an encrypted backup snapshot.
func (s *Service) RestoreBackup(ctx context.Context, snapshotFile string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vault == nil {
		return ErrMissingVaultStorage
	}

	projects, err := s.vault.ImportSnapshot(snapshotFile)
	if err != nil {
		return err
	}

	if s.projects != nil {
		if err := s.projects.ImportMap(projects); err != nil {
			return err
		}
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionBackupRestored, "", "", snapshotFile, nil)
	}

	return nil
}
