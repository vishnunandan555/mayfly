//go:build linux

package project

import (
	"os"
	"syscall"
)

// InspectDirectory resolves the filesystem identity on Linux systems using Stat_t.
func InspectDirectory(path string) (Identity, error) {
	canonical, err := ResolveDirectory(path)
	if err != nil {
		return Identity{}, err
	}

	info, err := os.Stat(canonical)
	if err != nil {
		return Identity{}, err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return Identity{
			ID:            GenerateID(0, 0, canonical),
			CanonicalPath: canonical,
		}, nil
	}

	dev := uint64(stat.Dev)
	ino := uint64(stat.Ino)

	return Identity{
		ID:            GenerateID(dev, ino, canonical),
		CanonicalPath: canonical,
		Device:        dev,
		Inode:         ino,
	}, nil
}
