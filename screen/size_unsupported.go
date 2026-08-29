//go:build !linux

package screen

import "os"

// TerminalSize is unavailable on this build without importing a platform
// terminal package. Callers can pass an explicit Size to NewApplication.
func TerminalSize(*os.File) (Size, error) { return Size{}, ErrTerminalSizeUnsupported }

func resizeSignal() os.Signal { return nil }
