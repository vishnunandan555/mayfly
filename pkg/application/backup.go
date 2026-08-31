package application

import (
	"context"
	"errors"
	"runtime"

	"mayfly/pkg/domain"
)

// RotatePassword re-encrypts the master vault storage using a new master password and a fresh random salt.
func (s *Service) RotatePassword(ctx context.Context, oldPassword, newPassword []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vault == nil {
		return ErrMissingVaultStorage
	}

	if len(newPassword) == 0 {
		return errors.New("application: new password cannot be empty")
	}

	// Verify old password by attempting to decrypt the storage record
	record, err := s.vault.Open(oldPassword)
	if err != nil {
		return err
	}

	newPassCopy := append([]byte(nil), newPassword...)
	if err := s.vault.Save(record, newPassCopy); err != nil {
		return err
	}

	// Wipe old password from memory
	for i := range s.password {
		s.password[i] = 0
	}
	runtime.KeepAlive(s.password)

	s.activeSecret = record
	s.password = newPassCopy
	s.isUnlocked = true
	s.resetAutoLockLocked()

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionVaultPasswordRotated, "", "", "", nil)
	}

	return nil
}

// ExportBackup creates an encrypted backup snapshot containing all project registries and encrypted secrets.
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

// RestoreBackup imports an encrypted backup snapshot, updating the project registry and vault storage.
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
