//go:build windows

package terminal

import "os"

func NotifyResize(ch chan os.Signal) {
	// Windows console resize is received directly through console input buffer
}
