//go:build !linux

package screen

import "os"

func enterRawMode(file *os.File) (func() error, error) {
	return nil, ErrRawModeUnsupported
}

func (r *rawFileReader) Read([]byte) (int, error) { return 0, ErrRawModeUnsupported }

func additionalRawSignals() []os.Signal {
	return nil
}
