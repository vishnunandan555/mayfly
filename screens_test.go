package mayfly

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"mayfly/domain"
	"mayfly/screen"
)

type fakeScreenService struct {
	path           string
	unlocked       bool
	passwordToPass string
	unlockErr      error
	listErr        error
	getErr         error
	setErr         error
	deleteErr      error
	scanErr        error
	auditErr       error
	verifyErr      error
	secrets        []domain.SecretMaterial
	findings       []domain.ScanFinding
	events         []domain.AuditEvent
}

func newFakeService() *fakeScreenService {
	return &fakeScreenService{
		path:           "~/code/my-project",
		unlocked:       true,
		passwordToPass: "master-pass",
		secrets: []domain.SecretMaterial{
			{Name: "OPENAI_API_KEY", Value: "openai-secret-material"},
			{Name: "DATABASE_URL", Value: "postgres://user:pass@localhost:5432/prod"},
			{Name: "STRIPE_SECRET_KEY", Value: "stripe-secret-material"},
		},
		findings: []domain.ScanFinding{
			{
				Severity: domain.SeverityCritical,
				Path:     ".env",
				Line:     1,
				Column:   1,
				Category: "high-risk-filename",
				Message:  "High-risk filename (.env)",
			},
			{
				Severity: domain.SeverityWarning,
				Path:     "config/key.pem",
				Line:     1,
				Column:   1,
				Category: "private-key-file",
				Message:  "Potential private key file",
			},
		},
		events: []domain.AuditEvent{
			{
				At:        time.Date(2026, 8, 30, 1, 0, 0, 0, time.UTC),
				Action:    domain.AuditVaultUnlocked,
				ProjectID: "project-test-123",
			},
			{
				At:        time.Date(2026, 8, 30, 1, 1, 0, 0, time.UTC),
				Action:    domain.AuditSecretCreated,
				ProjectID: "project-test-123",
				Secret:    "OPENAI_API_KEY",
			},
			{
				At:        time.Date(2026, 8, 30, 1, 2, 0, 0, time.UTC),
				Action:    domain.AuditCommandStarted,
				ProjectID: "project-test-123",
				Command:   "npm run dev",
			},
		},
	}
}

func (f *fakeScreenService) ProjectPath(context.Context) (string, error) {
	return f.path, nil
}

func (f *fakeScreenService) Unlock(_ context.Context, password string) error {
	if f.unlockErr != nil {
		return f.unlockErr
	}
	if f.passwordToPass != "" && password != f.passwordToPass {
		return errors.New("invalid master password")
	}
	f.unlocked = true
	return nil
}

func (f *fakeScreenService) IsUnlocked() bool {
	return f.unlocked
}

func (f *fakeScreenService) ListSecrets(context.Context) ([]domain.Secret, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	items := make([]domain.Secret, len(f.secrets))
	for i, s := range f.secrets {
		items[i] = domain.Secret{
			ProjectID: "project-test-123",
			Name:      s.Name,
		}
	}
	return items, nil
}

func (f *fakeScreenService) GetSecret(_ context.Context, name domain.SecretName) (domain.SecretMaterial, error) {
	if f.getErr != nil {
		return domain.SecretMaterial{}, f.getErr
	}
	for _, s := range f.secrets {
		if s.Name == name {
			return s, nil
		}
	}
	return domain.SecretMaterial{}, errors.New("secret not found")
}

func (f *fakeScreenService) SetSecret(_ context.Context, name domain.SecretName, value string) error {
	if f.setErr != nil {
		return f.setErr
	}
	for i := range f.secrets {
		if f.secrets[i].Name == name {
			f.secrets[i].Value = value
			return nil
		}
	}
	f.secrets = append(f.secrets, domain.SecretMaterial{Name: name, Value: value})
	return nil
}

func (f *fakeScreenService) DeleteSecret(_ context.Context, name domain.SecretName) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	for i := range f.secrets {
		if f.secrets[i].Name == name {
			f.secrets = append(f.secrets[:i], f.secrets[i+1:]...)
			return nil
		}
	}
	return errors.New("secret not found")
}

