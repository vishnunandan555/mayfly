//go:build linux

package screen

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// winsize is the Linux kernel structure used by TIOCGWINSZ. Keeping this
// small definition local avoids a dependency on a terminal library.
type winsize struct {
	rows    uint16
	columns uint16
	xpixels uint16
	ypixels uint16
}

// TerminalSize returns the current size of file using the Linux terminal
// ioctl. It does not invoke stty, tput, or any other executable.
func TerminalSize(file *os.File) (Size, error) {
	if file == nil {
		return Size{}, fmt.Errorf("%w: nil file", ErrTerminalSizeUnsupported)
	}
	var size winsize
	if err := ioctlWinsize(file.Fd(), syscall.TIOCGWINSZ, &size); err != nil {
		return Size{}, fmt.Errorf("%w: %v", ErrTerminalSizeUnsupported, err)
	}
	return Size{Rows: int(size.rows), Columns: int(size.columns)}, nil
}

func ioctlWinsize(fd uintptr, request uintptr, size *winsize) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, request, uintptr(unsafe.Pointer(size)), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func resizeSignal() os.Signal { return syscall.SIGWINCH }
