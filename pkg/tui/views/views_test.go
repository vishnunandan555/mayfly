package views

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mayfly/pkg/application"
	"mayfly/pkg/domain"
	"mayfly/pkg/project"
	"mayfly/pkg/tui/terminal"
	"mayfly/pkg/vault"
)


func setupTestViews(t *testing.T) (*Screens, *application.Service) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	regPath := filepath.Join(tempHome, ".mayfly", "projects.json")
	reg, err := project.NewRegistry(regPath)
	if err != nil {
		t.Fatal(err)
	}

	vaultPath := filepath.Join(tempHome, ".mayfly", "vault.enc")
	storage, err := vault.NewStorage(vaultPath, 1000)
	if err != nil {
		t.Fatal(err)
	}

	svc := application.NewService(application.Dependencies{
		Projects: reg,
		Vault:    storage,
	})

	screens := NewScreens(svc, tempHome)
	return screens, svc
}


func TestScreensNavigationAndDraw(t *testing.T) {
	screens, svc := setupTestViews(t)

	// Mode should initially be first-run setup if vault does not exist
	if screens.mode != ModeFirstRunSetup {
		t.Errorf("expected ModeFirstRunSetup, got %v", screens.mode)
	}

	// Initialize vault
	if err := svc.InitializeVault(context.Background(), []byte("testmasterpass")); err != nil {
		t.Fatal(err)
	}
	if err := svc.UnlockVault(context.Background(), []byte("testmasterpass")); err != nil {
		t.Fatal(err)
	}

	// Switch to Global Projects dashboard
	screens.mode = ModeGlobalProjects
	screens.reloadProjects()

	// Draw frame
	sz := terminal.Size{Rows: 24, Columns: 80}
	frame := terminal.NewFrame(sz)
	screens.Draw(frame)

	// Test mode switch to Scan via 's'
	screens.HandleKey(terminal.KeyEvent{Type: terminal.KeyRune, Rune: 's'})
	if screens.mode != ModeScan {
		t.Errorf("expected ModeScan after pressing 's', got %v", screens.mode)
	}

	// Escape back to Global Projects
	screens.HandleKey(terminal.KeyEvent{Type: terminal.KeyEscape})
	if screens.mode != ModeGlobalProjects {
		t.Errorf("expected ModeGlobalProjects after Escape, got %v", screens.mode)
	}

	// Test mode switch to Audit via 'a'
	screens.HandleKey(terminal.KeyEvent{Type: terminal.KeyRune, Rune: 'a'})
	if screens.mode != ModeAudit {
		t.Errorf("expected ModeAudit after pressing 'a', got %v", screens.mode)
	}

	// Escape back to Global Projects
	screens.HandleKey(terminal.KeyEvent{Type: terminal.KeyEscape})
	if screens.mode != ModeGlobalProjects {
		t.Errorf("expected ModeGlobalProjects after Escape, got %v", screens.mode)
	}

	// Test Status Bar setter and Tick
	screens.SetStatus("Testing status")
	screens.Tick()

	// Test quit on 'q'
	quit := screens.HandleKey(terminal.KeyEvent{Type: terminal.KeyRune, Rune: 'q'})
	if !quit {
		t.Errorf("expected HandleKey('q') to return true (quit)")
	}
}

func TestScreensProjectScopedMode(t *testing.T) {
	screens, svc := setupTestViews(t)
	_ = svc.InitializeVault(context.Background(), []byte("testmasterpass"))
	_ = svc.UnlockVault(context.Background(), []byte("testmasterpass"))

	proj := domain.Project{
		ID:            "test-proj-id",
		CanonicalPath: "/path/to/test-proj",
	}

	screens.SetProjectScoped(proj)
	if screens.mode != ModeProjectSecrets {
		t.Errorf("expected ModeProjectSecrets for project-scoped view, got %v", screens.mode)
	}
	if screens.selProject.ID != proj.ID {
		t.Errorf("expected selProject.ID to match, got %v", screens.selProject.ID)
	}
}

func TestScreensRevealSelectionOnly(t *testing.T) {
	screens, svc := setupTestViews(t)
	ctx := context.Background()
	_ = svc.InitializeVault(ctx, []byte("testmasterpass"))
	_ = svc.UnlockVault(ctx, []byte("testmasterpass"))

	projDir := filepath.Join(t.TempDir(), "test-app")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}

	proj, err := svc.RegisterProject(ctx, projDir)
	if err != nil {
		t.Fatal(err)
	}



	_ = svc.SetSecret(ctx, proj.ID, "API_KEY", "secret_api_123")
	_ = svc.SetSecret(ctx, proj.ID, "DB_PASS", "super_secret_db")

	screens.SetProjectScoped(proj)
	screens.reloadSecrets()

	// Initially all secrets must be masked
	for _, item := range screens.secretsList.Items {
		if item.Secondary != "••••••••••••••••" {
			t.Errorf("expected masked secret initially, got %q for %s", item.Secondary, item.Primary)
		}
	}

	// Press 'v' to reveal the currently selected secret (first item: API_KEY)
	screens.HandleKey(terminal.KeyEvent{Type: terminal.KeyRune, Rune: 'v'})

	// Verify ONLY API_KEY is revealed and DB_PASS remains masked
	var apiKeyFound, dbPassFound bool
	for _, item := range screens.secretsList.Items {
		if item.Primary == "API_KEY" {
			apiKeyFound = true
			if item.Secondary != "secret_api_123" {
				t.Errorf("expected API_KEY to be revealed as 'secret_api_123', got %q", item.Secondary)
			}
		}
		if item.Primary == "DB_PASS" {
			dbPassFound = true
			if item.Secondary != "••••••••••••••••" {
				t.Errorf("expected DB_PASS to remain masked, but got revealed %q", item.Secondary)
			}
		}
	}

	if !apiKeyFound || !dbPassFound {
		t.Fatalf("expected both API_KEY and DB_PASS to exist in list")
	}

	// Press 'v' again to toggle mask back
	screens.HandleKey(terminal.KeyEvent{Type: terminal.KeyRune, Rune: 'v'})
	for _, item := range screens.secretsList.Items {
		if item.Secondary != "••••••••••••••••" {
			t.Errorf("expected masked secret after toggle, got %q for %s", item.Secondary, item.Primary)
		}
	}
}

