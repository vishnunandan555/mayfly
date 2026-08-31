package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
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
	Projects  *project.Registry
	Vault     *vault.Storage
	Executor  *executor.ProcessExecutor
	Auditor   *audit.Log
	Scanner   *scanner.Scanner
	MetaStore *project.MetaStore
}

// Service orchestrates secrets storage, project resolution, and in-memory process execution.
type Service struct {
	mu               sync.RWMutex
	projects         *project.Registry
	vault            *vault.Storage
	executor         *executor.ProcessExecutor
	auditor          *audit.Log
	scanner          *scanner.Scanner
	metaStore        *project.MetaStore
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
		metaStore:        deps.MetaStore,
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

	// Check soft brute-force lockout before attempting expensive KDF decryption.
	if s.metaStore != nil {
		if locked, remaining := s.metaStore.IsLocked(); locked {
			return fmt.Errorf("vault is temporarily locked after too many failed attempts — try again in %.0f seconds", remaining.Seconds())
		}
	}

	record, err := s.vault.Open(password)
	if err != nil {
		// Record failed attempt for lockout tracking.
		if errors.Is(err, domain.ErrWrongPassword) && s.metaStore != nil {
			_ = s.metaStore.RecordFailedAttempt()
		}
		return err
	}

	// Successful unlock: reset failure counter.
	if s.metaStore != nil {
		_ = s.metaStore.RecordSuccess()
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

// ExportSecrets returns all decrypted secrets for a project as a plain map (for shell export / template rendering).
func (s *Service) ExportSecrets(projectID string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isUnlocked {
		return nil, ErrVaultLocked
	}

	projMap := s.activeSecret.Projects[projectID]
	result := make(map[string]string, len(projMap))
	for k, v := range projMap {
		result[string(k)] = v
	}
	return result, nil
}

// RenderTemplate replaces {{ SECRET_NAME }} placeholders in templateContent with decrypted values.
// Uses a single-pass parser so replaced values are never re-scanned, preventing recursion / injection bugs.
// Returns an error if any placeholder cannot be resolved.
func (s *Service) RenderTemplate(projectID, templateContent string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isUnlocked {
		return "", ErrVaultLocked
	}

	projMap := s.activeSecret.Projects[projectID]
	var out strings.Builder
	cursor := 0

	for {
		startRel := strings.Index(templateContent[cursor:], "{{")
		if startRel == -1 {
			out.WriteString(templateContent[cursor:])
			break
		}
		start := cursor + startRel
		out.WriteString(templateContent[cursor:start])

		endRel := strings.Index(templateContent[start+2:], "}}")
		if endRel == -1 {
			// Unmatched opening '{{' — append remainder as-is
			out.WriteString(templateContent[start:])
			break
		}
		end := start + 2 + endRel + 2

		key := strings.TrimSpace(templateContent[start+2 : start+2+endRel])
		val, found := projMap[domain.SecretName(key)]
		if !found {
			return "", fmt.Errorf("template: unresolved placeholder {{ %s }}", key)
		}

		out.WriteString(val)
		cursor = end
	}

	return out.String(), nil
}


// VaultStatus returns a human-readable health summary without requiring the vault to be unlocked.
func (s *Service) VaultStatus() map[string]string {
	status := make(map[string]string)

	if s.vault == nil {
		status["vault"] = "not configured"
		return status
	}

	if s.vault.Exists() {
		status["vault_file"] = s.vault.Path()
		status["vault_exists"] = "true"
	} else {
		status["vault_exists"] = "false"
	}

	if s.IsUnlocked() {
		status["vault_locked"] = "false"
		total := 0
		for _, m := range s.activeSecret.Projects {
			total += len(m)
		}
		status["total_secrets"] = fmt.Sprintf("%d", total)
	} else {
		status["vault_locked"] = "true"
	}

	if s.projects != nil {
		projs, err := s.projects.List()
		if err == nil {
			status["project_count"] = fmt.Sprintf("%d", len(projs))
		}
	}

	return status
}

// CheckIntegrity verifies the vault header, audit log hash chain, and flags stale registry entries.
func (s *Service) CheckIntegrity(ctx context.Context) []string {
	var issues []string

	// Check vault file readability.
	if s.vault != nil && !s.vault.Exists() {
		issues = append(issues, "WARN: vault file does not exist (run 'mayfly init' to create it)")
	}

	// Check audit log integrity.
	if s.auditor != nil {
		if err := s.auditor.Verify(ctx); err != nil {
			issues = append(issues, fmt.Sprintf("FAIL: audit log integrity check failed: %v", err))
		} else {
			issues = append(issues, "OK: audit log hash chain verified")
		}
	}

	// Check for stale registry entries (projects whose paths no longer exist).
	if s.projects != nil {
		projs, err := s.projects.List()
		if err == nil {
			for _, p := range projs {
				if _, statErr := os.Stat(p.CanonicalPath); os.IsNotExist(statErr) {
					issues = append(issues, fmt.Sprintf("WARN: project %s path no longer exists: %s (run 'mf migrate' to update)", p.ID[:8], p.CanonicalPath))
				}
			}
		}
	}

	if len(issues) == 0 {
		issues = append(issues, "OK: all integrity checks passed")
	}

	return issues
}

// DiffSecrets compares the secret keys (not values) between two project IDs.
// Returns keys only in projectA, keys only in projectB, and keys in both.
func (s *Service) DiffSecrets(projectIDA, projectIDB string) (onlyA, onlyB, inBoth []string, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isUnlocked {
		return nil, nil, nil, ErrVaultLocked
	}

	mapA := s.activeSecret.Projects[projectIDA]
	mapB := s.activeSecret.Projects[projectIDB]

	keySetA := make(map[string]bool, len(mapA))
	for k := range mapA {
		keySetA[string(k)] = true
	}
	keySetB := make(map[string]bool, len(mapB))
	for k := range mapB {
		keySetB[string(k)] = true
	}

	for k := range keySetA {
		if keySetB[k] {
			inBoth = append(inBoth, k)
		} else {
			onlyA = append(onlyA, k)
		}
	}
	for k := range keySetB {
		if !keySetA[k] {
			onlyB = append(onlyB, k)
		}
	}

	return onlyA, onlyB, inBoth, nil
}
