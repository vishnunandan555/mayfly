package screen

import "testing"

func TestFrameClipsCoordinatesAtBoundaries(t *testing.T) {
	frame := NewFrame(Size{Rows: 2, Columns: 3})
	if frame.SetCell(0, 0, Cell{Rune: 'a'}) != true {
		t.Fatal("SetCell rejected the origin")
	}
	if frame.SetCell(1, 2, Cell{Rune: 'z'}) != true {
		t.Fatal("SetCell rejected the last cell")
	}
	for _, point := range [][2]int{{-1, 0}, {0, -1}, {2, 0}, {0, 3}} {
		if frame.SetCell(point[0], point[1], Cell{Rune: 'x'}) {
			t.Fatalf("SetCell accepted out-of-bounds coordinate %v", point)
		}
	}

	if cell, ok := frame.Cell(0, 0); !ok || cell.Rune != 'a' {
		t.Fatalf("origin cell = %#v, %v", cell, ok)
	}
	if cell, ok := frame.Cell(1, 2); !ok || cell.Rune != 'z' {
		t.Fatalf("last cell = %#v, %v", cell, ok)
	}
}

func TestFrameDrawsUnicodeAndMultilineText(t *testing.T) {
	frame := NewFrame(Size{Rows: 3, Columns: 6})
	frame.DrawText(0, 1, Style{}, "é界\nGo")

	for column, want := range []rune{' ', 'é', '界', ' ', ' ', ' '} {
		cell, ok := frame.Cell(0, column)
		if !ok || cell.Rune != want {
			t.Fatalf("row 0 column %d = %#v, %v; want %q", column, cell, ok, want)
		}
	}
	for column, want := range []rune{' ', 'G', 'o', ' ', ' ', ' '} {
		cell, ok := frame.Cell(1, column)
		if !ok || cell.Rune != want {
			t.Fatalf("row 1 column %d = %#v, %v; want %q", column, cell, ok, want)
		}
	}
}

func TestFrameClipsMultilineText(t *testing.T) {
	frame := NewFrame(Size{Rows: 2, Columns: 3})
	frame.DrawText(-1, -1, Style{}, "abcdef\nXYZ")

	for column, want := range []rune{'Y', 'Z', ' '} {
		cell, _ := frame.Cell(0, column)
		if cell.Rune != want {
			t.Fatalf("clipped row 0 column %d = %q, want %q", column, cell.Rune, want)
		}
	}
}

func TestFrameSanitizesCellControlCharacters(t *testing.T) {
	frame := NewFrame(Size{Rows: 1, Columns: 4})
	frame.SetCell(0, 0, Cell{Rune: '\n'})
	frame.SetCell(0, 1, Cell{Rune: '\t'})
	frame.SetCell(0, 2, Cell{Rune: '\x1b'})
	frame.SetCell(0, 3, Cell{Rune: 0})
	for column := 0; column < 4; column++ {
		cell, _ := frame.Cell(0, column)
		if cell.Rune != ' ' {
			t.Fatalf("control cell at column %d = %q, want space", column, cell.Rune)
		}
	}
}

func TestFrameSetLineClearsPreviousContent(t *testing.T) {
	frame := NewFrame(Size{Rows: 1, Columns: 5})
	frame.DrawText(0, 0, Style{}, "hello")
	frame.SetLine(0, Style{}, "yo")

	for column, want := range []rune{'y', 'o', ' ', ' ', ' '} {
		cell, _ := frame.Cell(0, column)
		if cell.Rune != want {
			t.Fatalf("column %d = %q, want %q", column, cell.Rune, want)
		}
	}
}

func TestFrameSetLineDoesNotSpillOnNewline(t *testing.T) {
	frame := NewFrame(Size{Rows: 2, Columns: 3})
	frame.SetLine(0, Style{}, "a\nb")

	if cell, _ := frame.Cell(0, 1); cell.Rune != ' ' {
		t.Fatalf("newline in SetLine = %q, want space", cell.Rune)
	}
	if cell, _ := frame.Cell(0, 2); cell.Rune != 'b' {
		t.Fatalf("text after newline in SetLine = %q, want b", cell.Rune)
	}
	if cell, _ := frame.Cell(1, 0); cell.Rune != ' ' {
		t.Fatalf("SetLine spilled into next row with %q", cell.Rune)
	}
}
