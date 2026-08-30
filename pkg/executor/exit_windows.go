//go:build windows

package executor

import (
	"os/exec"
)

func extractExitCode(err *exec.ExitError) int {
	if err == nil {
		return 0
	}
	return err.ExitCode()
}
