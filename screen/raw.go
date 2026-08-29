package screen

import (
	"errors"
	"os"
	"os/signal"
	"sync"
)

// ErrRawModeUnsupported is returned when the supplied file is not an
// interactive terminal or the current operating system has no implementation
// in this package.
var ErrRawModeUnsupported = errors.New("screen: raw mode unsupported")

// RawInput is an InputReader backed by a terminal whose original termios state
// is restored by Close. It does not close the supplied *os.File.
type RawInput struct {
	input   *InputReader
	restore func() error

	mu       sync.Mutex
	closed   bool
	closeErr error
}

// rawFileReader bypasses os.File.Read's zero-byte-to-EOF normalization. With
// Linux VMIN=0/VTIME>0, the kernel uses (0, nil) to report a polling timeout;
// preserving that result is required to keep an idle TUI alive.
type rawFileReader struct{ file *os.File }

// NewRawInput switches file to noncanonical, no-echo mode and returns an input
// source. The operation is explicit and has no package initialization side
// effect. Call Close, or use RunRaw, to guarantee restoration.
func NewRawInput(file *os.File) (*RawInput, error) {
	restore, err := enterRawMode(file)
	if err != nil {
		return nil, err
	}
	return &RawInput{input: NewInput(&rawFileReader{file: file}), restore: restore}, nil
}

// ReadEvent reads the next event. A normal polling timeout is returned to the
// caller without changing terminal state; any other source error triggers
// restoration before the error is returned.
func (r *RawInput) ReadEvent() (Event, error) {
	if r == nil {
		return Event{}, ErrInputClosed
	}
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return Event{}, ErrInputClosed
	}

	event, err := r.input.ReadEvent()
	if err != nil {
		// A timeout is part of normal incremental escape-sequence parsing. It
		// must not end the input source or restore raw mode between events.
		if errors.Is(err, ErrInputTimeout) {
			return event, err
		}
		if restoreErr := r.Close(); restoreErr != nil {
			return event, errors.Join(err, restoreErr)
		}
	}
	return event, err
}

// Close restores the exact terminal state captured by NewRawInput. It is
// idempotent and does not close the caller's file descriptor.
func (r *RawInput) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return r.closeErr
	}
	r.closed = true
	if r.restore == nil {
		return nil
	}
	r.closeErr = r.restore()
	return r.closeErr
}

// RunRaw establishes raw mode, invokes fn, and restores terminal state on all
// return paths. SIGINT is watched while fn runs; receiving it restores the
// terminal and causes subsequent reads to return ErrInputClosed. Ctrl-C typed
// in raw mode is still delivered as EventCtrlC.
func RunRaw(file *os.File, fn func(Input) error) (err error) {
	if fn == nil {
		return errors.New("screen: nil raw input function")
	}
	input, err := NewRawInput(file)
	if err != nil {
		return err
	}

	signals := make(chan os.Signal, 1)
	done := make(chan struct{})
	finished := make(chan struct{})
	watchedSignals := append([]os.Signal{os.Interrupt}, additionalRawSignals()...)
	signal.Notify(signals, watchedSignals...)
	defer func() {
		signal.Stop(signals)
		close(done)
		<-finished
		err = errors.Join(err, input.Close())
	}()
	// This is the only goroutine created by RunRaw. It waits for a process
	// signal or the done channel; the deferred cleanup stops signal delivery,
	// closes done, waits for the goroutine to finish, and idempotently restores
	// the saved terminal state.
	go func() {
		defer close(finished)
		select {
		case <-signals:
			_ = input.Close()
		case <-done:
		}
	}()

	err = fn(input)
	return err
}
