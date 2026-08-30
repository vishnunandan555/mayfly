//go:build unix || linux || darwin

package terminal

import (
	"os"
	"os/signal"
	"syscall"
)

func NotifyResize(ch chan os.Signal) {
	signal.Notify(ch, syscall.SIGWINCH)
}
