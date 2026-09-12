//go:build darwin

package project

import (
	"os"
	"syscall"
)

// InspectDirectory resolves the filesystem identity on macOS / Darwin systems.
func InspectDirectory(path string) (Identity, error) {
	canonical, err := ResolveDirectory(path)
	if err != nil {
		return Identity{}, err
	}

	info, err := os.Stat(canonical)
	if err != nil {
		return Identity{}, err
	}
	id, err := readProjectID(canonical)
	if err != nil {
		return Identity{}, err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return Identity{
			ID:            id,
			CanonicalPath: canonical,
		}, nil
	}

	dev := uint64(stat.Dev)
	ino := uint64(stat.Ino)

	return Identity{
		ID:            id,
		CanonicalPath: canonical,
		Device:        dev,
		Inode:         ino,
	}, nil
}
