// Package vault provides MayFly's encrypted file vault.
//
// The file format is versioned and consists of an authenticated header plus
// an AES-256-GCM ciphertext. The JSON payload, including project names,
// secret names, metadata, and values, is encrypted. Only the format version,
// KDF parameters, salt, and nonce are stored outside the ciphertext.
package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"mayfly/application"
	"mayfly/domain"
)

const (
	defaultIterations uint32 = 600000
	minimumIterations uint32 = 100000
	maximumIterations uint32 = 5000000
	saltSize                 = 32
	keySize                  = 32
	maxVaultFileSize  int64  = 256 << 20
)

var (
	ErrVaultExists      = errors.New("vault: vault already exists")
	ErrVaultNotFound    = errors.New("vault: vault does not exist")
	ErrPasswordRequired = errors.New("vault: password is required")
	ErrWrongPassword    = errors.New("vault: wrong password or corrupt vault")
	ErrSecretNotFound   = errors.New("vault: secret not found")
	ErrProjectNotFound  = errors.New("vault: project not found")
	ErrVaultClosed      = errors.New("vault: vault is closed")
	ErrInvalidVaultPath = errors.New("vault: invalid vault path")
)

// Options controls construction of Storage. Random defaults to
// crypto/rand.Reader. Iterations is persisted in the vault header so future
// versions can increase the work factor without invalidating existing files.
type Options struct {
	Iterations uint32
	Random     io.Reader
}

// Storage owns the file location and initialization policy. It does not hold
// a password or derived key after an operation returns.
type Storage struct {
	path       string
	iterations uint32
	random     io.Reader
}

// NewStorage constructs an explicit file-backed vault boundary. It performs
// no I/O and does not create directories or files.
func NewStorage(path string, options Options) (*Storage, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, '\x00') {
		return nil, ErrInvalidVaultPath
	}
	if options.Iterations == 0 {
		options.Iterations = defaultIterations
	}
	if options.Iterations < minimumIterations || options.Iterations > maximumIterations {
		return nil, errors.New("vault: invalid PBKDF2 iteration count")
	}
	if options.Random == nil {
		options.Random = cryptorand.Reader
	}
	return &Storage{path: filepath.Clean(path), iterations: options.Iterations, random: options.Random}, nil
}

// Path returns the configured vault path. It contains no secret material.
func (s *Storage) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Initialize creates a new empty vault. It refuses to overwrite an existing
// file and writes only encrypted bytes to the temporary file used by the
// atomic-save routine.
func (s *Storage) Initialize(password []byte) error {
	if s == nil {
		return ErrInvalidVaultPath
	}
	if len(password) == 0 {
		return ErrPasswordRequired
	}
	if _, err := os.Stat(s.path); err == nil {
		return ErrVaultExists
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("vault: inspect vault: %w", err)
	}

	salt, err := randomBytes(s.random, saltSize)
	if err != nil {
		return errors.New("vault: generate encryption salt")
	}
	decrypted := diskPayload{Projects: []diskProject{}}
	serialized, err := s.encrypt(password, salt, s.iterations, decrypted)
	wipeBytes(salt)
	if err != nil {
		return err
	}
	return writeAtomic(s.path, serialized)
}

// Open unlocks and loads the vault as an application SecretService. The
// returned implementation keeps decrypted data only in memory for the life of
// the session. The password is not retained.
func (s *Storage) Open(ctx context.Context, password []byte) (application.SecretService, error) {
	vault, err := s.Unlock(ctx, password)
	if err != nil {
		return nil, err
	}
	return vault, nil
}

