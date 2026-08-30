package project

import (
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

// GenerateID produces a deterministic SHA-256 project ID from device, inode, and canonical path.
func GenerateID(device, inode uint64, canonicalPath string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s", device, inode, canonicalPath)))
	return "project-" + hex.EncodeToString(sum[:])
}
