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
}


func osMkdir(p string) error {
	return os.MkdirAll(p, 0755)
}