// Unlock loads and authenticates the vault, returning the concrete session
// type for callers that need Close or project-management helpers.
func (s *Storage) Unlock(ctx context.Context, password []byte) (*Vault, error) {
	if s == nil {
		return nil, ErrInvalidVaultPath
	}
	if len(password) == 0 {
		return nil, ErrPasswordRequired
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrVaultNotFound
		}
		return nil, fmt.Errorf("vault: read vault: %w", err)
	}
	if int64(len(data)) > maxVaultFileSize {
		return nil, ErrInvalidFormat
	}

	header, headerSize, err := parseHeader(data)
	if err != nil {
		return nil, err
	}
	if header.iterations < minimumIterations || header.iterations > maximumIterations {
		return nil, ErrInvalidFormat
	}
	if len(data) <= headerSize {
		return nil, ErrInvalidFormat
	}
	key := deriveKey(password, header.salt, int(header.iterations), keySize)
	payloadBytes, err := decrypt(key, header, data[headerSize:])
	if err != nil {
		wipeBytes(key)
		return nil, ErrWrongPassword
	}
	var payload diskPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		wipeBytes(key)
		wipeBytes(payloadBytes)
		return nil, ErrInvalidFormat
	}
	wipeBytes(payloadBytes)
	projects, err := decodePayload(payload)
	if err != nil {
		wipeBytes(key)
		return nil, err
	}

	return &Vault{
		path:       s.path,
		iterations: header.iterations,
		salt:       append([]byte(nil), header.salt...),
		key:        key,
		random:     s.random,
		projects:   projects,
	}, nil
}

// Initialize creates a vault using default policy.
func Initialize(path string, password []byte) error {
	storage, err := NewStorage(path, Options{})
	if err != nil {
		return err
	}
	return storage.Initialize(password)
}

// Unlock opens a vault using default policy.
func Unlock(ctx context.Context, path string, password []byte) (*Vault, error) {
	storage, err := NewStorage(path, Options{})
	if err != nil {
		return nil, err
	}
	return storage.Unlock(ctx, password)
}

// Load is an explicit alias for Unlock for callers that prefer storage
// terminology.
func Load(ctx context.Context, path string, password []byte) (*Vault, error) {
	return Unlock(ctx, path, password)
}

// Vault is an unlocked in-memory session backed by one encrypted file. The
// derived key is held only in this live session because encryption of updates
// requires it; it is never serialized. Call Close when the session is no
// longer needed. Go does not guarantee cryptographic erasure of strings or
// all memory copies after garbage collection.
type Vault struct {
	mu         sync.RWMutex
	path       string
	iterations uint32
	salt       []byte
	key        []byte
	random     io.Reader
	projects   map[domain.ProjectID]diskProject
	closed     bool
}

var _ application.VaultStorage = (*Storage)(nil)
var _ application.SecretService = (*Vault)(nil)

// Close clears the session's key and in-memory records. It is idempotent.
func (v *Vault) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return nil
	}
	wipeBytes(v.key)
	wipeBytes(v.salt)
	v.key = nil
	v.salt = nil
	v.projects = nil
	v.closed = true
	return nil
}

// Save persists the current decrypted session using a fresh GCM nonce.
func (v *Vault) Save(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.ensureOpen(); err != nil {
		return err
	}
	return v.saveLocked()
}

