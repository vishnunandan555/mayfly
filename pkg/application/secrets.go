package application

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"strings"

	"mayfly/pkg/domain"
)


// ListSecrets returns the list of secrets for a specific project.
func (s *Service) ListSecrets(projectID string) ([]domain.Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isUnlocked {
		return nil, ErrVaultLocked
	}

	projMap, ok := s.activeSecret.Projects[projectID]
	if !ok {
		return []domain.Secret{}, nil
	}

	var list []domain.Secret
	for k, v := range projMap {
		list = append(list, domain.Secret{
			Name:  k,
			Value: v,
		})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list, nil
}


// GetSecret retrieves a single decrypted secret value for the requested project and key name.
func (s *Service) GetSecret(ctx context.Context, projectID string, name domain.SecretName) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isUnlocked {
		return "", ErrVaultLocked
	}

	projMap, ok := s.activeSecret.Projects[projectID]
	if !ok {
		return "", ErrSecretNotFound
	}

	val, found := projMap[name]
	if !found {
		return "", ErrSecretNotFound
	}

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionSecretGet, projectID, string(name), "", nil)
	}

	return val, nil
}

// SetSecret adds or updates a secret key-value pair in the encrypted vault.
func (s *Service) SetSecret(ctx context.Context, projectID string, name domain.SecretName, value string) error {
	if err := name.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isUnlocked {
		return ErrVaultLocked
	}

	if s.activeSecret.Projects == nil {
		s.activeSecret.Projects = make(map[string]map[domain.SecretName]string)
	}

	if _, ok := s.activeSecret.Projects[projectID]; !ok {
		s.activeSecret.Projects[projectID] = make(map[domain.SecretName]string)
	}

	s.activeSecret.Projects[projectID][name] = value

	if err := s.vault.Save(s.activeSecret, s.password); err != nil {
		return err
	}

	s.resetAutoLockLocked()

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionSecretSet, projectID, string(name), "", nil)
	}

	return nil
}

// DeleteSecret removes a secret from the encrypted project space.
func (s *Service) DeleteSecret(ctx context.Context, projectID string, name domain.SecretName) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isUnlocked {
		return ErrVaultLocked
	}

	projMap, ok := s.activeSecret.Projects[projectID]
	if !ok {
		return ErrSecretNotFound
	}

	if _, found := projMap[name]; !found {
		return ErrSecretNotFound
	}

	delete(projMap, name)
	if err := s.vault.Save(s.activeSecret, s.password); err != nil {
		return err
	}

	s.resetAutoLockLocked()

	if s.auditor != nil {
		_ = s.auditor.Record(ctx, domain.ActionSecretDeleted, projectID, string(name), "", nil)
	}

	return nil
}

// ImportEnv parses .env formatted file contents and stores the extracted secrets in the project vault.
func (s *Service) ImportEnv(ctx context.Context, projectID string, content string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isUnlocked {
		return 0, ErrVaultLocked
	}

	if s.activeSecret.Projects == nil {
		s.activeSecret.Projects = make(map[string]map[domain.SecretName]string)
	}
	if _, ok := s.activeSecret.Projects[projectID]; !ok {
		s.activeSecret.Projects[projectID] = make(map[domain.SecretName]string)
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	count := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Strip surrounding single or double quotes
		if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
			(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
			if len(val) >= 2 {
				val = val[1 : len(val)-1]
			}
		}

		secName := domain.SecretName(key)
		if err := secName.Validate(); err != nil {
			continue
		}

		s.activeSecret.Projects[projectID][secName] = val
		count++
	}

	if count > 0 {
		if err := s.vault.Save(s.activeSecret, s.password); err != nil {
			return 0, err
		}
		if s.auditor != nil {
			_ = s.auditor.Record(ctx, domain.ActionSecretImported, projectID, fmt.Sprintf("%d secrets", count), "", nil)
		}
	}

	s.resetAutoLockLocked()
	return count, nil
}
