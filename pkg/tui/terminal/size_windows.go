//go:build windows

package terminal

import (
	"os"
	"syscall"
	"unsafe"
)

var procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")

type coord struct {
	X int16
	Y int16
}

type smallRect struct {
	Left   int16
	Top    int16
	Right  int16
	Bottom int16
}

type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

func GetSize(f *os.File) (Size, error) {
	hOut := syscall.Handle(os.Stdout.Fd())
	var csbi consoleScreenBufferInfo
	r1, _, _ := procGetConsoleScreenBufferInfo.Call(uintptr(hOut), uintptr(unsafe.Pointer(&csbi)))
	if r1 == 0 {
		return Size{Rows: 24, Columns: 80}, nil
	}

	rows := int(csbi.Window.Bottom - csbi.Window.Top + 1)
	cols := int(csbi.Window.Right - csbi.Window.Left + 1)
	if rows <= 0 || cols <= 0 {
		return Size{Rows: 24, Columns: 80}, nil
	}
	return Size{Rows: rows, Columns: cols}, nil
}
