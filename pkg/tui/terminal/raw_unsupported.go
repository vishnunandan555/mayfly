//go:build !linux && !darwin && !windows

package terminal

import "os"

type State struct{}

func EnableRaw(f *os.File) (*State, error) {
	return &State{}, nil
}

func Restore(f *os.File, state *State) error {
	return nil
}