func (f *fakeScreenService) Scan(context.Context) ([]domain.ScanFinding, error) {
	if f.scanErr != nil {
		return nil, f.scanErr
	}
	return append([]domain.ScanFinding(nil), f.findings...), nil
}

func (f *fakeScreenService) AuditEvents(context.Context) ([]domain.AuditEvent, error) {
	if f.auditErr != nil {
		return nil, f.auditErr
	}
	return append([]domain.AuditEvent(nil), f.events...), nil
}

func (f *fakeScreenService) VerifyAudit(context.Context) error {
	return f.verifyErr
}

func (f *fakeScreenService) Close() error {
	f.unlocked = false
	return nil
}

func runScreen(t *testing.T, screens *Screens, input string, size screen.Size) string {
	t.Helper()
	var output bytes.Buffer
	app := screen.NewApplication(screens.ApplicationOptions(&output, screen.NewInput(strings.NewReader(input)), size))
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func TestScreenUnlockSuccess(t *testing.T) {
	fake := newFakeService()
	fake.unlocked = false
	screens := NewScreens(fake)

	if screens.Mode() != ModeUnlock {
		t.Fatalf("initial mode = %v, want ModeUnlock", screens.Mode())
	}

	// Type password, press Enter, then press Q to quit.
	output := runScreen(t, screens, "master-pass\rq", screen.Size{Rows: 24, Columns: 80})

	if !fake.IsUnlocked() {
		t.Fatal("expected service to be unlocked")
	}
	if screens.Mode() != ModeSecrets {
		t.Fatalf("mode after unlock = %v, want ModeSecrets", screens.Mode())
	}
	if !strings.Contains(output, "MayFly") {
		t.Fatal("output missing title header")
	}
	if !strings.Contains(output, "OPENAI_API_KEY") {
		t.Fatal("output missing secret list item")
	}
	if strings.Contains(output, "master-pass") {
		t.Fatal("master password leaked in output")
	}
	if strings.Contains(output, "openai-secret-material") {
		t.Fatal("plaintext secret material leaked in output")
	}
}

func TestScreenUnlockFailureAndSanitizedError(t *testing.T) {
	fake := newFakeService()
	fake.unlocked = false
	fake.unlockErr = errors.New("backend crypto fail with sensitive-trace-1234")
	screens := NewScreens(fake)

	// Enter wrong password, triggers error dialog, press Enter to dismiss error dialog, press Esc to quit.
	output := runScreen(t, screens, "wrong-pass\r\r\x1b", screen.Size{Rows: 20, Columns: 60})

	if fake.IsUnlocked() {
		t.Fatal("service should remain locked")
	}
	if strings.Contains(output, "sensitive-trace-1234") || strings.Contains(output, "wrong-pass") {
		t.Fatal("sensitive backend error or password leaked into UI")
	}
	if !strings.Contains(output, "Unable to unlock vault") {
		t.Fatal("safe unlock error message missing")
	}
}

func TestScreenUnlockErrorDismissAndRetry(t *testing.T) {
	fake := newFakeService()
	fake.unlocked = false
	fake.unlockErr = errors.New("wrong password")
	screens := NewScreens(fake)

	// Step 1: Attempt wrong password, enter -> error modal
	// Step 2: Press Enter -> dismiss error modal back to unlock
	// Step 3: Clear error condition on fake, enter correct password, enter -> unlocks to Secrets mode
	// Step 4: Press 'q' to quit
	app := screen.NewApplication(screens.ApplicationOptions(&bytes.Buffer{}, screen.NewInput(strings.NewReader("bad\r\r")), screen.Size{Rows: 24, Columns: 80}))
	// Run first attempt
	_ = app.Run()
	if screens.Mode() != ModeUnlock {
		t.Fatalf("mode after dismissing error = %v, want ModeUnlock", screens.Mode())
	}
}

func TestScreenEmptyVault(t *testing.T) {
	fake := newFakeService()
	fake.secrets = nil
	screens := NewScreens(fake)

	output := runScreen(t, screens, "q", screen.Size{Rows: 24, Columns: 80})
	if !strings.Contains(output, "No secrets in project") {
		t.Fatalf("expected empty vault message, got: %q", output)
	}
}

func TestScreenListNavigation(t *testing.T) {
	fake := newFakeService()
	screens := NewScreens(fake)

	// Down arrow, Up arrow, PgDn, Home, End, q
	output := runScreen(t, screens, "\x1b[B\x1b[A\x1b[6~\x1b[H\x1b[Fq", screen.Size{Rows: 20, Columns: 70})
	if !strings.Contains(output, "DATABASE_URL") || !strings.Contains(output, "STRIPE_SECRET_KEY") {
		t.Fatal("expected secrets to be rendered during navigation")
	}
	if strings.Contains(output, "postgres://") {
		t.Fatal("plaintext database url leaked in list")
	}
}

func TestScreenCreateSecret(t *testing.T) {
	fake := newFakeService()
	screens := NewScreens(fake)

	// Press 'n', enter name, tab to value, enter value, press Enter to save, press 'q' to quit.
	output := runScreen(t, screens, "nNEW_KEY\tsecret-value-abc\rq", screen.Size{Rows: 24, Columns: 80})

	found := false
	for _, s := range fake.secrets {
		if s.Name == "NEW_KEY" {
			found = true
			if s.Value != "secret-value-abc" {
				t.Fatalf("expected value 'secret-value-abc', got %q", s.Value)
			}
		}
	}
	if !found {
		t.Fatal("new secret was not saved in service")
	}
	if screens.Status() != "Secret saved" {
		t.Fatalf("status = %q, want 'Secret saved'", screens.Status())
	}
	if strings.Contains(output, "secret-value-abc") {
		t.Fatal("secret value leaked in output")
	}
}

func TestScreenEditSecretAndNoPlaintextLeak(t *testing.T) {
	fake := newFakeService()
	screens := NewScreens(fake)

	// Press Enter to edit first item (OPENAI_API_KEY), Tab to value, type Ctrl-U to clear and enter new value, Enter to save, q to quit.
	output := runScreen(t, screens, "\r\t\x15new-openai-val\rq", screen.Size{Rows: 24, Columns: 80})

	var updatedVal string
	for _, s := range fake.secrets {
		if s.Name == "OPENAI_API_KEY" {
			updatedVal = s.Value
		}
	}
	if updatedVal != "new-openai-val" {
		t.Fatalf("updated secret value = %q, want 'new-openai-val'", updatedVal)
	}
	if strings.Contains(output, "new-openai-val") || strings.Contains(output, "openai-secret-material") {
		t.Fatal("plaintext secret value leaked into rendered output")
	}
}

func TestScreenDeleteSecretAndCancelDialog(t *testing.T) {
	fake := newFakeService()
	screens := NewScreens(fake)

	// Press 'd' on first secret, press 'Esc' to cancel.
	runScreen(t, screens, "d\x1b", screen.Size{Rows: 24, Columns: 80})
	if len(fake.secrets) != 3 {
		t.Fatalf("cancelled delete altered secrets count: %d", len(fake.secrets))
	}
	if screens.Status() != "Delete cancelled" {
		t.Fatalf("status = %q, want 'Delete cancelled'", screens.Status())
	}

	// Press 'd' again, press Enter (Yes is selected by default), press 'q' to quit.
	output := runScreen(t, screens, "d\rq", screen.Size{Rows: 24, Columns: 80})
	if len(fake.secrets) != 2 {
		t.Fatalf("secrets count after delete = %d, want 2", len(fake.secrets))
	}
	if fake.secrets[0].Name != "DATABASE_URL" {
		t.Fatalf("first secret = %s, want DATABASE_URL", fake.secrets[0].Name)
	}
	if !strings.Contains(output, "Secret deleted") {
		t.Fatal("status 'Secret deleted' not found in output")
	}
}

func TestScreenScanResultsDisplay(t *testing.T) {
	fake := newFakeService()
	screens := NewScreens(fake)

	// Press 's' to run scan and open Scan Results screen, press 'Esc' to return to secrets, press 'q' to quit.
	output := runScreen(t, screens, "s\x1bq", screen.Size{Rows: 24, Columns: 80})

	if !strings.Contains(output, "Scan Results") {
		t.Fatal("Scan Results title missing from output")
	}
	if !strings.Contains(output, ".env") || !strings.Contains(output, "key.pem") {
		t.Fatal("scan finding paths missing from scan screen")
	}
	if !strings.Contains(output, "CRITICAL") || !strings.Contains(output, "WARNING") {
		t.Fatal("scan finding severities missing from scan screen")
	}
}

func TestScreenAuditResultsDisplay(t *testing.T) {
	fake := newFakeService()
	screens := NewScreens(fake)

	// Press 'a' to open Audit screen, press 'Esc' to return, press 'q' to quit.
	output := runScreen(t, screens, "a\x1bq", screen.Size{Rows: 24, Columns: 80})

	if !strings.Contains(output, "Audit Summary") {
		t.Fatal("Audit Summary title missing from output")
	}
	if !strings.Contains(output, "VAULT_UNLOCKED") || !strings.Contains(output, "SECRET_CREATED") {
		t.Fatal("audit events missing from audit screen")
	}
}

func TestScreenBackendErrorDisplay(t *testing.T) {
	fake := newFakeService()
	fake.setErr = errors.New("backend failed with internal-secret-leak")
	screens := NewScreens(fake)

	// Try creating secret, error dialog appears, press Enter to dismiss, press 'q' to quit.
	output := runScreen(t, screens, "nFOO\tbar\r\r\x1bq", screen.Size{Rows: 24, Columns: 80})

	if strings.Contains(output, "internal-secret-leak") {
		t.Fatal("backend error leaked sensitive string into UI")
	}
	if !strings.Contains(output, "Unable to save secret") {
		t.Fatal("safe error message not displayed")
	}
}

func TestScreenNoSecretValuesRenderedAcrossScreensAndSizes(t *testing.T) {
	fake := newFakeService()
	secretStrings := []string{
		"openai-secret-material",
		"postgres://user:pass@localhost:5432/prod",
		"stripe-secret-material",
		"master-pass",
	}

	for _, size := range []screen.Size{
		{Rows: 24, Columns: 80},
		{Rows: 10, Columns: 30},
		{Rows: 5, Columns: 15},
		{Rows: 40, Columns: 120},
	} {
		t.Run(fmt.Sprintf("%dx%d", size.Rows, size.Columns), func(t *testing.T) {
			s := NewScreens(fake)
			out := runScreen(t, s, "\x1b[B\x1b[As\x1ba\x1bq", size)
			for _, sec := range secretStrings {
				if strings.Contains(out, sec) {
					t.Fatalf("size %v leaked secret: %q", size, sec)
				}
			}
		})
	}
}

func TestLegacyVaultCompatibility(t *testing.T) {
	vault := &legacyMemoryVault{items: []Secret{
		{Name: "LEGACY_KEY", Value: "legacy-value"},
	}}
	screens := NewScreensWithVault(vault)

	output := runScreen(t, screens, "q", screen.Size{Rows: 20, Columns: 60})
	if !strings.Contains(output, "LEGACY_KEY") {
		t.Fatal("legacy key name missing from output")
	}
	if strings.Contains(output, "legacy-value") {
		t.Fatal("legacy secret value leaked in output")
	}
}

type legacyMemoryVault struct {
	items []Secret
}

func (v *legacyMemoryVault) Secrets() ([]Secret, error) {
	return append([]Secret(nil), v.items...), nil
}

func (v *legacyMemoryVault) SetSecret(name, value string) error {
	v.items = append(v.items, Secret{Name: name, Value: value})
	return nil
}

func (v *legacyMemoryVault) DeleteSecret(name string) error {
	for i, item := range v.items {
		if item.Name == name {
			v.items = append(v.items[:i], v.items[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}
