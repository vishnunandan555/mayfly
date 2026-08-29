package screen

import (
	"errors"
	"io"
	"strings"
)

// Input is the high-level keyboard input abstraction used by the TUI.
// Implementations return one decoded event at a time.
type Input interface {
	ReadEvent() (Event, error)
}

// EventType identifies a keyboard event.
type EventType uint8

const (
	EventUnknown EventType = iota
	EventRune
	EventEnter
	EventEscape
	EventBackspace
	EventTab
	EventArrowUp
	EventArrowDown
	EventArrowLeft
	EventArrowRight
	EventHome
	EventEnd
	EventDelete
	EventCtrlC
	EventCtrlD
	EventCtrlU
	EventCtrlW
	EventPageUp
	EventPageDown
	EventShiftTab
)

// Event is one keyboard event. Rune is populated for EventRune. Bytes is
// populated for EventUnknown so callers can inspect malformed or unsupported
// input without the parser silently discarding it.
type Event struct {
	Type  EventType
	Rune  rune
	Bytes []byte
}

var (
	// ErrInputClosed indicates that an input source was closed by its owner.
	ErrInputClosed = errors.New("screen: input closed")

	// errInputTimeout is internal. A raw terminal read uses VTIME to report
	// that no byte arrived during the escape-sequence ambiguity window.
	errInputTimeout = errors.New("screen: input read timeout")

	// ErrInputTimeout indicates that a raw terminal had no byte available
	// during its configured polling interval.
	ErrInputTimeout = errInputTimeout
)

// InputReader combines a byte reader and the incremental ANSI parser. It is
// useful with any io.Reader, including bytes.Buffer or a pipe, and does not
// change terminal state.
type InputReader struct {
	bytes      *ByteReader
	parser     Parser
	queued     []Event
	sourceDone bool
	sourceErr  error
}

// NewInput creates an InputReader over r. It performs no terminal setup.
func NewInput(r io.Reader) *InputReader {
	if r == nil {
		r = strings.NewReader("")
	}
	return &InputReader{bytes: NewByteReader(r)}
}

// ReadEvent reads and decodes the next event. A standalone Escape is emitted
// when the source ends or returns a raw-mode timeout while no continuation
// bytes arrive.
func (r *InputReader) ReadEvent() (Event, error) {
	for {
		if len(r.queued) > 0 {
			event := r.queued[0]
			r.queued = r.queued[1:]
			return event, nil
		}
		if r.sourceDone {
			return Event{}, r.sourceErr
		}

		value, err := r.bytes.ReadByte()
		switch err {
		case nil:
			r.queued = append(r.queued, r.parser.Feed([]byte{value})...)
		case errInputTimeout:
			r.queued = append(r.queued, r.parser.Flush()...)
			if len(r.queued) == 0 {
				return Event{}, ErrInputTimeout
			}
		case io.EOF:
			r.sourceDone = true
			r.sourceErr = io.EOF
			r.queued = append(r.queued, r.parser.Flush()...)
		default:
			r.sourceDone = true
			r.sourceErr = err
			r.queued = append(r.queued, r.parser.Flush()...)
		}
	}
}

// Close makes InputReader return ErrInputClosed. It is a no-op for terminal
// state and exists so non-raw and raw inputs can be managed uniformly.
func (r *InputReader) Close() error {
	if r.sourceDone {
		return nil
	}
	r.sourceDone = true
	r.sourceErr = ErrInputClosed
	r.queued = nil
	return nil
}

// ByteReader reads exactly one byte at a time from an injected io.Reader. A
// zero-byte, nil-error read is treated as a timeout, which is how the Linux
// raw-mode VTIME setting is surfaced by os.File.
type ByteReader struct {
	source io.Reader
	buffer [1]byte
}

// NewByteReader creates a one-byte reader.
func NewByteReader(source io.Reader) *ByteReader {
	if source == nil {
		source = strings.NewReader("")
	}
	return &ByteReader{source: source}
}

// ReadByte reads one byte or returns the source error. If the underlying
// reader returns no byte and no error, ReadByte returns an internal timeout.
func (r *ByteReader) ReadByte() (byte, error) {
	n, err := r.source.Read(r.buffer[:])
	if n > 0 {
		return r.buffer[0], nil
	}
	if err != nil {
		return 0, err
	}
	return 0, errInputTimeout
}
