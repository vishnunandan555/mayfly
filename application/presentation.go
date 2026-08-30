package application

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"mayfly/domain"
)

// ScreenSecret is the transitional, value-bearing record contract used by
// the earlier test harnesses.
type ScreenSecret struct {
	Name  string
	Value string
}

// ScreenVault is the legacy application-facing contract consumed by
// earlier test screens.
type ScreenVault interface {
	Secrets() ([]ScreenSecret, error)
	SetSecret(name, value string) error
	DeleteSecret(name string) error
}

// ScreenVaultOpener is the legacy unlock boundary used by earlier test screens.
type ScreenVaultOpener interface {
	Unlock(password string) (ScreenVault, error)
}

// ScreenService represents the complete application use-case boundary required by
// the MayFly TUI screens. The TUI interacts with MayFly purely through this
// interface, without knowing about PBKDF2, AES-GCM, storage format, process execution,
// or OS-level environment inspection.
type ScreenService interface {
	// ProjectPath returns the current project path or display name for screen headers.
	ProjectPath(ctx context.Context) (string, error)

	// Unlock unlocks the vault using the given master password.
	Unlock(ctx context.Context, password string) error

	// IsUnlocked returns whether the service currently has an unlocked vault session.
	IsUnlocked() bool

	// ListSecrets returns the metadata/names of all secrets in the current project.
	// Plaintext secret values are NEVER returned in this list.
	ListSecrets(ctx context.Context) ([]domain.Secret, error)

	// GetSecret retrieves a single secret value on demand for editing.
	GetSecret(ctx context.Context, name domain.SecretName) (domain.SecretMaterial, error)

	// SetSecret creates or updates a secret in the current project.
	SetSecret(ctx context.Context, name domain.SecretName, value string) error

	// DeleteSecret removes a secret from the current project.
	DeleteSecret(ctx context.Context, name domain.SecretName) error

	// Scan performs heuristic secret scanning on the current project.
	Scan(ctx context.Context) ([]domain.ScanFinding, error)

	// AuditEvents returns the safe metadata audit log events.
	AuditEvents(ctx context.Context) ([]domain.AuditEvent, error)

	// VerifyAudit verifies the integrity of the audit trail.
	VerifyAudit(ctx context.Context) error

	// Close closes any active unlocked vault session.
	Close() error
}

type screenServiceAdapter struct {
	base     *Service
	unlocked *Service
}

// NewScreenService creates a ScreenService backed by an application.Service.
func NewScreenService(service *Service) ScreenService {
	if service == nil {
		return &screenServiceAdapter{}
	}
	adapter := &screenServiceAdapter{}
	if service.secrets != nil {
		adapter.unlocked = service
		adapter.base = NewService(Dependencies{
			Projects: service.projects,
			Vault:    service.vault,
			Executor: service.executor,
			Auditor:  service.auditor,
			Scanner:  service.scanner,
		})
	} else {
		adapter.base = service
	}
	return adapter
}

func (a *screenServiceAdapter) active() *Service {
	if a == nil {
		return nil
	}
	if a.unlocked != nil {
		return a.unlocked
	}
	return a.base
}

func (a *screenServiceAdapter) ProjectPath(ctx context.Context) (string, error) {
	if a == nil || a.active() == nil {
		return "", ErrMissingProject
	}
	project, err := a.active().CurrentProject(ctx)
	if err != nil {
		return "", err
	}
	if project.Path != "" {
		return cleanDisplayPath(project.Path), nil
	}
	return project.Name, nil
}

func (a *screenServiceAdapter) Unlock(ctx context.Context, password string) error {
	if a == nil || a.base == nil {
		return ErrMissingVaultStorage
	}
	opened, err := a.base.OpenVault(ctx, []byte(password))
	if err != nil {
		return err
	}
	a.unlocked = opened
	return nil
}

func (a *screenServiceAdapter) IsUnlocked() bool {
	if a == nil {
		return false
	}
	return a.unlocked != nil
}

func (a *screenServiceAdapter) ListSecrets(ctx context.Context) ([]domain.Secret, error) {
	if a == nil || a.unlocked == nil {
		return nil, ErrMissingSecrets
	}
	return a.unlocked.ListCurrentSecrets(ctx)
}

func (a *screenServiceAdapter) GetSecret(ctx context.Context, name domain.SecretName) (domain.SecretMaterial, error) {
	if a == nil || a.unlocked == nil {
		return domain.SecretMaterial{}, ErrMissingSecrets
	}
	return a.unlocked.GetCurrentSecret(ctx, name)
}

func (a *screenServiceAdapter) SetSecret(ctx context.Context, name domain.SecretName, value string) error {
	if a == nil || a.unlocked == nil {
		return ErrMissingSecrets
	}
	return a.unlocked.SetCurrentSecret(ctx, name, value)
}

func (a *screenServiceAdapter) DeleteSecret(ctx context.Context, name domain.SecretName) error {
	if a == nil || a.unlocked == nil {
		return ErrMissingSecrets
	}
	return a.unlocked.DeleteCurrentSecret(ctx, name)
}

func (a *screenServiceAdapter) Scan(ctx context.Context) ([]domain.ScanFinding, error) {
	if a == nil || a.active() == nil {
		return nil, ErrMissingScanner
	}
	return a.active().ScanCurrentProject(ctx)
}

func (a *screenServiceAdapter) AuditEvents(ctx context.Context) ([]domain.AuditEvent, error) {
	if a == nil || a.active() == nil {
		return nil, ErrAuditFailed
	}
	return a.active().AuditEvents(ctx)
}

func (a *screenServiceAdapter) VerifyAudit(ctx context.Context) error {
	if a == nil || a.active() == nil {
		return ErrAuditFailed
	}
	return a.active().VerifyAudit(ctx)
}

