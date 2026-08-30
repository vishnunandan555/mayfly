package terminal

import (
	"bytes"
	"testing"
)

func TestCanvasAndBoxDrawing(t *testing.T) {
	frame := NewFrame(Size{Rows: 10, Columns: 30})
	boxRect := NewRect(0, 0, 5, 20)
	frame.DrawBox(boxRect, Style{Foreground: ColorBrightCyan}, "Test")

	if frame.cells[0].Rune != '┌' {
		t.Fatalf("expected '┌', got %c", frame.cells[0].Rune)
	}

	var buf bytes.Buffer
	term := NewTerminal(&buf, Size{Rows: 10, Columns: 30})
	if err := term.Render(frame); err != nil {
		t.Fatal(err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected rendered ANSI output in buffer")
	}
}

func TestParser(t *testing.T) {
	parser := NewParser()

	// Up arrow: \x1b[A
	events := parser.Feed([]byte("\x1b[A"))
	if len(events) != 1 || events[0].Type != KeyUp {
		t.Fatalf("expected KeyUp, got: %#v", events)
	}

	// Down arrow: \x1b[B
	events = parser.Feed([]byte("\x1b[B"))
	if len(events) != 1 || events[0].Type != KeyDown {
		t.Fatalf("expected KeyDown, got: %#v", events)
	}

	// Enter: \n
	events = parser.Feed([]byte("\n"))
	if len(events) != 1 || events[0].Type != KeyEnter {
		t.Fatalf("expected KeyEnter, got: %#v", events)
	}
}
