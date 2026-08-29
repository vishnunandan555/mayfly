// tui-demo runs MayFly's application screens against an in-memory demo backend.
// It allows testing all 6 screens (Unlock, Secrets List, Editor, Delete, Scan,
// Audit) interactively in a real terminal without persistent disk side effects.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"mayfly"
	"mayfly/domain"
)

type demoService struct {
	unlocked bool
	secrets  []domain.SecretMaterial
	findings []domain.ScanFinding
	events   []domain.AuditEvent
}

func newDemoService() *demoService {
	return &demoService{
		unlocked: false,
		secrets: []domain.SecretMaterial{
			{Name: "OPENAI_API_KEY", Value: "sk-demo-live-openai-key-sample"},
			{Name: "DATABASE_URL", Value: "postgres://user:pass@localhost:5432/app_db"},
			{Name: "STRIPE_SECRET_KEY", Value: "sk_test_demo_stripe_token_sample"},
			{Name: "AWS_ACCESS_KEY_ID", Value: "AKIAIOSFODNN7EXAMPLE"},
			{Name: "REDIS_AUTH_TOKEN", Value: "auth-sample-token-12345"},
		},
		findings: []domain.ScanFinding{
			{
				Severity: domain.SeverityCritical,
				Path:     ".env.local",
				Line:     1,
				Column:   1,
				Category: "high-risk-filename",
				Message:  "High-risk credential filename (.env.local)",
			},
			{
				Severity: domain.SeverityWarning,
				Path:     "src/api/client.js",
				Line:     18,
				Column:   7,
				Category: "potential-api-key",
				Message:  "Potential hardcoded API key assignment",
			},
			{
				Severity: domain.SeverityInfo,
				Path:     "config/database.yml",
				Line:     4,
				Column:   12,
				Category: "password-field",
				Message:  "Password configuration field in source tree",
			},
		},
		events: []domain.AuditEvent{
			{
				At:        time.Now().Add(-15 * time.Minute),
				Action:    domain.AuditProjectInitialized,
				ProjectID: "project-mayfly-demo",
			},
			{
				At:        time.Now().Add(-10 * time.Minute),
				Action:    domain.AuditVaultUnlocked,
				ProjectID: "project-mayfly-demo",
			},
			{
				At:        time.Now().Add(-8 * time.Minute),
				Action:    domain.AuditSecretCreated,
				ProjectID: "project-mayfly-demo",
				Secret:    "OPENAI_API_KEY",
			},
			{
				At:        time.Now().Add(-5 * time.Minute),
				Action:    domain.AuditSecretCreated,
				ProjectID: "project-mayfly-demo",
				Secret:    "DATABASE_URL",
			},
			{
				At:        time.Now().Add(-2 * time.Minute),
				Action:    domain.AuditScanCompleted,
				ProjectID: "project-mayfly-demo",
			},
		},
	}
}

func (d *demoService) ProjectPath(context.Context) (string, error) {
	return "~/code/mayfly-demo", nil
}

func (d *demoService) Unlock(context.Context, string) error {
	d.unlocked = true
	d.events = append(d.events, domain.AuditEvent{
		At:        time.Now(),
		Action:    domain.AuditVaultUnlocked,
		ProjectID: "project-mayfly-demo",
	})
	return nil
}

func (d *demoService) IsUnlocked() bool {
	return d.unlocked
}

func (d *demoService) ListSecrets(context.Context) ([]domain.Secret, error) {
	items := make([]domain.Secret, len(d.secrets))
	for i, s := range d.secrets {
		items[i] = domain.Secret{
			ProjectID: "project-mayfly-demo",
			Name:      s.Name,
		}
	}
	return items, nil
}

func (d *demoService) GetSecret(_ context.Context, name domain.SecretName) (domain.SecretMaterial, error) {
	for _, s := range d.secrets {
		if s.Name == name {
			return s, nil
		}
	}
	return domain.SecretMaterial{}, domain.ErrInvalidSecretName
}

func (d *demoService) SetSecret(_ context.Context, name domain.SecretName, value string) error {
	for i := range d.secrets {
		if d.secrets[i].Name == name {
			d.secrets[i].Value = value
			d.events = append(d.events, domain.AuditEvent{
				At:        time.Now(),
				Action:    domain.AuditSecretUpdated,
				ProjectID: "project-mayfly-demo",
				Secret:    name,
			})
			return nil
		}
	}
	d.secrets = append(d.secrets, domain.SecretMaterial{Name: name, Value: value})
	d.events = append(d.events, domain.AuditEvent{
		At:        time.Now(),
		Action:    domain.AuditSecretCreated,
		ProjectID: "project-mayfly-demo",
		Secret:    name,
	})
	return nil
}

func (d *demoService) DeleteSecret(_ context.Context, name domain.SecretName) error {
	for i := range d.secrets {
		if d.secrets[i].Name == name {
			d.secrets = append(d.secrets[:i], d.secrets[i+1:]...)
			d.events = append(d.events, domain.AuditEvent{
				At:        time.Now(),
				Action:    domain.AuditSecretDeleted,
				ProjectID: "project-mayfly-demo",
				Secret:    name,
			})
			return nil
		}
	}
	return fmt.Errorf("secret not found")
}

func (d *demoService) Scan(context.Context) ([]domain.ScanFinding, error) {
	d.events = append(d.events, domain.AuditEvent{
		At:        time.Now(),
		Action:    domain.AuditScanCompleted,
		ProjectID: "project-mayfly-demo",
	})
	return append([]domain.ScanFinding(nil), d.findings...), nil
}

func (d *demoService) AuditEvents(context.Context) ([]domain.AuditEvent, error) {
	return append([]domain.AuditEvent(nil), d.events...), nil
}

func (d *demoService) VerifyAudit(context.Context) error {
	return nil
}

func (d *demoService) Close() error {
	d.unlocked = false
	return nil
}

func main() {
	service := newDemoService()
	screens := mayfly.NewScreens(service)
	if err := screens.RunIO(os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "tui-demo: TUI stopped:", err)
	}
}
