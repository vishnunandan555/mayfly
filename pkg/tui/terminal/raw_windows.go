//go:build windows

package terminal

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
)

const (
	enableProcessedInput        = 0x0001
	enableLineInput             = 0x0002
	enableEchoInput             = 0x0004
	enableWindowInput           = 0x0008
	enableMouseInput            = 0x0010
	enableVirtualTerminalInput  = 0x0200
	enableProcessedOutput       = 0x0001
	enableWrapAtEOLOutput       = 0x0002
	enableVirtualTerminalOutput = 0x0004
)

type State struct {
	inMode  uint32
	outMode uint32
}

// EnableRaw configures the Windows console for raw ANSI input and output.
func EnableRaw(f *os.File) (*State, error) {
	hIn := syscall.Handle(os.Stdin.Fd())
	hOut := syscall.Handle(os.Stdout.Fd())

	var inMode, outMode uint32
	r1, _, err := procGetConsoleMode.Call(uintptr(hIn), uintptr(unsafe.Pointer(&inMode)))
	if r1 == 0 {
		return nil, err
	}
	r1, _, err = procGetConsoleMode.Call(uintptr(hOut), uintptr(unsafe.Pointer(&outMode)))
	if r1 == 0 {
		return nil, err
	}

	rawIn := inMode &^ (enableLineInput | enableEchoInput | enableProcessedInput | enableMouseInput)
	rawIn |= enableVirtualTerminalInput | enableWindowInput

	rawOut := outMode | enableProcessedOutput | enableVirtualTerminalOutput

	r1, _, err = procSetConsoleMode.Call(uintptr(hIn), uintptr(rawIn))
	if r1 == 0 {
		return nil, err
	}
	r1, _, err = procSetConsoleMode.Call(uintptr(hOut), uintptr(rawOut))
	if r1 == 0 {
		return nil, err
	}

	return &State{inMode: inMode, outMode: outMode}, nil
}

// Restore restores previous console modes on Windows.
func Restore(f *os.File, state *State) error {
	if state == nil {
		return nil
	}
	hIn := syscall.Handle(os.Stdin.Fd())
	hOut := syscall.Handle(os.Stdout.Fd())

	procSetConsoleMode.Call(uintptr(hIn), uintptr(state.inMode))
	procSetConsoleMode.Call(uintptr(hOut), uintptr(state.outMode))
	return nil
}
