package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mayfly/pkg/domain"
)

var (
	ErrProjectAlreadyExists = domain.ErrProjectExists
	ErrRegistryMissing      = errors.New("project: registry file missing")
)

// Registry manages the project definitions stored on disk in ~/.mayfly/projects.json.
type Registry struct {
	mu          sync.RWMutex
	path        string
	cache       map[string]domain.Project
	cacheLoaded bool
}

func DefaultRegistryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mayfly", "projects.json"), nil
}

func NewRegistry(path string) (*Registry, error) {
	if path == "" {
		var err error
		path, err = DefaultRegistryPath()
		if err != nil {
			return nil, err
		}
	}
	return &Registry{path: path}, nil
}

func (r *Registry) Path() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.path
}

func (r *Registry) loadLocked() (map[string]domain.Project, error) {
	// Return from in-memory cache if populated.
	if r.cacheLoaded {
		// Return a shallow copy so callers can't mutate the cache directly.
		copy := make(map[string]domain.Project, len(r.cache))
		for k, v := range r.cache {
			copy[k] = v
		}
		return copy, nil
	}

	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			r.cache = make(map[string]domain.Project)
			r.cacheLoaded = true
			return make(map[string]domain.Project), nil
		}
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	var projects map[string]domain.Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}
	if projects == nil {
		projects = make(map[string]domain.Project)
	}

	// Populate cache.
	r.cache = projects
	r.cacheLoaded = true

	// Return a copy.
	copy := make(map[string]domain.Project, len(projects))
	for k, v := range projects {
		copy[k] = v
	}
	return copy, nil
}

func (r *Registry) saveLocked(projects map[string]domain.Project) error {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, ".mayfly-projects-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := os.Chmod(tmpName, 0600); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, r.path); err != nil {
		return err
	}

	// Update cache to reflect the newly saved state.
	r.cache = make(map[string]domain.Project, len(projects))
	for k, v := range projects {
		r.cache[k] = v
	}
	r.cacheLoaded = true

	return nil
}

// InvalidateCache clears the in-memory registry cache, forcing the next read to reload from disk.
// Primarily useful in tests.
func (r *Registry) InvalidateCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cacheLoaded = false
	r.cache = nil
}

// Register adds a new project directory to the registry.
func (r *Registry) Register(dirPath string) (domain.Project, error) {
	identity, err := InspectDirectory(dirPath)
	if err != nil {
		return domain.Project{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	projects, err := r.loadLocked()
	if err != nil {
		return domain.Project{}, err
	}

	now := time.Now().UTC()
	if existing, found := projects[identity.ID]; found {
		// Update path and timestamp
		existing.CanonicalPath = identity.CanonicalPath
		existing.UpdatedAt = now
		projects[identity.ID] = existing
		if err := r.saveLocked(projects); err != nil {
			return domain.Project{}, err
		}
		return existing, nil
	}

	project := domain.Project{
		ID:            identity.ID,
		CanonicalPath: identity.CanonicalPath,
		Device:        identity.Device,
		Inode:         identity.Inode,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	projects[identity.ID] = project
	if err := r.saveLocked(projects); err != nil {
		return domain.Project{}, err
	}

	return project, nil
}

// Resolve identifies the registered project corresponding to a given folder path.
func (r *Registry) Resolve(dirPath string) (domain.Project, error) {
	identity, err := InspectDirectory(dirPath)
	if err != nil {
		return domain.Project{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	projects, err := r.loadLocked()
	if err != nil {
		return domain.Project{}, err
	}

	// 1. Direct match by ID
	if proj, found := projects[identity.ID]; found {
		return proj, nil
	}

	// 2. Match by canonical path
	for _, proj := range projects {
		if proj.CanonicalPath == identity.CanonicalPath {
			return proj, nil
		}
	}

	return domain.Project{}, ErrProjectNotFound
}

// Get retrieves a project by its unique ID.
func (r *Registry) Get(projectID string) (domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projects, err := r.loadLocked()
	if err != nil {
		return domain.Project{}, err
	}

	proj, found := projects[projectID]
	if !found {
		return domain.Project{}, ErrProjectNotFound
	}
	return proj, nil
}

// List returns all registered projects.
func (r *Registry) List() ([]domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projects, err := r.loadLocked()
	if err != nil {
		return nil, err
	}

	list := make([]domain.Project, 0, len(projects))
	for _, p := range projects {
		list = append(list, p)
	}
	return list, nil
}

// AllMap returns a map of all projects (used for snapshots and TUI views).
func (r *Registry) AllMap() (map[string]domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.loadLocked()
}

// ImportMap overrides the registry with an imported map.
func (r *Registry) ImportMap(imported map[string]domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveLocked(imported)
}

// Delete removes a project from the registry.
func (r *Registry) Delete(projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	projects, err := r.loadLocked()
	if err != nil {
		return err
	}

	if _, found := projects[projectID]; !found {
		return ErrProjectNotFound
	}

	delete(projects, projectID)
	return r.saveLocked(projects)
}

// MigrateProject re-points a project from oldDirPath to newDirPath.
func (r *Registry) MigrateProject(oldDirPath, newDirPath string) (domain.Project, domain.Project, error) {
	newIdentity, err := InspectDirectory(newDirPath)
	if err != nil {
		return domain.Project{}, domain.Project{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	projects, err := r.loadLocked()
	if err != nil {
		return domain.Project{}, domain.Project{}, err
	}

	// Find the old project
	var oldID string
	var oldProj domain.Project
	found := false

	oldCanonical, err := ResolveDirectory(oldDirPath)
	if err == nil {
		for id, p := range projects {
			if p.CanonicalPath == oldCanonical {
				oldID = id
				oldProj = p
				found = true
				break
			}
		}
	}

	if !found {
		// Try matching by raw ID
		if p, ok := projects[oldDirPath]; ok {
			oldID = oldDirPath
			oldProj = p
			found = true
		}
	}

	if !found {
		return domain.Project{}, domain.Project{}, ErrProjectNotFound
	}

	now := time.Now().UTC()
	newProj := domain.Project{
		ID:            newIdentity.ID,
		CanonicalPath: newIdentity.CanonicalPath,
		Device:        newIdentity.Device,
		Inode:         newIdentity.Inode,
		CreatedAt:     oldProj.CreatedAt,
		UpdatedAt:     now,
	}

	delete(projects, oldID)
	projects[newIdentity.ID] = newProj

	if err := r.saveLocked(projects); err != nil {
		return domain.Project{}, domain.Project{}, err
	}

	return oldProj, newProj, nil
}
