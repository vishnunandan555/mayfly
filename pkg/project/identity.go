package project

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mayfly/pkg/domain"
)

var (
	ErrInvalidPath     = domain.ErrInvalidProjectPath
	ErrProjectNotFound = domain.ErrProjectNotFound
)

// Identity represents the unique deterministic hardware/filesystem identity of a folder.
type Identity struct {
	ID            string
	CanonicalPath string
	Device        uint64
	Inode         uint64
}

const projectIdentityFile = ".mayfly-project-id"

func readProjectID(canonicalPath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(canonicalPath, projectIdentityFile))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	id := strings.TrimSpace(string(data))
	if id == "" {
		return "", nil
	}
	return id, nil
}

func writeProjectID(canonicalPath, id string) error {
	return os.WriteFile(filepath.Join(canonicalPath, projectIdentityFile), []byte(id+"\n"), 0600)
}

func newProjectID() (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "project-" + hex.EncodeToString(random), nil
}

func ensureProjectID(identity Identity, id string) error {
	if id == "" {
		return fmt.Errorf("project: empty project identity")
	}
	return writeProjectID(identity.CanonicalPath, id)
}

// ResolveDirectory converts any directory path to its canonical, symlink-resolved absolute path.
func ResolveDirectory(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		// If symlink resolution fails but path exists, fallback to abs
		canonical = abs
	}

	info, err := os.Stat(canonical)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: path %s is not a directory", ErrInvalidPath, canonical)
	}

	return canonical, nil
}

// GenerateID is retained for compatibility with older registry entries.
func GenerateID(device, inode uint64, canonicalPath string) string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%d:%d:%s", device, inode, canonicalPath))
	return "project-" + hex.EncodeToString(sum[:])
}
