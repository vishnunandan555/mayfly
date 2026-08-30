//go:build darwin

package terminal

import (
	"os"
	"syscall"
	"unsafe"
)

type State struct {
	termios syscall.Termios
}

const (
	ioctlTIOCGETA = 0x40487413
	ioctlTIOCSETA = 0x80487414
)

// EnableRaw puts the terminal into raw mode on macOS.
func EnableRaw(f *os.File) (*State, error) {
	fd := f.Fd()
	var oldState State
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		ioctlTIOCGETA,
		uintptr(unsafe.Pointer(&oldState.termios)),
	)
	if err != 0 {
		return nil, err
	}

	raw := oldState.termios
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Iflag &^= syscall.ICRNL | syscall.IXON | syscall.BRKINT | syscall.INPCK | syscall.ISTRIP
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	_, _, err = syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		ioctlTIOCSETA,
		uintptr(unsafe.Pointer(&raw)),
	)
	if err != 0 {
		return nil, err
	}

	return &oldState, nil
}

// Restore restores previous terminal settings on macOS.
func Restore(f *os.File, state *State) error {
	if state == nil {
		return nil
	}
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		ioctlTIOCSETA,
		uintptr(unsafe.Pointer(&state.termios)),
	)
	if err != 0 {
		return err
	}
	return nil
}
