//go:build linux

package screen

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// Linux raw mode uses the kernel termios ioctls directly. This is deliberately
// Linux-specific: the standard library exposes syscall, but it does not offer
// one portable raw-terminal API. Other operating systems return
// ErrRawModeUnsupported from their platform implementation.
func enterRawMode(file *os.File) (func() error, error) {
	if file == nil {
		return nil, fmt.Errorf("%w: nil file", ErrRawModeUnsupported)
	}

	var original syscall.Termios
	if err := ioctlTermios(file.Fd(), syscall.TCGETS, &original); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRawModeUnsupported, err)
	}

	raw := original
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP |
		syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	raw.Cc[syscall.VMIN] = 0
	raw.Cc[syscall.VTIME] = 1

	if err := ioctlTermios(file.Fd(), syscall.TCSETS, &raw); err != nil {
		return nil, fmt.Errorf("screen: set raw terminal mode: %w", err)
	}
	return func() error {
		return ioctlTermios(file.Fd(), syscall.TCSETS, &original)
	}, nil
}

func (r *rawFileReader) Read(buffer []byte) (int, error) {
	if r == nil || r.file == nil {
		return 0, ErrInputClosed
	}
	return syscall.Read(int(r.file.Fd()), buffer)
}

func ioctlTermios(fd uintptr, request uintptr, termios *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, request, uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func additionalRawSignals() []os.Signal {
	return []os.Signal{syscall.SIGTERM}
}
