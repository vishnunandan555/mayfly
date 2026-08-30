//go:build linux

package terminal

import (
	"os"
	"syscall"
	"unsafe"
)

type State struct {
	termios syscall.Termios
}

// EnableRaw puts the terminal into raw character-by-character mode on Linux.
func EnableRaw(f *os.File) (*State, error) {
	fd := f.Fd()
	var oldState State
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		syscall.TCGETS,
		uintptr(unsafe.Pointer(&oldState.termios)),
	)
	if err != 0 {
		return nil, err
	}

	raw := oldState.termios
	// Disable ECHO, ICANON, ISIG, IEXTEN
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	// Disable ICRNL, IXON
	raw.Iflag &^= syscall.ICRNL | syscall.IXON | syscall.BRKINT | syscall.INPCK | syscall.ISTRIP
	// Disable OPOST
	raw.Oflag &^= syscall.OPOST
	// Set 8-bit chars
	raw.Cflag |= syscall.CS8
	// Minimum 1 byte, no timeout
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	_, _, err = syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		syscall.TCSETS,
		uintptr(unsafe.Pointer(&raw)),
	)
	if err != 0 {
		return nil, err
	}

	return &oldState, nil
}

// Restore restores the previous terminal settings on Linux.
func Restore(f *os.File, state *State) error {
	if state == nil {
		return nil
	}
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		syscall.TCSETS,
		uintptr(unsafe.Pointer(&state.termios)),
	)
	if err != 0 {
		return err
	}
	return nil
}
