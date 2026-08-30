//go:build !linux && !darwin && !windows

package terminal

import "os"

func GetSize(f *os.File) (Size, error) {
	return Size{Rows: 24, Columns: 80}, nil
}
