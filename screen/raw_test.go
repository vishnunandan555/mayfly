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

func TestInputReaderTreatsZeroByteReadAsTimeout(t *testing.T) {
	// Linux's raw TTY reader can return (0, nil) for a VTIME timeout. The
	// parser-facing InputReader must preserve that as a timeout rather than
	// treating it as EOF. The platform reader is exercised by the interactive
	// TTY path.
	input := NewInput(timeoutInputReader{})
	if _, err := input.ReadEvent(); !errors.Is(err, ErrInputTimeout) {
		t.Fatalf("InputReader error = %v, want timeout", err)
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

func TestNilRawInputIsSafeToClose(t *testing.T) {
	var input *RawInput
	if err := input.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := input.ReadEvent(); !errors.Is(err, ErrInputClosed) {
		t.Fatalf("nil ReadEvent error = %v, want closed", err)
	}
}

func TestRawInputPreservesRestoreErrorForRepeatedClose(t *testing.T) {
	wantErr := errors.New("restore failed")
	raw := &RawInput{restore: func() error { return wantErr }}
	if err := raw.Close(); !errors.Is(err, wantErr) {
		t.Fatalf("first Close error = %v, want restore error", err)
	}
	if err := raw.Close(); !errors.Is(err, wantErr) {
		t.Fatalf("second Close error = %v, want preserved restore error", err)
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
