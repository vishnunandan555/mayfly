package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"mayfly/application"
	"mayfly/domain"
)

// Registry stores project metadata outside project trees. It contains no
// secret values. A single registry file avoids turning user-controlled paths
// into filenames.
type Registry struct {
	path string
}

type registryFile struct {
	Version  int                 `json:"version"`
	Projects []registeredProject `json:"projects"`
}

type registeredProject struct {
	ID       string `json:"id"`
	Root     string `json:"root"`
	Identity string `json:"identity"`
	Name     string `json:"name"`
}

const registryVersion = 1

// NewRegistry constructs an inert registry. It does not create or read files.
func NewRegistry(path string) (*Registry, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, '\x00') {
		return nil, ErrInvalidRegistry
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, ErrInvalidRegistry
	}
	return &Registry{path: filepath.Clean(abs)}, nil
}

// DefaultRegistryPath places MayFly metadata under the user's home directory
// in ~/.mayfly/projects.json. Init rejects this location if it would be inside
// the selected project, such as when the home directory itself is the project.
func DefaultRegistryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", ErrInvalidRegistry
	}
	return filepath.Join(home, ".mayfly", "projects.json"), nil
}

// Initialize idempotently registers an existing project. It returns whether a
// new entry was created. Repeated init of the same filesystem identity returns
// the same project ID. A moved project is recognized by its identity and its
// current canonical root is recorded.
func (r *Registry) Initialize(path string) (domain.Project, bool, error) {
	if r == nil {
		return domain.Project{}, false, ErrInvalidRegistry
	}
	root, err := canonicalPath(path, false)
	if err != nil {
		return domain.Project{}, false, err
	}
	if registryPathInsideRoot(r.path, root) {
		return domain.Project{}, false, ErrRegistryInsideProject
	}
	id := projectID(root)
	identity := filesystemIdentity(root)
	if identity == "" {
		identity = "path:" + root
	}
	projects, err := r.read()
	if err != nil {
		return domain.Project{}, false, err
	}
	for index := range projects {
		if projects[index].Identity == identity || projects[index].ID == string(id) {
			projects[index].Root = root
			projects[index].Name = projectName(root)
			if err := r.write(projects); err != nil {
				return domain.Project{}, false, err
			}
			return toDomain(projects[index]), false, nil
		}
	}
	entry := registeredProject{ID: string(id), Root: root, Identity: identity, Name: projectName(root)}
	projects = append(projects, entry)
	if err := r.write(projects); err != nil {
		return domain.Project{}, false, err
	}
	return toDomain(entry), true, nil
}

// Discover resolves path and returns the initialized project containing that
// exact directory. Nested directories are accepted. A different project with
// the same basename cannot match because identity and registry root are used.
func (r *Registry) Discover(ctx context.Context, path string) (domain.Project, error) {
	if err := contextError(ctx); err != nil {
		return domain.Project{}, err
	}
	if r == nil {
		return domain.Project{}, ErrInvalidRegistry
	}
	root, err := canonicalPath(path, true)
	if err != nil {
		return domain.Project{}, err
	}
	projects, err := r.read()
	if err != nil {
		return domain.Project{}, err
	}
	for candidate := root; ; candidate = filepath.Dir(candidate) {
		candidateIdentity := filesystemIdentity(candidate)
		for index, entry := range projects {
			matchesIdentity := candidateIdentity != "" && entry.Identity == candidateIdentity
			matchesFallbackPath := candidateIdentity == "" && entry.Root == candidate
			if matchesIdentity || matchesFallbackPath {
				if entry.Root != candidate {
					entry.Root = candidate
					entry.Name = projectName(candidate)
					projects[index] = entry
					if err := r.write(projects); err != nil {
						return domain.Project{}, err
					}
				}
				project := toDomain(entry)
				project.Path = candidate
				return project, nil
			}
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
	}
	return domain.Project{}, ErrProjectNotInitialized
}

func (r *Registry) Current(ctx context.Context) (domain.Project, error) {
	if err := contextError(ctx); err != nil {
		return domain.Project{}, err
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return domain.Project{}, ErrProjectNotInitialized
	}
	return r.Discover(ctx, workingDirectory)
}

func (r *Registry) Get(ctx context.Context, id domain.ProjectID) (domain.Project, error) {
	if err := contextError(ctx); err != nil {
		return domain.Project{}, err
	}
	if err := id.Validate(); err != nil {
		return domain.Project{}, err
	}
	projects, err := r.read()
	if err != nil {
		return domain.Project{}, err
	}
	for _, entry := range projects {
		if entry.ID == string(id) {
			return toDomain(entry), nil
		}
	}
	return domain.Project{}, ErrProjectNotInitialized
}

func (r *Registry) read() ([]registeredProject, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, wrapPathError("project: read registry", err)
	}
	var file registryFile
	if err := json.Unmarshal(data, &file); err != nil || file.Version != registryVersion {
		return nil, ErrInvalidRegistry
	}
	seen := make(map[string]struct{}, len(file.Projects))
	for _, entry := range file.Projects {
		id := domain.ProjectID(entry.ID)
		if entry.ID == "" || entry.Root == "" || entry.Identity == "" || entry.Name == "" || filepath.Clean(entry.Root) != entry.Root || !filepath.IsAbs(entry.Root) || !validSafeText(entry.Identity) {
			return nil, ErrInvalidRegistry
		}
		if err := id.Validate(); err != nil || string(projectIDFromIdentity(entry.Identity)) != entry.ID {
			return nil, ErrInvalidRegistry
		}
		if _, exists := seen[entry.ID]; exists {
			return nil, ErrInvalidRegistry
		}
		seen[entry.ID] = struct{}{}
	}
	return file.Projects, nil
}

func validSafeText(value string) bool {
	for _, r := range value {
		if r == '\x00' || r == '\n' || r == '\r' || r == '\t' {
			return false
		}
	}
	return true
}

func (r *Registry) write(projects []registeredProject) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0700); err != nil {
		return wrapPathError("project: create registry directory", err)
	}
	payload, err := json.MarshalIndent(registryFile{Version: registryVersion, Projects: projects}, "", "  ")
	if err != nil {
		return errors.New("project: encode registry")
	}
	payload = append(payload, '\n')
	return writeAtomic(r.path, payload)
}

func writeAtomic(path string, data []byte) (err error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".mayfly-projects-*")
	if err != nil {
		return wrapPathError("project: create temporary registry", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err = temporary.Chmod(0600); err != nil {
		_ = temporary.Close()
		return errors.New("project: set registry permissions")
	}
	if _, err = temporary.Write(data); err != nil {
		_ = temporary.Close()
		return errors.New("project: write registry")
	}
	if err = temporary.Sync(); err != nil {
		_ = temporary.Close()
		return errors.New("project: sync registry")
	}
	if err = temporary.Close(); err != nil {
		return errors.New("project: close registry")
	}
	if err = os.Rename(temporaryPath, path); err != nil {
		return errors.New("project: replace registry")
	}
	return nil
}

func toDomain(entry registeredProject) domain.Project {
	return domain.Project{ID: domain.ProjectID(entry.ID), Name: entry.Name, Path: entry.Root}
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

var _ application.ProjectLookup = (*Registry)(nil)
