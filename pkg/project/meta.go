package project

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	maxFailedAttempts = 5
	lockoutDuration   = 30 * time.Second
)

// VaultMeta tracks vault access metadata stored in ~/.mayfly/meta.json.
// It is used to implement a soft brute-force lockout after repeated bad passwords.
type VaultMeta struct {
	FailedAttempts int       `json:"failed_attempts"`
	LastFailedAt   time.Time `json:"last_failed_at,omitempty"`
	LockedUntil    time.Time `json:"locked_until,omitempty"`
}

// MetaStore manages the vault meta.json file on disk.
type MetaStore struct {
	mu   sync.Mutex
	path string
}

// DefaultMetaPath returns the default path for the vault meta file.
func DefaultMetaPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mayfly", "meta.json"), nil
}

// NewMetaStore creates a new MetaStore. If path is empty, uses the default.
func NewMetaStore(path string) (*MetaStore, error) {
	if path == "" {
		var err error
		path, err = DefaultMetaPath()
		if err != nil {
			return nil, err
		}
	}
	return &MetaStore{path: path}, nil
}

// Read loads the current VaultMeta from disk. Returns zero value if file doesn't exist.
func (m *MetaStore) Read() (VaultMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readLocked()
}

func (m *MetaStore) readLocked() (VaultMeta, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return VaultMeta{}, nil
		}
		return VaultMeta{}, err
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var meta VaultMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		// If meta file is corrupt, start fresh rather than blocking the user.
		return VaultMeta{}, nil
	}
	return meta, nil
}

func (m *MetaStore) writeLocked(meta VaultMeta) error {
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".mayfly-meta-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := os.Chmod(tmpName, 0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, m.path)
}

// IsLocked returns true and the time remaining if the vault is currently locked out.
func (m *MetaStore) IsLocked() (bool, time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	meta, err := m.readLocked()
	if err != nil || meta.LockedUntil.IsZero() {
		return false, 0
	}

	remaining := time.Until(meta.LockedUntil)
	if remaining > 0 {
		return true, remaining
	}
	return false, 0
}

// RecordFailedAttempt increments the failure counter and enforces lockout after maxFailedAttempts.
func (m *MetaStore) RecordFailedAttempt() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	meta, err := m.readLocked()
	if err != nil {
		meta = VaultMeta{}
	}

	meta.FailedAttempts++
	meta.LastFailedAt = time.Now().UTC()

	if meta.FailedAttempts >= maxFailedAttempts {
		meta.LockedUntil = time.Now().UTC().Add(lockoutDuration)
	}

	return m.writeLocked(meta)
}

// RecordSuccess resets the failure counter and clears any lockout.
func (m *MetaStore) RecordSuccess() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.writeLocked(VaultMeta{})
}
