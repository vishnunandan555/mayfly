package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mayfly/pkg/project"
	"mayfly/pkg/vault"
)

func TestApplicationRotatePasswordAndImport(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	regPath := filepath.Join(tmpDir, "projects.json")
	reg, err := project.NewRegistry(regPath)
	if err != nil {
		t.Fatal(err)
	}

	vaultPath := filepath.Join(tmpDir, "vault.enc")
	vStorage, err := vault.NewStorage(vaultPath, 1000)
	if err != nil {
		t.Fatal(err)
	}

	svc := NewService(Dependencies{
		Projects: reg,
		Vault:    vStorage,
	})

	// 1. Initialize vault
	masterPass := []byte("masterpass_123")
	if err := svc.InitializeVault(ctx, masterPass); err != nil {
		t.Fatal(err)
	}

	// 2. Register a project
	projDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}
	proj, err := svc.RegisterProject(ctx, projDir)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Import .env
	envContent := "API_KEY=\"sk_test_12345\"\nDATABASE_URL=postgres://localhost/mydb\n# comment\n"
	count, err := svc.ImportEnv(ctx, proj.ID, envContent)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 imported secrets, got %d", count)
	}

	val, err := svc.GetSecret(ctx, proj.ID, "API_KEY")
	if err != nil || val != "sk_test_12345" {
		t.Fatalf("failed to retrieve imported secret: %v, val=%s", err, val)
	}

	// 4. Rotate Master Password
	newPass := []byte("brand_new_masterpass_456")
	if err := svc.RotatePassword(ctx, masterPass, newPass); err != nil {
		t.Fatal(err)
	}

	// 5. Lock and reopen with new password
	svc.LockVault()
	if svc.IsUnlocked() {
		t.Fatalf("vault should be locked")
	}

	// Old password should fail
	if err := svc.UnlockVault(ctx, masterPass); err == nil {
		t.Fatalf("expected old password to fail")
	}

	// New password should succeed
	if err := svc.UnlockVault(ctx, newPass); err != nil {
		t.Fatalf("new password failed to unlock vault: %v", err)
	}

	val2, err := svc.GetSecret(ctx, proj.ID, "DATABASE_URL")
	if err != nil || val2 != "postgres://localhost/mydb" {
		t.Fatalf("failed to retrieve secret after rotation: %v, val=%s", err, val2)
	}
}

func TestApplicationAutoLockTimer(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	vaultPath := filepath.Join(tmpDir, "vault.enc")
	vStorage, err := vault.NewStorage(vaultPath, 1000)
	if err != nil {
		t.Fatal(err)
	}

	svc := NewService(Dependencies{
		Vault: vStorage,
	})

	masterPass := []byte("autolock_pass")
	if err := svc.InitializeVault(ctx, masterPass); err != nil {
		t.Fatal(err)
	}

	// Set short timeout: 50 milliseconds
	svc.SetAutoLockTimeout(50 * time.Millisecond)
	if !svc.IsUnlocked() {
		t.Fatalf("expected vault to be unlocked")
	}

	time.Sleep(120 * time.Millisecond)

	if svc.IsUnlocked() {
		t.Fatalf("expected vault to be automatically locked after timeout")
	}
}
