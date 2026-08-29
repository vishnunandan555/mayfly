// Package project discovers initialized MayFly projects without depending on
// Git, shell commands, or project-local marker files.
package project

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mayfly/application"
	"mayfly/domain"
)

var (
	ErrProjectPath           = errors.New("project: invalid project path")
	ErrProjectNotInitialized = application.ErrProjectNotInitialized
	ErrRegistryInsideProject = errors.New("project: registry must be outside the project")
	ErrInvalidRegistry       = errors.New("project: invalid registry")
)

// CanonicalRoot applies MayFly's project-root policy:
//
//   - the input must exist and be a directory;
//   - relative paths are made absolute;
//   - lexical components are cleaned;
//   - symlinks are resolved to their existing target.
//
// A symlink used to invoke MayFly therefore refers to the target project. A
// moved directory keeps its identity on Linux because identity uses its
// filesystem device/inode pair rather than its old path.
func CanonicalRoot(path string) (string, error) {
	return canonicalPath(path, false)
}

func canonicalPath(path string, allowFile bool) (string, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, '\x00') {
		return "", ErrProjectPath
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", ErrProjectPath
	}
	abs = filepath.Clean(abs)
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() && !allowFile {
		return "", ErrProjectPath
	}
	if !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", ErrProjectPath
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return "", ErrProjectPath
	}
	canonical = filepath.Clean(canonical)
	canonicalInfo, err := os.Stat(canonical)
	if err != nil || !canonicalInfo.IsDir() {
		return "", ErrProjectPath
	}
	return canonical, nil
}

// ProjectIDForRoot returns the deterministic ID for an existing canonical
// project root. On Linux the ID is derived from the filesystem identity of
// the directory (device and inode), which is stable across renames and moves
// on the same filesystem and changes when a directory is deleted/recreated.
// Unsupported platforms use the canonical path as a conservative fallback.
func ProjectIDForRoot(root string) (domain.ProjectID, error) {
	canonical, err := CanonicalRoot(root)
	if err != nil {
		return "", err
	}
	return projectID(canonical), nil
}

func projectID(canonicalRoot string) domain.ProjectID {
	identity := filesystemIdentity(canonicalRoot)
	if identity == "" {
		identity = "path:" + canonicalRoot
	}
	return projectIDFromIdentity(identity)
}

func projectIDFromIdentity(identity string) domain.ProjectID {
	digest := sha256.Sum256([]byte(identity))
	return domain.ProjectID("project-" + hex.EncodeToString(digest[:]))
}

func projectName(root string) string {
	name := filepath.Base(root)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return root
	}
	return name
}

func registryPathInsideRoot(registryPath, root string) bool {
	absolute, err := filepath.Abs(registryPath)
	if err != nil {
		return true
	}
	relative, err := filepath.Rel(root, filepath.Clean(absolute))
	if err != nil {
		return true
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative))
}

func wrapPathError(message string, err error) error {
	if err == nil {
		return errors.New(message)
	}
	return fmt.Errorf("%s: %w", message, err)
}
