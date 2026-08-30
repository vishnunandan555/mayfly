//go:build linux

package project

import (
	"fmt"
	"os"
	"syscall"
)

func filesystemIdentity(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return ""
	}
	return fmt.Sprintf("linux:%d:%d", stat.Dev, stat.Ino)
}
