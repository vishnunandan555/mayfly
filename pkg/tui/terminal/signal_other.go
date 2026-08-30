//go:build !unix && !linux && !darwin && !windows

package terminal

import "os"

func NotifyResize(ch chan os.Signal) {}
