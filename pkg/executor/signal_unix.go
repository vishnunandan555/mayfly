//go:build !windows

package executor

import (
	"os"
	"os/exec"
)

func configureProcess(cmd *exec.Cmd) {}

func terminateChild(cmd *exec.Cmd, sig os.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Signal(sig)
}
