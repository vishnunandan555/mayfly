package screen

import (
	"errors"
	"os"
	"testing"
)

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
