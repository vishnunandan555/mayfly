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

	mu     sync.Mutex
	closed bool
}

// NewRawInput switches file to noncanonical, no-echo mode and returns an input
// source. The operation is explicit and has no package initialization side
// effect. Call Close, or use RunRaw, to guarantee restoration.
func NewRawInput(file *os.File) (*RawInput, error) {
	restore, err := enterRawMode(file)
	if err != nil {
		return nil, err
	}
	return &RawInput{input: NewInput(file), restore: restore}, nil
}

// ReadEvent reads the next event. A normal polling timeout is returned to the
// caller without changing terminal state; any other source error triggers
// restoration before the error is returned.
func (r *RawInput) ReadEvent() (Event, error) {
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
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	return r.restore()
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
	watchedSignals := append([]os.Signal{os.Interrupt}, additionalRawSignals()...)
	signal.Notify(signals, watchedSignals...)
	defer func() {
		signal.Stop(signals)
		close(done)
		err = errors.Join(err, input.Close())
	}()
	go func() {
		select {
		case <-signals:
			_ = input.Close()
		case <-done:
		}
	}()

	err = fn(input)
	return err
}
