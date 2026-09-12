package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	regPath := filepath.Join(tmpDir, "projects.json")

	reg, err := NewRegistry(regPath)
	if err != nil {
		t.Fatal(err)
	}

	projDir := filepath.Join(tmpDir, "app1")
	if err := osMkdir(projDir); err != nil {
		t.Fatal(err)
	}

	// 1. Register
	proj, err := reg.Register(projDir)
	if err != nil {
		t.Fatal(err)
	}
	if proj.ID == "" {
		t.Fatal("expected non-empty project ID")
	}

	// 2. Resolve
	resolved, err := reg.Resolve(projDir)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ID != proj.ID {
		t.Fatalf("expected project %s, got %s", proj.ID, resolved.ID)
	}

	// 3. List
	list, err := reg.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 project, got %d", len(list))
	}

	// 4. Migrate
	newProjDir := filepath.Join(tmpDir, "app1-moved")
	if err := osMkdir(newProjDir); err != nil {
		t.Fatal(err)
	}

	oldP, newP, err := reg.MigrateProject(projDir, newProjDir)
	if err != nil {
		t.Fatal(err)
	}
	if oldP.ID != proj.ID {
		t.Fatalf("expected old ID %s, got %s", proj.ID, oldP.ID)
	}
	expectedPath, err := filepath.EvalSymlinks(newProjDir)
	if err != nil {
		expectedPath = newProjDir
	}
	if newP.CanonicalPath != expectedPath {
		t.Fatalf("expected new path %s, got %s", expectedPath, newP.CanonicalPath)
	}
	if newP.ID != proj.ID {
		t.Fatalf("expected migration to preserve project ID %s, got %s", proj.ID, newP.ID)
	}
}

func TestProjectRegistryDoesNotMatchReplacementDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := NewRegistry(filepath.Join(tmpDir, "projects.json"))
	if err != nil {
		t.Fatal(err)
	}

	projDir := filepath.Join(tmpDir, "app")
	if err := osMkdir(projDir); err != nil {
		t.Fatal(err)
	}
	proj, err := reg.Register(projDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(projDir); err != nil {
		t.Fatal(err)
	}
	if err := osMkdir(projDir); err != nil {
		t.Fatal(err)
	}

	if _, err := reg.Resolve(projDir); err != ErrProjectNotFound {
		t.Fatalf("expected replacement directory to be unregistered, got %v", err)
	}
	if replacement, err := reg.Register(projDir); err != nil {
		t.Fatal(err)
	} else if replacement.ID == proj.ID {
		t.Fatal("replacement directory reused the deleted project's identity")
	}
}

func TestProjectRegistryIdentitySurvivesMove(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := NewRegistry(filepath.Join(tmpDir, "projects.json"))
	if err != nil {
		t.Fatal(err)
	}

	oldDir := filepath.Join(tmpDir, "old")
	newDir := filepath.Join(tmpDir, "new")
	if err := osMkdir(oldDir); err != nil {
		t.Fatal(err)
	}
	proj, err := reg.Register(oldDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		t.Fatal(err)
	}

	resolved, err := reg.Resolve(newDir)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ID != proj.ID {
		t.Fatalf("expected moved project ID %s, got %s", proj.ID, resolved.ID)
	}
}

func osMkdir(p string) error {
	return os.MkdirAll(p, 0755)
}
