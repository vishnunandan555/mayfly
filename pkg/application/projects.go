package application

import (
	"context"
	"fmt"

	"mayfly/pkg/domain"
)

// Projects returns all currently registered project workspaces.
func (s *Service) Projects() ([]domain.Project, error) {
	if s.projects == nil {
		return nil, ErrProjectNotFound
	}
	return s.projects.List()
}

// ResolveCurrentProject identifies the registered project workspace corresponding to the given path.
func (s *Service) ResolveCurrentProject(dir string) (domain.Project, error) {
	if s.projects == nil {
		return domain.Project{}, ErrProjectNotFound
	}
	return s.projects.Resolve(dir)
}

// RegisterProject registers a new directory path as an encrypted project workspace.
func (s *Service) RegisterProject(ctx context.Context, dir string) (domain.Project, error) {
	if s.projects == nil {
		return domain.Project{}, ErrProjectNotFound
	}
	proj, err := s.projects.Register(dir)
	if err != nil {
		return domain.Project{}, err
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionProjectInit, proj.ID, "", "", nil)
	}

	return proj, nil
}

// DeleteProject unregisters a project workspace and wipes its associated encrypted secrets.
func (s *Service) DeleteProject(ctx context.Context, projectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.projects != nil {
		if err := s.projects.Delete(projectID); err != nil {
			return err
		}
	}

	if s.isUnlocked && s.vault != nil {
		delete(s.activeSecret.Projects, projectID)
		if err := s.vault.Save(s.activeSecret, s.password); err != nil {
			return err
		}
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionProjectDeleted, projectID, "", "", nil)
	}

	return nil
}

// MigrateProject updates project paths when a directory moves to a new location.
func (s *Service) MigrateProject(ctx context.Context, oldDir, newDir string) (domain.Project, domain.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.projects == nil {
		return domain.Project{}, domain.Project{}, ErrProjectNotFound
	}

	oldProj, newProj, err := s.projects.MigrateProject(oldDir, newDir)
	if err != nil {
		return domain.Project{}, domain.Project{}, err
	}

	// Migrate secrets map entry to the new project ID
	if s.isUnlocked && s.vault != nil {
		if secrets, ok := s.activeSecret.Projects[oldProj.ID]; ok {
			s.activeSecret.Projects[newProj.ID] = secrets
			delete(s.activeSecret.Projects, oldProj.ID)
			_ = s.vault.Save(s.activeSecret, s.password)
		}
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionProjectMigrated, newProj.ID, "", fmt.Sprintf("%s -> %s", oldDir, newDir), nil)
	}

	return oldProj, newProj, nil
}
