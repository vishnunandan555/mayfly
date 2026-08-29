//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package executor

import (
	"os"
	"syscall"
)

func processExitCode(state *os.ProcessState) int {
	if state == nil {
		return 1
	}
	if code := state.ExitCode(); code >= 0 {
		return code
	}
	if status, ok := state.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return 1
}
