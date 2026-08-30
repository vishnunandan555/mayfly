//go:build !linux && !darwin && !windows

package terminal

import (
	"bufio"
	"os"
	"strings"
)

type State struct{}

func EnableRaw(f *os.File) (*State, error) {
	return &State{}, nil
}

func Restore(f *os.File, state *State) error {
	return nil
}

func IsTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	stat, err := f.Stat()
	return err == nil && (stat.Mode()&os.ModeCharDevice) != 0
}

func ReadPassword(f *os.File) ([]byte, error) {
	if f == nil {
		return nil, os.ErrInvalid
	}
	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		return []byte(strings.TrimRight(scanner.Text(), "\r\n")), nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, os.ErrInvalid
}

