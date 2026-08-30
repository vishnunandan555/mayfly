package terminal

import (
	"unicode/utf8"
)

type KeyType int

const (
	KeyRune KeyType = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyEscape
	KeyBackspace
	KeyDelete
	KeyTab
	KeyShiftTab
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
)

type KeyEvent struct {
	Type KeyType
	Rune rune
}

// Parser is a finite-state machine that parses byte chunks into KeyEvents.
type Parser struct {
	buf []byte
}

func NewParser() *Parser {
	return &Parser{buf: make([]byte, 0, 32)}
}

func (p *Parser) Feed(data []byte) []KeyEvent {
	p.buf = append(p.buf, data...)
	var events []KeyEvent

	for len(p.buf) > 0 {
		// Escape sequences
		if p.buf[0] == 0x1b {
			if len(p.buf) == 1 {
				// Standalone Escape
				events = append(events, KeyEvent{Type: KeyEscape})
				p.buf = p.buf[1:]
				continue
			}

			if p.buf[1] == '[' {
				if len(p.buf) < 3 {
					break // Incomplete sequence, wait for more bytes
				}

				switch p.buf[2] {
				case 'A':
					events = append(events, KeyEvent{Type: KeyUp})
					p.buf = p.buf[3:]
					continue
				case 'B':
					events = append(events, KeyEvent{Type: KeyDown})
					p.buf = p.buf[3:]
					continue
				case 'C':
					events = append(events, KeyEvent{Type: KeyRight})
					p.buf = p.buf[3:]
					continue
				case 'D':
					events = append(events, KeyEvent{Type: KeyLeft})
					p.buf = p.buf[3:]
					continue
				case 'H':
					events = append(events, KeyEvent{Type: KeyHome})
					p.buf = p.buf[3:]
					continue
				case 'F':
					events = append(events, KeyEvent{Type: KeyEnd})
					p.buf = p.buf[3:]
					continue
				case 'Z':
					events = append(events, KeyEvent{Type: KeyShiftTab})
					p.buf = p.buf[3:]
					continue
				case '1', '2', '3', '4', '5', '6':
					if len(p.buf) < 4 {
						break
					}
					if p.buf[3] == '~' {
						switch p.buf[2] {
						case '3':
							events = append(events, KeyEvent{Type: KeyDelete})
						case '5':
							events = append(events, KeyEvent{Type: KeyPageUp})
						case '6':
							events = append(events, KeyEvent{Type: KeyPageDown})
						}
						p.buf = p.buf[4:]
						continue
					}
				}
			}

			// Unknown escape, drop escape byte
			events = append(events, KeyEvent{Type: KeyEscape})
			p.buf = p.buf[1:]
			continue
		}

		// Control characters
		b := p.buf[0]
		switch b {
		case '\r', '\n':
			events = append(events, KeyEvent{Type: KeyEnter})
			p.buf = p.buf[1:]
			continue
		case '\t':
			events = append(events, KeyEvent{Type: KeyTab})
			p.buf = p.buf[1:]
			continue
		case 0x7f, 0x08:
			events = append(events, KeyEvent{Type: KeyBackspace})
			p.buf = p.buf[1:]
			continue
		case 0x03: // Ctrl+C
			events = append(events, KeyEvent{Type: KeyEscape})
			p.buf = p.buf[1:]
			continue
		}

		// UTF-8 Rune
		r, size := utf8.DecodeRune(p.buf)
		if r != utf8.RuneError || size > 1 {
			events = append(events, KeyEvent{Type: KeyRune, Rune: r})
			p.buf = p.buf[size:]
			continue
		}

		p.buf = p.buf[1:]
	}

	return events
}
