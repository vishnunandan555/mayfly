package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mayfly/domain"
)

func newRegistry(t *testing.T) *Registry {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metadata", "projects.json")
	registry, err := NewRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func makeDirectory(t *testing.T, parent, name string) string {
	t.Helper()
	path := filepath.Join(parent, name)
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSameBasenameDifferentPathsAreIsolated(t *testing.T) {
	parent := t.TempDir()
	first := makeDirectory(t, parent, filepath.Join("one", "service"))
	second := makeDirectory(t, parent, filepath.Join("two", "service"))
	registry := newRegistry(t)
	firstProject, created, err := registry.Initialize(first)
	if err != nil || !created {
		t.Fatalf("first init = %#v, %v, created:%v", firstProject, err, created)
	}
	secondProject, created, err := registry.Initialize(second)
	if err != nil || !created {
		t.Fatalf("second init = %#v, %v, created:%v", secondProject, err, created)
	}
	if firstProject.ID == secondProject.ID {
		t.Fatal("different projects with the same basename received the same ID")
	}
	if err := os.Mkdir(filepath.Join(first, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	got, err := registry.Discover(context.Background(), filepath.Join(first, "nested"))
	if err != nil || got.ID != firstProject.ID {
		t.Fatalf("nested discovery = %#v, %v", got, err)
	}
	got, err = registry.Discover(context.Background(), second)
	if err != nil || got.ID != secondProject.ID {
		t.Fatalf("second discovery = %#v, %v", got, err)
	}
}

func TestSpacesUnicodeAndSymlinksResolveToTheSameProject(t *testing.T) {
	parent := t.TempDir()
	root := makeDirectory(t, parent, "Project with 空格")
	nested := makeDirectory(t, root, "内側")
	registry := newRegistry(t)
	want, _, err := registry.Initialize(root)
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "project-link")
	if err := os.Symlink(root, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	got, err := registry.Discover(context.Background(), filepath.Join(link, "内側"))
	if err != nil || got.ID != want.ID || got.Path != root {
		t.Fatalf("symlink discovery = %#v, %v; want %#v", got, err, want)
	}
	canonical, err := CanonicalRoot(link)
	if err != nil || canonical != root {
		t.Fatalf("canonical symlink root = %q, %v; want %q", canonical, err, root)
	}
	if _, err := registry.Discover(context.Background(), nested); err != nil {
		t.Fatal(err)
	}
}

func TestRepeatedInitIsIdempotent(t *testing.T) {
	root := makeDirectory(t, t.TempDir(), "project")
	registry := newRegistry(t)
	first, created, err := registry.Initialize(root)
	if err != nil || !created {
		t.Fatalf("first init = %#v, %v, created:%v", first, err, created)
	}
	second, created, err := registry.Initialize(root)
	if err != nil || created || second.ID != first.ID {
		t.Fatalf("repeated init = %#v, %v, created:%v", second, err, created)
	}
}

func TestMovedProjectKeepsIdentityAndRecreatedProjectDoesNot(t *testing.T) {
	parent := t.TempDir()
	oldRoot := makeDirectory(t, parent, "old-name")
	registry := newRegistry(t)
	original, _, err := registry.Initialize(oldRoot)
	if err != nil {
		t.Fatal(err)
	}
	movedRoot := filepath.Join(parent, "new-name")
	if err := os.Rename(oldRoot, movedRoot); err != nil {
		t.Fatal(err)
	}
	moved, err := registry.Discover(context.Background(), movedRoot)
	if err != nil || moved.ID != original.ID || moved.Path != movedRoot {
		t.Fatalf("moved discovery = %#v, %v; want ID %q/path %q", moved, err, original.ID, movedRoot)
	}
	refreshed, err := registry.Get(context.Background(), original.ID)
	if err != nil || refreshed.Path != movedRoot {
		t.Fatalf("moved registry path = %#v, %v; want %q", refreshed, err, movedRoot)
	}
	if err := os.RemoveAll(movedRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(movedRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Discover(context.Background(), movedRoot); !errors.Is(err, ErrProjectNotInitialized) {
		t.Fatalf("recreated discovery error = %v, want not initialized", err)
	}
	recreated, _, err := registry.Initialize(movedRoot)
	if err != nil {
		t.Fatal(err)
	}
	if recreated.ID == original.ID {
		t.Fatal("deleted and recreated project reused the old project ID")
	}
}

func TestRootAndOutsideHomePaths(t *testing.T) {
	rootID, err := ProjectIDForRoot(string(filepath.Separator))
	if err != nil || !strings.HasPrefix(string(rootID), "project-") {
		t.Fatalf("root project ID = %q, %v", rootID, err)
	}
	root, err := CanonicalRoot(string(filepath.Separator))
	if err != nil || root != string(filepath.Separator) {
		t.Fatalf("canonical root = %q, %v", root, err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("home directory unavailable: %v", err)
	}
	outside := t.TempDir()
	if relative, relErr := filepath.Rel(home, outside); relErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Skip("temporary directory is inside the home directory")
	}
	registry := newRegistry(t)
	if _, _, err := registry.Initialize(outside); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidAndNonexistentPathsAndRegistryPlacement(t *testing.T) {
	registry := newRegistry(t)
	if _, _, err := registry.Initialize(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, ErrProjectPath) {
		t.Fatalf("missing path error = %v, want ErrProjectPath", err)
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("not a project"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := registry.Initialize(file); !errors.Is(err, ErrProjectPath) {
		t.Fatalf("file path error = %v, want ErrProjectPath", err)
	}
	root := makeDirectory(t, t.TempDir(), "project")
	inside, err := NewRegistry(filepath.Join(root, ".mayfly", "projects.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := inside.Initialize(root); !errors.Is(err, ErrRegistryInsideProject) {
		t.Fatalf("inside registry error = %v, want ErrRegistryInsideProject", err)
	}
	if _, err := registry.Discover(context.Background(), root); !errors.Is(err, ErrProjectNotInitialized) {
		t.Fatalf("uninitialized discovery error = %v, want ErrProjectNotInitialized", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := registry.Discover(ctx, root); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled discovery error = %v", err)
	}
	if _, err := NewRegistry(""); !errors.Is(err, ErrInvalidRegistry) {
		t.Fatalf("empty registry error = %v, want ErrInvalidRegistry", err)
	}
}

func TestMalformedRegistryIsRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"projects":[`), 0600); err != nil {
		t.Fatal(err)
	}
	registry, err := NewRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	root := makeDirectory(t, t.TempDir(), "project")
	if _, err := registry.Discover(context.Background(), root); !errors.Is(err, ErrInvalidRegistry) {
		t.Fatalf("malformed registry error = %v, want ErrInvalidRegistry", err)
	}
}

func TestProjectIdentityIsDeterministic(t *testing.T) {
	root := makeDirectory(t, t.TempDir(), "project")
	first, err := ProjectIDForRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ProjectIDForRoot(root)
	if err != nil || first != second {
		t.Fatalf("IDs = %q/%q, %v", first, second, err)
	}
	if err := (domain.Project{ID: first, Name: "project", Path: root}).Validate(); err != nil {
		t.Fatal(err)
	}
}
