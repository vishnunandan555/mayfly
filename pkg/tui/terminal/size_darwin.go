//go:build darwin

package terminal

import (
	"os"
	"syscall"
	"unsafe"
)

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

const ioctlTIOCGWINSZ = 0x40087468

func GetSize(f *os.File) (Size, error) {
	var ws winsize
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		ioctlTIOCGWINSZ,
		uintptr(unsafe.Pointer(&ws)),
	)
	if err != 0 || ws.Row == 0 || ws.Col == 0 {
		return Size{Rows: 24, Columns: 80}, nil
	}
	return Size{Rows: int(ws.Row), Columns: int(ws.Col)}, nil
}
