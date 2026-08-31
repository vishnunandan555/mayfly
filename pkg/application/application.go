package application

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

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

// Dependencies holds the external subsystem instances required by Service.
type Dependencies struct {
	Projects *project.Registry
	Vault    *vault.Storage
	Executor *executor.ProcessExecutor
	Auditor  *audit.Log
	Scanner  *scanner.Scanner
}

// Service orchestrates secrets storage, project resolution, and in-memory process execution.
type Service struct {
	mu               sync.RWMutex
	projects         *project.Registry
	vault            *vault.Storage
	executor         *executor.ProcessExecutor
	auditor          *audit.Log
	scanner          *scanner.Scanner
	activeSecret     vault.StorageRecord
	password         []byte
	isUnlocked       bool
	autoLockDuration time.Duration
	autoLockTimer    *time.Timer
}

// NewService constructs a new application service instance with a default 15-minute auto-lock timeout.
func NewService(deps Dependencies) *Service {
	return &Service{
		projects:         deps.Projects,
		vault:            deps.Vault,
		executor:         deps.Executor,
		auditor:          deps.Auditor,
		scanner:          deps.Scanner,
		autoLockDuration: 15 * time.Minute,
	}
}

// SetAutoLockTimeout configures the vault auto-lock idle timeout duration.
func (s *Service) SetAutoLockTimeout(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autoLockDuration = d
	s.resetAutoLockLocked()
}

func (s *Service) resetAutoLockLocked() {
	if s.autoLockTimer != nil {
		s.autoLockTimer.Stop()
		s.autoLockTimer = nil
	}
	if s.autoLockDuration > 0 && s.isUnlocked {
		s.autoLockTimer = time.AfterFunc(s.autoLockDuration, func() {
			s.LockVault()
		})
	}
}

// VaultExists checks whether the encrypted vault storage file exists on disk.
func (s *Service) VaultExists() bool {
	if s.vault == nil {
		return false
	}
	return s.vault.Exists()
}

// IsUnlocked returns true if the master password has been verified and the vault is held in memory.
func (s *Service) IsUnlocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isUnlocked
}

// InitializeVault creates a new encrypted vault container with the provided master password.
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
	s.resetAutoLockLocked()

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionVaultInitialized, "", "", "", nil)
	}

	return nil
}

// UnlockVault decrypts the vault storage into active volatile memory using the master password.
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
	s.resetAutoLockLocked()

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionVaultUnlocked, "", "", "", nil)
	}

	return nil
}

// LockVault zeroes out decrypted memory buffers and locks the vault.
func (s *Service) LockVault() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.autoLockTimer != nil {
		s.autoLockTimer.Stop()
		s.autoLockTimer = nil
	}

	for i := range s.password {
		s.password[i] = 0
	}
	runtime.KeepAlive(s.password)
	s.password = nil
	s.activeSecret = vault.StorageRecord{}
	s.isUnlocked = false
}

// Run launches a target command with decrypted project secrets injected directly into volatile RAM.
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

// Scan crawls a directory to detect unencrypted plaintext credential leaks and hardcoded keys.
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

// AuditTrail returns chronological access logs from the cryptographic audit log.
func (s *Service) AuditTrail(ctx context.Context) ([]domain.AuditEvent, error) {
	if s.auditor == nil {
		return nil, nil
	}
	return s.auditor.Events(ctx)
}

// VerifyAudit verifies the SHA-256 Merkle-style hash chain of the audit log file.
func (s *Service) VerifyAudit(ctx context.Context) error {
	if s.auditor == nil {
		return nil
	}
	return s.auditor.Verify(ctx)
}
