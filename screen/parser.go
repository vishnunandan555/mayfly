package screen

import (
	"bytes"
	"unicode"
	"unicode/utf8"
)

const maxPendingInput = 64

// Parser incrementally decodes printable UTF-8 and common ANSI/VT keyboard
// sequences. Feed may be called with arbitrary chunks, including one byte at
// a time. Escape is held until more input arrives or Flush is called because
// it may begin an ANSI sequence.
type Parser struct {
	pending []byte
}

// NewParser creates an empty incremental parser.
func NewParser() *Parser {
	return &Parser{}
}

// Pending reports whether the parser is waiting for more bytes to complete a
// UTF-8 character or escape sequence.
func (p *Parser) Pending() bool {
	return len(p.pending) > 0
}

// Feed adds bytes and returns every event that can be decoded without waiting
// for more bytes.
func (p *Parser) Feed(data []byte) []Event {
	var events []Event
	for len(data) > 0 {
		chunkSize := len(data)
		if chunkSize > maxPendingInput {
			chunkSize = maxPendingInput
		}
		p.pending = append(p.pending, data[:chunkSize]...)
		events = append(events, p.parse(false)...)
		data = data[chunkSize:]
	}
	return events
}

// Flush resolves incomplete input. A lone Escape becomes EventEscape; an
// incomplete escape sequence becomes EventUnknown so malformed input is
// observable rather than silently lost. Incomplete UTF-8 bytes likewise
// become one unknown event.
func (p *Parser) Flush() []Event {
	return p.parse(true)
}

func (p *Parser) parse(flush bool) []Event {
	var events []Event
	for len(p.pending) > 0 {
		if p.pending[0] == 0x1b {
			if event, consumed, complete := parseEscape(p.pending); complete {
				events = append(events, event)
				p.pending = p.pending[consumed:]
				continue
			}
			if !flush {
				if len(p.pending) > maxPendingInput {
					events = append(events, Event{Type: EventUnknown, Bytes: cloneBytes(p.pending[:maxPendingInput])})
					p.pending = nil
				}
				break
			}
			if len(p.pending) == 1 {
				events = append(events, Event{Type: EventEscape})
			} else {
				events = append(events, Event{Type: EventUnknown, Bytes: cloneBytes(p.pending)})
			}
			p.pending = nil
			continue
		}

		event, consumed, complete := parseText(p.pending, flush)
		if !complete {
			break
		}
		events = append(events, event)
		p.pending = p.pending[consumed:]
	}
	return events
}

func parseText(data []byte, flush bool) (Event, int, bool) {
	first := data[0]
	switch first {
	case '\r', '\n':
		return Event{Type: EventEnter}, 1, true
	case '\b', 0x7f:
		return Event{Type: EventBackspace}, 1, true
	case '\t':
		return Event{Type: EventTab}, 1, true
	case 0x03:
		return Event{Type: EventCtrlC}, 1, true
	case 0x04:
		return Event{Type: EventCtrlD}, 1, true
	case 0x15:
		return Event{Type: EventCtrlU}, 1, true
	case 0x17:
		return Event{Type: EventCtrlW}, 1, true
	}

	runeValue, size := utf8.DecodeRune(data)
	if runeValue == utf8.RuneError && size == 1 && !utf8.FullRune(data) && !flush {
		return Event{}, 0, false
	}
	if runeValue == utf8.RuneError && size == 1 && data[0] >= utf8.RuneSelf {
		return Event{Type: EventUnknown, Bytes: []byte{data[0]}}, 1, true
	}
	if unicode.IsPrint(runeValue) {
		return Event{Type: EventRune, Rune: runeValue}, size, true
	}
	return Event{Type: EventUnknown, Bytes: cloneBytes(data[:size])}, size, true
}

func parseEscape(data []byte) (Event, int, bool) {
	if len(data) == 1 {
		return Event{}, 0, false
	}
	if data[1] != '[' && data[1] != 'O' {
		// An escape followed by an ordinary byte is an Escape event followed by
		// ordinary input. Consume only Escape so the next parse handles it.
		return Event{Type: EventEscape}, 1, true
	}
	if data[1] == 'O' {
		if len(data) < 3 {
			return Event{}, 0, false
		}
		event := parseVTFunction(data[2])
		if event.Type == EventUnknown {
			return Event{Type: EventUnknown, Bytes: cloneBytes(data[:3])}, 3, true
		}
		return event, 3, true
	}

	for index := 2; index < len(data); index++ {
		if data[index] < 0x40 || data[index] > 0x7e {
			continue
		}
		event := parseCSI(data[2:index], data[index])
		if index+1 > maxPendingInput {
			return Event{Type: EventUnknown, Bytes: cloneBytes(data[:index+1])}, index + 1, true
		}
		if event.Type == EventUnknown {
			return Event{Type: EventUnknown, Bytes: cloneBytes(data[:index+1])}, index + 1, true
		}
		return event, index + 1, true
	}
	return Event{}, 0, false
}

func parseVTFunction(final byte) Event {
	switch final {
	case 'A':
		return Event{Type: EventArrowUp}
	case 'B':
		return Event{Type: EventArrowDown}
	case 'C':
		return Event{Type: EventArrowRight}
	case 'D':
		return Event{Type: EventArrowLeft}
	case 'H':
		return Event{Type: EventHome}
	case 'F':
		return Event{Type: EventEnd}
	default:
		return Event{Type: EventUnknown}
	}
}

func parseCSI(params []byte, final byte) Event {
	if !validCSIParams(params) {
		return Event{Type: EventUnknown}
	}
	switch final {
	case 'A':
		return Event{Type: EventArrowUp}
	case 'B':
		return Event{Type: EventArrowDown}
	case 'C':
		return Event{Type: EventArrowRight}
	case 'D':
		return Event{Type: EventArrowLeft}
	case 'H':
		return Event{Type: EventHome}
	case 'F':
		return Event{Type: EventEnd}
	case '~':
		if bytes.Equal(params, []byte("1")) || bytes.Equal(params, []byte("7")) {
			return Event{Type: EventHome}
		}
		if bytes.Equal(params, []byte("5")) {
			return Event{Type: EventPageUp}
		}
		if bytes.Equal(params, []byte("6")) {
			return Event{Type: EventPageDown}
		}
		if bytes.Equal(params, []byte("3")) {
			return Event{Type: EventDelete}
		}
		if bytes.Equal(params, []byte("4")) || bytes.Equal(params, []byte("8")) {
			return Event{Type: EventEnd}
		}
	case 'Z':
		return Event{Type: EventShiftTab}
	}
	return Event{Type: EventUnknown}
}

func validCSIParams(params []byte) bool {
	for _, value := range params {
		if (value < '0' || value > '9') && value != ';' && value != '?' && value != '>' {
			return false
		}
	}
	return true
}

func cloneBytes(data []byte) []byte {
	if len(data) > maxPendingInput {
		data = data[:maxPendingInput]
	}
	return append([]byte(nil), data...)
}
