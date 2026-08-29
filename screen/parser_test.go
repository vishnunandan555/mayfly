package screen

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func parseAll(data []byte) []Event {
	parser := NewParser()
	events := parser.Feed(data)
	return append(events, parser.Flush()...)
}

func TestParserDecodesTextAndCommonKeys(t *testing.T) {
	data := []byte("aé\r\n\b\x7f\t\x03\x04\x15" +
		"\x1b[A\x1b[B\x1b[C\x1b[D" +
		"\x1b[H\x1b[F\x1b[1~\x1b[4~\x1b[3~\x1b[7~\x1b[8~" +
		"\x1bOA\x1bOB\x1bOC\x1bOD\x1bOH\x1bOF")
	want := []Event{
		{Type: EventRune, Rune: 'a'},
		{Type: EventRune, Rune: 'é'},
		{Type: EventEnter}, {Type: EventEnter},
		{Type: EventBackspace}, {Type: EventBackspace}, {Type: EventTab},
		{Type: EventCtrlC}, {Type: EventCtrlD}, {Type: EventCtrlU},
		{Type: EventArrowUp}, {Type: EventArrowDown}, {Type: EventArrowRight}, {Type: EventArrowLeft},
		{Type: EventHome}, {Type: EventEnd}, {Type: EventHome}, {Type: EventEnd}, {Type: EventDelete},
		{Type: EventHome}, {Type: EventEnd},
		{Type: EventArrowUp}, {Type: EventArrowDown}, {Type: EventArrowRight}, {Type: EventArrowLeft},
		{Type: EventHome}, {Type: EventEnd},
	}
	if got := parseAll(data); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestParserHandlesSplitEscapeAndUTF8Sequences(t *testing.T) {
	parser := NewParser()
	var got []Event
	for _, chunk := range [][]byte{
		{0x1b}, {'['}, {'D'},
		{0xc3}, {0xa9},
	} {
		got = append(got, parser.Feed(chunk)...)
	}
	got = append(got, parser.Flush()...)

	want := []Event{{Type: EventArrowLeft}, {Type: EventRune, Rune: 'é'}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("split sequence events = %#v, want %#v", got, want)
	}
}

func TestParserDelaysStandaloneEscape(t *testing.T) {
	parser := NewParser()
	if got := parser.Feed([]byte{0x1b}); len(got) != 0 {
		t.Fatalf("event before ambiguity timeout = %#v, want none", got)
	}
	if !parser.Pending() {
		t.Fatal("parser should have a pending Escape")
	}
	if got, want := parser.Flush(), []Event{{Type: EventEscape}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("flushed Escape = %#v, want %#v", got, want)
	}
	if parser.Pending() {
		t.Fatal("parser retained flushed Escape")
	}
}

func TestParserRecoversFromMalformedAndUnexpectedInput(t *testing.T) {
	parser := NewParser()
	got := parser.Feed([]byte{0x1b, '[', 'x', 0x02, 0xff})
	got = append(got, parser.Flush()...)
	want := []Event{
		{Type: EventUnknown, Bytes: []byte{0x1b, '[', 'x'}},
		{Type: EventUnknown, Bytes: []byte{0x02}},
		{Type: EventUnknown, Bytes: []byte{0xff}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("malformed events = %#v, want %#v", got, want)
	}

	got = parseAll([]byte{0x1b, '[', '9'})
	want = []Event{{Type: EventUnknown, Bytes: []byte{0x1b, '[', '9'}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("incomplete sequence events = %#v, want %#v", got, want)
	}
}

func TestParserHandlesRepeatedSequences(t *testing.T) {
	data := bytes.Repeat([]byte("\x1b[A\x1b[3~"), 3)
	want := []Event{
		{Type: EventArrowUp}, {Type: EventDelete},
		{Type: EventArrowUp}, {Type: EventDelete},
		{Type: EventArrowUp}, {Type: EventDelete},
	}
	if got := parseAll(data); !reflect.DeepEqual(got, want) {
		t.Fatalf("repeated events = %#v, want %#v", got, want)
	}
}

func TestInputReaderReadsEventsUntilEOF(t *testing.T) {
	input := NewInput(bytes.NewBufferString("\x1b[Dq\x1b"))
	want := []Event{{Type: EventArrowLeft}, {Type: EventRune, Rune: 'q'}, {Type: EventEscape}}
	for index, expected := range want {
		got, err := input.ReadEvent()
		if err != nil {
			t.Fatalf("ReadEvent(%d) error = %v", index, err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("ReadEvent(%d) = %#v, want %#v", index, got, expected)
		}
	}
	if _, err := input.ReadEvent(); err != io.EOF {
		t.Fatalf("ReadEvent after EOF = %v, want EOF", err)
	}
}