func (a *screenServiceAdapter) Close() error {
	if a == nil {
		return nil
	}
	if a.unlocked != nil {
		err := a.unlocked.Close()
		a.unlocked = nil
		return err
	}
	return nil
}

func cleanDisplayPath(path string) string {
	cleaned := filepath.Clean(path)
	return strings.ReplaceAll(cleaned, "\\", "/")
}

// ScreenServiceFromVault adapts a legacy ScreenVault to the ScreenService interface.
func ScreenServiceFromVault(vault ScreenVault) ScreenService {
	return &legacyVaultAdapter{vault: vault}
}

// ScreenServiceFromOpener adapts a legacy ScreenVaultOpener to the ScreenService interface.
func ScreenServiceFromOpener(opener ScreenVaultOpener) ScreenService {
	return &legacyOpenerAdapter{opener: opener}
}

type legacyVaultAdapter struct {
	vault  ScreenVault
	closed bool
}

func (l *legacyVaultAdapter) ProjectPath(context.Context) (string, error) { return "", nil }
func (l *legacyVaultAdapter) Unlock(context.Context, string) error        { return nil }
func (l *legacyVaultAdapter) IsUnlocked() bool                           { return l != nil && l.vault != nil && !l.closed }
func (l *legacyVaultAdapter) Close() error                               { if l != nil { l.closed = true; l.vault = nil }; return nil }

func (l *legacyVaultAdapter) ListSecrets(context.Context) ([]domain.Secret, error) {
	if l.vault == nil {
		return nil, errors.New("vault unavailable")
	}
	items, err := l.vault.Secrets()
	if err != nil {
		return nil, err
	}
	secrets := make([]domain.Secret, len(items))
	for i, item := range items {
		secrets[i] = domain.Secret{Name: domain.SecretName(item.Name)}
	}
	return secrets, nil
}

func (l *legacyVaultAdapter) GetSecret(_ context.Context, name domain.SecretName) (domain.SecretMaterial, error) {
	if l.vault == nil {
		return domain.SecretMaterial{}, errors.New("vault unavailable")
	}
	items, err := l.vault.Secrets()
	if err != nil {
		return domain.SecretMaterial{}, err
	}
	for _, item := range items {
		if item.Name == string(name) {
			return domain.SecretMaterial{Name: name, Value: item.Value}, nil
		}
	}
	return domain.SecretMaterial{}, ErrSecretNotFound
}

func (l *legacyVaultAdapter) SetSecret(_ context.Context, name domain.SecretName, value string) error {
	if l.vault == nil {
		return errors.New("vault unavailable")
	}
	return l.vault.SetSecret(string(name), value)
}

func (l *legacyVaultAdapter) DeleteSecret(_ context.Context, name domain.SecretName) error {
	if l.vault == nil {
		return errors.New("vault unavailable")
	}
	return l.vault.DeleteSecret(string(name))
}

func (l *legacyVaultAdapter) Scan(context.Context) ([]domain.ScanFinding, error) {
	return nil, nil
}

func (l *legacyVaultAdapter) AuditEvents(context.Context) ([]domain.AuditEvent, error) {
	return nil, nil
}

func (l *legacyVaultAdapter) VerifyAudit(context.Context) error {
	return nil
}

type legacyOpenerAdapter struct {
	opener ScreenVaultOpener
	vault  ScreenVault
}

func (l *legacyOpenerAdapter) ProjectPath(context.Context) (string, error) { return "", nil }

func (l *legacyOpenerAdapter) Unlock(_ context.Context, password string) error {
	if l.opener == nil {
		return errors.New("opener unavailable")
	}
	vault, err := l.opener.Unlock(password)
	if err != nil {
		return err
	}
	if vault == nil {
		return errors.New("unlock failed")
	}
	l.vault = vault
	return nil
}

func (l *legacyOpenerAdapter) IsUnlocked() bool { return l.vault != nil }
func (l *legacyOpenerAdapter) Close() error     { return nil }

func (l *legacyOpenerAdapter) ListSecrets(ctx context.Context) ([]domain.Secret, error) {
	if l.vault == nil {
		return nil, errors.New("vault not unlocked")
	}
	adapter := legacyVaultAdapter{vault: l.vault}
	return adapter.ListSecrets(ctx)
}

func (l *legacyOpenerAdapter) GetSecret(ctx context.Context, name domain.SecretName) (domain.SecretMaterial, error) {
	if l.vault == nil {
		return domain.SecretMaterial{}, errors.New("vault not unlocked")
	}
	adapter := legacyVaultAdapter{vault: l.vault}
	return adapter.GetSecret(ctx, name)
}

func (l *legacyOpenerAdapter) SetSecret(ctx context.Context, name domain.SecretName, value string) error {
	if l.vault == nil {
		return errors.New("vault not unlocked")
	}
	adapter := legacyVaultAdapter{vault: l.vault}
	return adapter.SetSecret(ctx, name, value)
}

func (l *legacyOpenerAdapter) DeleteSecret(ctx context.Context, name domain.SecretName) error {
	if l.vault == nil {
		return errors.New("vault not unlocked")
	}
	adapter := legacyVaultAdapter{vault: l.vault}
	return adapter.DeleteSecret(ctx, name)
}

func (l *legacyOpenerAdapter) Scan(context.Context) ([]domain.ScanFinding, error) {
	return nil, nil
}

func (l *legacyOpenerAdapter) AuditEvents(context.Context) ([]domain.AuditEvent, error) {
	return nil, nil
}

func (l *legacyOpenerAdapter) VerifyAudit(context.Context) error {
	return nil
}
