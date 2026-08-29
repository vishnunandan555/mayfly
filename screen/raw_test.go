package screen

import (
	"errors"
	"os"
	"testing"
)

type timeoutInputReader struct{}

func (timeoutInputReader) Read([]byte) (int, error) { return 0, nil }

func TestRawInputTimeoutDoesNotRestoreMode(t *testing.T) {
	restores := 0
	raw := &RawInput{
		input:   NewInput(timeoutInputReader{}),
		restore: func() error { restores++; return nil },
	}
	if _, err := raw.ReadEvent(); !errors.Is(err, ErrInputTimeout) {
		t.Fatalf("ReadEvent error = %v, want timeout", err)
	}
	if restores != 0 {
		t.Fatalf("timeout restored terminal %d times", restores)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	if restores != 1 {
		t.Fatalf("Close restored terminal %d times, want 1", restores)
	}
}

func TestRawModeRejectsNonTTYWithoutRequiringInteractiveTerminal(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()

	input, err := NewRawInput(reader)
	if err == nil {
		if closeErr := input.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		t.Skip("the test pipe was unexpectedly accepted as a terminal")
	}
	if !errors.Is(err, ErrRawModeUnsupported) {
		t.Fatalf("NewRawInput(pipe) error = %v, want ErrRawModeUnsupported", err)
	}
}

func TestRunRawRejectsNonTTYBeforeCallingFunction(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()

	called := false
	err = RunRaw(reader, func(Input) error {
		called = true
		return nil
	})
	if err == nil || !errors.Is(err, ErrRawModeUnsupported) {
		t.Fatalf("RunRaw(pipe) error = %v, want ErrRawModeUnsupported", err)
	}
	if called {
		t.Fatal("RunRaw called callback after raw-mode initialization failed")
	}
}
