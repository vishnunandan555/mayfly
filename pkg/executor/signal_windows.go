//go:build windows

package executor

import (
	"os"
	"os/exec"
	"syscall"
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	generateConsoleCtrlEv = kernel32.NewProc("GenerateConsoleCtrlEvent")
)

func configureProcess(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func terminateChild(cmd *exec.Cmd, sig os.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if sig == nil {
		return cmd.Process.Kill()
	}

	if sig == os.Interrupt || sig == syscall.SIGINT || sig == syscall.SIGTERM {
		if _, _, err := generateConsoleCtrlEv.Call(uintptr(syscall.CTRL_BREAK_EVENT), uintptr(cmd.Process.Pid)); err != nil && err != syscall.Errno(0) {
			return cmd.Process.Kill()
		}
		return nil
	}
	return cmd.Process.Kill()
}
