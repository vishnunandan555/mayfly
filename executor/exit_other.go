//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package executor

import "os"

func processExitCode(state *os.ProcessState) int {
	if state == nil || state.ExitCode() < 0 {
		return 1
	}
	return state.ExitCode()
}
