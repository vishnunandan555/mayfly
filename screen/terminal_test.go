package screen

import (
	"bytes"
	"strings"
	"testing"
)

func newTestTerminal(viewport Size) (*Terminal, *bytes.Buffer) {
	var output bytes.Buffer
	return NewTerminalWithConfig(&output, viewport, StyleConfig{ColorMode: ColorModeBright}), &output
}

func TestControlSequences(t *testing.T) {
	terminal, output := newTestTerminal(Size{Rows: 4, Columns: 8})

	if err := terminal.ClearScreen(); err != nil {
		t.Fatal(err)
	}
	if err := terminal.ClearLine(); err != nil {
		t.Fatal(err)
	}
	if err := terminal.HideCursor(); err != nil {
		t.Fatal(err)
	}
	if err := terminal.ShowCursor(); err != nil {
		t.Fatal(err)
	}
	if err := terminal.SaveCursor(); err != nil {
		t.Fatal(err)
	}
	if err := terminal.RestoreCursor(); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Reset(); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}

	want := "\x1b[2J\x1b[H\x1b[2K\x1b[?25l\x1b[?25h\x1b[s\x1b[u\x1b[0m\x1b[?25h"
	if got := output.String(); got != want {
		t.Fatalf("control sequences = %q, want %q", got, want)
	}
}

func TestMoveCursorUsesZeroBasedCoordinatesAndClips(t *testing.T) {
	terminal, output := newTestTerminal(Size{Rows: 3, Columns: 4})
	for _, point := range [][2]int{{0, 0}, {1, 1}, {-1, -1}, {9, 9}} {
		if err := terminal.MoveCursor(point[0], point[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}

	want := "\x1b[1;1H\x1b[2;2H\x1b[1;1H\x1b[3;4H"
	if got := output.String(); got != want {
		t.Fatalf("cursor sequences = %q, want %q", got, want)
	}
}

func TestStyleSGRAndStyledText(t *testing.T) {
	style := Style{
		Foreground: ColorRed,
		Background: ColorBlue,
		Attributes: AttrBold | AttrUnderline,
	}
	if got, want := style.SGR(), "\x1b[1;4;31;44m"; got != want {
		t.Fatalf("SGR = %q, want %q", got, want)
	}
	if got := (Style{}).SGR(); got != "" {
		t.Fatalf("zero style SGR = %q, want empty", got)
	}

	terminal, output := newTestTerminal(Size{Rows: 1, Columns: 10})
	if err := terminal.WriteStyled(style, "red"); err != nil {
		t.Fatal(err)
	}
	if err := terminal.WriteStyled(Style{}, " plain"); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}

	want := "\x1b[1;4;31;44mred\x1b[0m plain"
	if got := output.String(); got != want {
		t.Fatalf("styled output = %q, want %q", got, want)
	}
}

func TestRenderClipsFrameAndOverwritesRows(t *testing.T) {
	terminal, output := newTestTerminal(Size{Rows: 2, Columns: 4})
	frame := NewFrame(Size{Rows: 4, Columns: 7})
	frame.DrawText(0, 0, Style{}, "abcdef")
	frame.DrawText(1, 0, Style{}, "xy")
	frame.DrawText(3, 0, Style{}, "not visible")

	if err := terminal.Render(frame); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}

	want := "\x1b[?25l\x1b[1;1H\x1b[2Kabcd\x1b[2;1H\x1b[2Kxy  \x1b[1;1H"
	if got := output.String(); got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
}

func TestRenderClearsRowsFromPreviousLargerFrame(t *testing.T) {
	terminal, output := newTestTerminal(Size{Rows: 3, Columns: 5})
	first := NewFrame(Size{Rows: 3, Columns: 5})
	first.DrawText(2, 0, Style{}, "old")
	second := NewFrame(Size{Rows: 1, Columns: 5})

	if err := terminal.Render(first); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Render(second); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output.String(), "\x1b[3;1H\x1b[2K") {
		t.Fatalf("second render did not clear the previous third row: %q", output.String())
	}
}

func TestRenderEmptyFrameIsNoOp(t *testing.T) {
	terminal, output := newTestTerminal(Size{Rows: 3, Columns: 5})
	if err := terminal.Render(NewFrame(Size{})); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "" {
		t.Fatalf("empty frame output = %q, want empty", got)
	}
}

func TestRenderUnicodeAndMultilineText(t *testing.T) {
	terminal, output := newTestTerminal(Size{Rows: 2, Columns: 5})
	frame := NewFrame(Size{Rows: 2, Columns: 5})
	frame.DrawText(0, 0, Style{}, "π\n界")

	if err := terminal.Render(frame); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}

	want := "\x1b[?25l\x1b[1;1H\x1b[2Kπ    \x1b[2;1H\x1b[2K界   \x1b[1;1H"
	if got := output.String(); got != want {
		t.Fatalf("Unicode multiline render = %q, want %q", got, want)
	}
}

func TestOutputIsBufferedUntilFlush(t *testing.T) {
	terminal, output := newTestTerminal(Size{Rows: 1, Columns: 1})
	if err := terminal.WriteStyled(Style{}, "buffered"); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "" {
		t.Fatalf("output before Flush = %q, want empty", got)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "buffered" {
		t.Fatalf("output after Flush = %q, want %q", got, "buffered")
	}
}

func TestNilWriterIsDiscarded(t *testing.T) {
	terminal := NewTerminal(nil, Size{Rows: 1, Columns: 1})
	if err := terminal.WriteStyled(Style{}, "ignored"); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}
}

func TestWriteStyledSanitizesTerminalControlCharacters(t *testing.T) {
	terminal, output := newTestTerminal(Size{Rows: 1, Columns: 40})
	if err := terminal.WriteStyled(Style{}, "safe\x1b[2J\r\n\ttext"); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "safe [2J   text"; got != want {
		t.Fatalf("sanitized text = %q, want %q", got, want)
	}
}

func TestRenderZeroViewportClearsPriorFrameWithoutOutOfBoundsCursor(t *testing.T) {
	terminal, output := newTestTerminal(Size{Rows: 3, Columns: 5})
	if err := terminal.Render(NewFrame(Size{Rows: 2, Columns: 5})); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	terminal.SetViewport(Size{})
	if err := terminal.Render(NewFrame(Size{Rows: 2, Columns: 5})); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "\x1b[3;") || strings.Contains(output.String(), "\x1b[2;") {
		t.Fatalf("zero viewport emitted row movement: %q", output.String())
	}
	if !strings.Contains(output.String(), "\x1b[2J\x1b[H") {
		t.Fatalf("zero viewport did not clear prior frame: %q", output.String())
	}
}
