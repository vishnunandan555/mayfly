//go:build unix || linux || darwin

package executor

import (
	"os/exec"
	"syscall"
)

func extractExitCode(err *exec.ExitError) int {
	if err == nil {
		return 0
	}
	if ws, ok := err.Sys().(syscall.WaitStatus); ok {
		if ws.Exited() {
			return ws.ExitStatus()
		}
		if ws.Signaled() {
			return 128 + int(ws.Signal())
		}
	}
	return err.ExitCode()
}
