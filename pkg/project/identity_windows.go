//go:build windows

package project

import (
	"strings"
	"syscall"
)

// InspectDirectory resolves the filesystem identity on Windows.
func InspectDirectory(path string) (Identity, error) {
	canonical, err := ResolveDirectory(path)
	if err != nil {
		return Identity{}, err
	}

	// Normalize windows path (lowercase for case-insensitive Windows filesystems)
	normalized := strings.ToLower(canonical)

	var dev uint64 = 0
	var ino uint64 = 0

	// Attempt to query Windows File ID via GetFileInformationByHandle
	pathPtr, err := syscall.UTF16PtrFromString(canonical)
	if err == nil {
		h, err := syscall.CreateFile(
			pathPtr,
			0,
			syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
			nil,
			syscall.OPEN_EXISTING,
			syscall.FILE_FLAG_BACKUP_SEMANTICS,
			0,
		)
		if err == nil {
			defer syscall.CloseHandle(h)
			var fi syscall.ByHandleFileInformation
			if err := syscall.GetFileInformationByHandle(h, &fi); err == nil {
				dev = uint64(fi.VolumeSerialNumber)
				ino = (uint64(fi.FileIndexHigh) << 32) | uint64(fi.FileIndexLow)
			}
		}
	}

	id := GenerateID(dev, ino, normalized)

	return Identity{
		ID:            id,
		CanonicalPath: canonical,
		Device:        dev,
		Inode:         ino,
	}, nil
}
