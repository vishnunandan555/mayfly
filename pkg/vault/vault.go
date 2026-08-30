package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"mayfly/pkg/domain"
)

var (
	ErrVaultExists    = errors.New("vault: vault already initialized")
	ErrVaultMissing   = errors.New("vault: vault file does not exist")
	ErrWrongPassword  = domain.ErrWrongPassword
	ErrVaultLocked    = domain.ErrVaultLocked
	ErrCorruptPayload = errors.New("vault: decrypted payload is corrupt")
)

// StorageRecord represents the decrypted in-memory vault contents.
type StorageRecord struct {
	Version   int                                `json:"version"`
	UpdatedAt time.Time                          `json:"updated_at"`
	Projects  map[string]map[domain.SecretName]string `json:"projects"`
}

// Storage manages the encrypted vault file on disk.
type Storage struct {
	mu         sync.RWMutex
	path       string
	iterations int
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mayfly", "vault.enc"), nil
}

func NewStorage(path string, iterations int) (*Storage, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	if iterations <= 0 {
		iterations = DefaultIterations
	}
	return &Storage{
		path:       path,
		iterations: iterations,
	}, nil
}

func (s *Storage) Path() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

func (s *Storage) Exists() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, err := os.Stat(s.path)
	return err == nil
}

// Initialize creates a fresh, empty encrypted vault on disk.
func (s *Storage) Initialize(password []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.path); err == nil {
		return ErrVaultExists
	}

	initial := StorageRecord{
		Version:   CurrentVersion,
		UpdatedAt: time.Now().UTC(),
		Projects:  make(map[string]map[domain.SecretName]string),
	}

	return s.saveLocked(initial, password)
}

// Open decrypts and returns the StorageRecord using the master password.
func (s *Storage) Open(password []byte) (StorageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, err := os.Stat(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return StorageRecord{}, ErrVaultMissing
		}
		return StorageRecord{}, err
	}
	// Self-heal permissions to 0600 if overly permissive
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		_ = os.Chmod(s.path, 0600)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return StorageRecord{}, err
	}

	header, offset, err := UnmarshalHeader(data)
	if err != nil {
		return StorageRecord{}, err
	}

	key, err := DeriveKey(password, header.Salt, int(header.Iterations), 32)
	if err != nil {
		return StorageRecord{}, err
	}
	defer clearBytes(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return StorageRecord{}, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return StorageRecord{}, err
	}

	headerBytes := data[:offset]
	ciphertext := data[offset:]

	plaintext, err := gcm.Open(nil, header.Nonce, ciphertext, headerBytes)
	if err != nil {
		return StorageRecord{}, ErrWrongPassword
	}
	defer clearBytes(plaintext)

	var record StorageRecord
	if err := json.Unmarshal(plaintext, &record); err != nil {
		return StorageRecord{}, ErrCorruptPayload
	}

	if record.Projects == nil {
		record.Projects = make(map[string]map[domain.SecretName]string)
	}

	return record, nil
}

// Save encrypts and atomically writes the StorageRecord to disk.
func (s *Storage) Save(record StorageRecord, password []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(record, password)
}

func (s *Storage) saveLocked(record StorageRecord, password []byte) error {
	record.UpdatedAt = time.Now().UTC()
	plaintext, err := json.Marshal(record)
	if err != nil {
		return err
	}
	defer clearBytes(plaintext)

	salt := make([]byte, DefaultSaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}

	nonce := make([]byte, DefaultNonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	header := Header{
		Version:    CurrentVersion,
		KDFID:      KDFPBKDF2SHA256,
		Iterations: uint32(s.iterations),
		SaltLen:    uint16(len(salt)),
		NonceLen:   uint8(len(nonce)),
		Salt:       salt,
		Nonce:      nonce,
	}
	copy(header.Magic[:], MagicBytes)

	headerBytes, err := header.MarshalBinary()
	if err != nil {
		return err
	}

	key, err := DeriveKey(password, salt, s.iterations, 32)
	if err != nil {
		return err
	}
	defer clearBytes(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, headerBytes)

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, ".mayfly-vault-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if err := os.Chmod(tmpName, 0600); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if _, err := tmpFile.Write(headerBytes); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if _, err := tmpFile.Write(ciphertext); err != nil {
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

	return os.Rename(tmpName, s.path)
}

// ExportSnapshot creates an encrypted backup snapshot file.
func (s *Storage) ExportSnapshot(targetPath string, projects map[string]domain.Project) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	snapshot := domain.BackupSnapshot{
		Version:   CurrentVersion,
		CreatedAt: time.Now().UTC(),
		Projects:  projects,
		Payload:   data,
	}

	outData, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0700); err != nil {
		return err
	}

	return os.WriteFile(targetPath, outData, 0600)
}

// ImportSnapshot restores an encrypted backup snapshot file.
func (s *Storage) ImportSnapshot(snapshotPath string) (map[string]domain.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return nil, err
	}

	var snapshot domain.BackupSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidBackupFile, err)
	}

	if len(snapshot.Payload) < 15 {
		return nil, domain.ErrInvalidBackupFile
	}

	// Verify header format
	if _, _, err := UnmarshalHeader(snapshot.Payload); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidBackupFile, err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	tmpFile, err := os.CreateTemp(dir, ".mayfly-restore-tmp-*")
	if err != nil {
		return nil, err
	}
	tmpName := tmpFile.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := os.Chmod(tmpName, 0600); err != nil {
		_ = tmpFile.Close()
		return nil, err
	}

	if _, err := tmpFile.Write(snapshot.Payload); err != nil {
		_ = tmpFile.Close()
		return nil, err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return nil, err
	}
	if err := tmpFile.Close(); err != nil {
		return nil, err
	}

	if err := os.Rename(tmpName, s.path); err != nil {
		return nil, err
	}

	return snapshot.Projects, nil
}

func clearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}
