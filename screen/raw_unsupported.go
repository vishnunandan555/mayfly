//go:build !linux

package screen

import "os"

func enterRawMode(file *os.File) (func() error, error) {
	return nil, ErrRawModeUnsupported
}

func additionalRawSignals() []os.Signal {
	return nil
}