func (v *Vault) List(ctx context.Context, projectID domain.ProjectID) ([]domain.Secret, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if err := projectID.Validate(); err != nil {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if err := v.ensureOpen(); err != nil {
		return nil, err
	}
	project, ok := v.projects[projectID]
	if !ok {
		return []domain.Secret{}, nil
	}
	result := make([]domain.Secret, 0, len(project.Secrets))
	for _, secret := range project.Secrets {
		result = append(result, domain.Secret{
			ProjectID: projectID,
			Name:      domain.SecretName(secret.Name),
			Metadata:  secret.Metadata,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (v *Vault) Get(ctx context.Context, projectID domain.ProjectID, name domain.SecretName) (domain.SecretMaterial, error) {
	if err := contextError(ctx); err != nil {
		return domain.SecretMaterial{}, err
	}
	if err := projectID.Validate(); err != nil {
		return domain.SecretMaterial{}, err
	}
	if err := name.Validate(); err != nil {
		return domain.SecretMaterial{}, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if err := v.ensureOpen(); err != nil {
		return domain.SecretMaterial{}, err
	}
	project, ok := v.projects[projectID]
	if !ok {
		return domain.SecretMaterial{}, ErrSecretNotFound
	}
	for _, secret := range project.Secrets {
		if secret.Name == string(name) {
			return domain.SecretMaterial{Name: name, Value: secret.Value}, nil
		}
	}
	return domain.SecretMaterial{}, ErrSecretNotFound
}

func (v *Vault) Put(ctx context.Context, input domain.SecretInput) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := input.Validate(); err != nil {
		return err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.ensureOpen(); err != nil {
		return err
	}
	next := cloneProjects(v.projects)
	project := next[input.ProjectID]
	if project.ID == "" {
		project = diskProject{ID: string(input.ProjectID), Name: string(input.ProjectID)}
	}
	now := time.Now().UTC()
	found := false
	for index := range project.Secrets {
		if project.Secrets[index].Name == string(input.Name) {
			project.Secrets[index].Value = input.Value
			project.Secrets[index].Metadata.UpdatedAt = now
			found = true
			break
		}
	}
	if !found {
		project.Secrets = append(project.Secrets, diskSecret{
			Name: string(input.Name), Value: input.Value,
			Metadata: domain.SecretMetadata{CreatedAt: now, UpdatedAt: now},
		})
	}
	next[input.ProjectID] = project
	if err := v.saveProjectsLocked(next); err != nil {
		return err
	}
	v.projects = next
	return nil
}

func (v *Vault) Delete(ctx context.Context, projectID domain.ProjectID, name domain.SecretName) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := projectID.Validate(); err != nil {
		return err
	}
	if err := name.Validate(); err != nil {
		return err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.ensureOpen(); err != nil {
		return err
	}
	project, ok := v.projects[projectID]
	if !ok {
		return ErrSecretNotFound
	}
	index := -1
	for candidate, secret := range project.Secrets {
		if secret.Name == string(name) {
			index = candidate
			break
		}
	}
	if index < 0 {
		return ErrSecretNotFound
	}
	next := cloneProjects(v.projects)
	project = next[projectID]
	project.Secrets = append(project.Secrets[:index], project.Secrets[index+1:]...)
	next[projectID] = project
	if err := v.saveProjectsLocked(next); err != nil {
		return err
	}
	v.projects = next
	return nil
}

// PutProject stores non-sensitive project identity metadata. Secret CRUD can
// also create a project record with its ID when no metadata is available.
func (v *Vault) PutProject(ctx context.Context, project domain.Project) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := project.Validate(); err != nil {
		return err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.ensureOpen(); err != nil {
		return err
	}
	next := cloneProjects(v.projects)
	record := next[project.ID]
	record.ID = string(project.ID)
	record.Name = project.Name
	record.Path = project.Path
	next[project.ID] = record
	if err := v.saveProjectsLocked(next); err != nil {
		return err
	}
	v.projects = next
	return nil
}

// ListProjects returns only non-sensitive project metadata in stable order.
func (v *Vault) ListProjects(ctx context.Context) ([]domain.Project, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if err := v.ensureOpen(); err != nil {
		return nil, err
	}
	projects := make([]domain.Project, 0, len(v.projects))
	for _, project := range v.projects {
		projects = append(projects, domain.Project{ID: domain.ProjectID(project.ID), Name: project.Name, Path: project.Path})
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].ID < projects[j].ID })
	return projects, nil
}

func (v *Vault) ensureOpen() error {
	if v == nil || v.closed || len(v.key) != keySize || v.projects == nil {
		return ErrVaultClosed
	}
	return nil
}

func (v *Vault) saveLocked() error { return v.saveProjectsLocked(v.projects) }

func (v *Vault) saveProjectsLocked(projects map[domain.ProjectID]diskProject) error {
	data, err := encryptWithKey(v.key, v.salt, v.iterations, diskPayload{Projects: projectSlice(projects)}, v.random)
	if err != nil {
		return err
	}
	return writeAtomic(v.path, data)
}

type diskPayload struct {
	Projects []diskProject `json:"projects"`
}

type diskProject struct {
	ID      string       `json:"id"`
	Name    string       `json:"name,omitempty"`
	Path    string       `json:"path,omitempty"`
	Secrets []diskSecret `json:"secrets"`
}

type diskSecret struct {
	Name     string                `json:"name"`
	Value    string                `json:"value"`
	Metadata domain.SecretMetadata `json:"metadata"`
}

func (s *Storage) encrypt(password, salt []byte, iterations uint32, payload diskPayload) ([]byte, error) {
	key := deriveKey(password, salt, int(iterations), keySize)
	data, err := encryptWithKey(key, salt, iterations, payload, s.random)
	wipeBytes(key)
	return data, err
}

func encryptWithKey(key, salt []byte, iterations uint32, payload diskPayload, random io.Reader) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("vault: initialize encryption")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.New("vault: initialize authenticated encryption")
	}
	nonce, err := randomBytes(random, gcm.NonceSize())
	if err != nil {
		return nil, errors.New("vault: generate encryption nonce")
	}
	decrypted, err := json.Marshal(payload)
	if err != nil {
		wipeBytes(nonce)
		return nil, errors.New("vault: encode vault contents")
	}
	header := vaultHeader{version: formatVersion, kdf: kdfPBKDF2SHA256, iterations: iterations, salt: salt, nonce: nonce}
	headerBytes := header.marshal()
	ciphertext := gcm.Seal(nil, nonce, decrypted, headerBytes)
	result := make([]byte, 0, len(headerBytes)+len(ciphertext))
	result = append(result, headerBytes...)
	result = append(result, ciphertext...)
	wipeBytes(nonce)
	wipeBytes(decrypted)
	return result, nil
}

func decrypt(key []byte, header vaultHeader, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(header.nonce) != gcm.NonceSize() {
		return nil, ErrWrongPassword
	}
	return gcm.Open(nil, header.nonce, ciphertext, header.marshal())
}

func decodePayload(payload diskPayload) (map[domain.ProjectID]diskProject, error) {
	projects := make(map[domain.ProjectID]diskProject, len(payload.Projects))
	for _, project := range payload.Projects {
		id := domain.ProjectID(project.ID)
		if err := id.Validate(); err != nil || project.Name != "" && !validSafeText(project.Name) || project.Path != "" && !validSafeText(project.Path) {
			return nil, ErrInvalidFormat
		}
		if _, exists := projects[id]; exists {
			return nil, ErrInvalidFormat
		}
		seen := make(map[string]struct{}, len(project.Secrets))
		for _, secret := range project.Secrets {
			name := domain.SecretName(secret.Name)
			input := domain.SecretInput{ProjectID: id, Name: name, Value: secret.Value}
			if err := input.Validate(); err != nil {
				return nil, ErrInvalidFormat
			}
			if _, exists := seen[secret.Name]; exists {
				return nil, ErrInvalidFormat
			}
			seen[secret.Name] = struct{}{}
		}
		projects[id] = project
	}
	return projects, nil
}

func cloneProjects(source map[domain.ProjectID]diskProject) map[domain.ProjectID]diskProject {
	result := make(map[domain.ProjectID]diskProject, len(source))
	for id, project := range source {
		project.Secrets = append([]diskSecret(nil), project.Secrets...)
		result[id] = project
	}
	return result
}

func projectSlice(projects map[domain.ProjectID]diskProject) []diskProject {
	ids := make([]domain.ProjectID, 0, len(projects))
	for id := range projects {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	result := make([]diskProject, 0, len(ids))
	for _, id := range ids {
		project := projects[id]
		sort.Slice(project.Secrets, func(i, j int) bool { return project.Secrets[i].Name < project.Secrets[j].Name })
		result = append(result, project)
	}
	return result
}

func validSafeText(value string) bool {
	for _, r := range value {
		if r == '\x00' || r == '\n' || r == '\r' || r == '\t' {
			return false
		}
	}
	return true
}

func randomBytes(random io.Reader, size int) ([]byte, error) {
	if size <= 0 || random == nil {
		return nil, errors.New("vault: invalid random source")
	}
	buffer := make([]byte, size)
	if _, err := io.ReadFull(random, buffer); err != nil {
		wipeBytes(buffer)
		return nil, err
	}
	return buffer, nil
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

func wipeBytes(buffer []byte) {
	for index := range buffer {
		buffer[index] = 0
	}
}

func writeAtomic(path string, data []byte) (err error) {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return fmt.Errorf("vault: create vault directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".mayfly-vault-*")
	if err != nil {
		return fmt.Errorf("vault: create temporary vault: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err = temporary.Chmod(0600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("vault: set vault permissions: %w", err)
	}
	if _, err = temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("vault: write vault: %w", err)
	}
	if err = temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("vault: sync vault: %w", err)
	}
	if err = temporary.Close(); err != nil {
		return fmt.Errorf("vault: close temporary vault: %w", err)
	}
	if err = os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("vault: replace vault: %w", err)
	}
	// File contents are synchronized before rename. Directory synchronization
	// is best effort because support varies by filesystem/platform.
	if directoryFile, openErr := os.Open(directory); openErr == nil {
		_ = directoryFile.Sync()
		_ = directoryFile.Close()
	}
	return nil
}
