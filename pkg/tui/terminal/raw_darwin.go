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

// IsTerminal checks if the given file descriptor refers to a terminal on macOS.
func IsTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	var termios syscall.Termios
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		ioctlTIOCGETA,
		uintptr(unsafe.Pointer(&termios)),
	)
	return err == 0
}

// ReadPassword reads a line from a terminal with echo disabled on macOS.
func ReadPassword(f *os.File) ([]byte, error) {
	if f == nil {
		return nil, os.ErrInvalid
	}
	fd := f.Fd()
	var oldState syscall.Termios
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		ioctlTIOCGETA,
		uintptr(unsafe.Pointer(&oldState)),
	)
	if err != 0 {
		return nil, err
	}

	noEcho := oldState
	noEcho.Lflag &^= syscall.ECHO
	noEcho.Lflag |= syscall.ICANON | syscall.ISIG

	_, _, err = syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		ioctlTIOCSETA,
		uintptr(unsafe.Pointer(&noEcho)),
	)
	if err != 0 {
		return nil, err
	}

	defer func() {
		_, _, _ = syscall.Syscall(
			syscall.SYS_IOCTL,
			fd,
			ioctlTIOCSETA,
			uintptr(unsafe.Pointer(&oldState)),
		)
	}()

	var buf [1]byte
	var pass []byte
	for {
		n, rErr := f.Read(buf[:])
		if n > 0 {
			b := buf[0]
			if b == '\n' || b == '\r' {
				break
			}
			pass = append(pass, b)
		}
		if rErr != nil {
			if len(pass) > 0 {
				break
			}
			return nil, rErr
		}
	}
	return pass, nil
}

