package screen

import "testing"

func TestRectPointAndPaddingGeometry(t *testing.T) {
	rect := NewRect(1, 2, 3, 4)
	if got, want := rect.Size(), (Size{Rows: 3, Columns: 4}); got != want {
		t.Fatalf("rect size = %#v, want %#v", got, want)
	}
	if !rect.Contains(Point{Row: 1, Column: 2}) || !rect.Contains(Point{Row: 3, Column: 5}) {
		t.Fatal("rect did not contain its boundary interior")
	}
	if rect.Contains(Point{Row: 4, Column: 2}) || rect.Contains(Point{Row: 1, Column: 6}) {
		t.Fatal("rect included its half-open maximum")
	}
	if got, want := (Padding{Top: 1, Right: 2, Bottom: 1, Left: 2}).Inset(NewRect(0, 0, 4, 8)), NewRect(1, 2, 2, 4); got != want {
		t.Fatalf("inset = %#v, want %#v", got, want)
	}

	intersection := rect.Intersect(NewRect(0, 4, 4, 4))
	if got, want := intersection, NewRect(1, 4, 3, 2); got != want {
		t.Fatalf("intersection = %#v, want %#v", got, want)
	}
}

func TestFrameDrawsASCIIAndClipsAtBoundaries(t *testing.T) {
	frame := NewFrame(Size{Rows: 2, Columns: 5})
	frame.DrawText(0, -1, Style{}, "abcdef")
	frame.DrawText(1, 3, Style{}, "xyz")

	if got := frameRow(frame, 0); got != "bcdef" {
		t.Fatalf("clipped row 0 = %q, want %q", got, "bcdef")
	}
	if got := frameRow(frame, 1); got != "   xy" {
		t.Fatalf("clipped row 1 = %q, want %q", got, "   xy")
	}
}

func TestFrameDrawsUTF8UsingDocumentedWidths(t *testing.T) {
	if got, want := RuneWidth('a'), 1; got != want {
		t.Fatalf("ASCII width = %d, want %d", got, want)
	}
	if got, want := RuneWidth('é'), 1; got != want {
		t.Fatalf("Latin UTF-8 width = %d, want %d", got, want)
	}
	if got, want := RuneWidth('界'), 2; got != want {
		t.Fatalf("East Asian width = %d, want %d", got, want)
	}
	if got, want := RuneWidth('😀'), 2; got != want {
		t.Fatalf("emoji width = %d, want %d", got, want)
	}
	if got, want := RuneWidth('\u0301'), 0; got != want {
		t.Fatalf("combining width = %d, want %d", got, want)
	}
	if got, want := RuneWidth('\u200d'), 0; got != want {
		t.Fatalf("ZWJ width = %d, want %d", got, want)
	}
	if got, want := TextWidth("a\n界"), 2; got != want {
		t.Fatalf("multiline text width = %d, want %d", got, want)
	}

	frame := NewFrame(Size{Rows: 1, Columns: 6})
	frame.DrawText(0, 0, Style{}, "e\u0301界x")
	if got := frameRow(frame, 0); got != "e界x  " {
		t.Fatalf("UTF-8 row = %q, want %q", got, "e界x  ")
	}
	if cell, _ := frame.Cell(0, 2); !cell.Continuation {
		t.Fatal("wide rune did not create a continuation cell")
	}
}

func TestFrameHandlesEmptyLongMultilineAndOverlappingDraws(t *testing.T) {
	frame := NewFrame(Size{Rows: 3, Columns: 4})
	frame.DrawText(0, 0, Style{}, "")
	frame.DrawText(0, 0, Style{}, "abcdefghijk")
	frame.DrawText(1, 1, Style{}, "one\ntwo")
	frame.DrawText(0, 1, Style{}, "XY")

	if got := frameRow(frame, 0); got != "aXYd" {
		t.Fatalf("overlapping row 0 = %q, want %q", got, "aXYd")
	}
	if got := frameRow(frame, 1); got != " one" {
		t.Fatalf("multiline row 1 = %q, want %q", got, " one")
	}
	if got := frameRow(frame, 2); got != " two" {
		t.Fatalf("multiline row 2 = %q, want %q", got, " two")
	}
}

func TestFrameDrawTextInSupportsAlignmentAndPadding(t *testing.T) {
	frame := NewFrame(Size{Rows: 5, Columns: 10})
	frame.DrawTextIn(NewRect(0, 0, 5, 10), "hi", TextOptions{
		Horizontal: AlignRight,
		Vertical:   AlignBottom,
		Padding:    Padding{Top: 1, Right: 1, Bottom: 1, Left: 1},
	})

	if got := frameRow(frame, 0); got != "          " {
		t.Fatalf("aligned row 0 = %q", got)
	}
	if got := frameRow(frame, 3); got != "       hi " {
		t.Fatalf("aligned row 3 = %q, want %q", got, "       hi ")
	}
	if got := frameRow(frame, 4); got != "          " {
		t.Fatalf("padded row 4 = %q", got)
	}
}

func TestFrameFillClearRegionAndDrawBox(t *testing.T) {
	frame := NewFrame(Size{Rows: 5, Columns: 7})
	frame.Fill(NewRect(1, 1, 2, 3), Cell{Rune: '#', Style: Style{Foreground: ColorRed}})
	if got := frameRow(frame, 1); got != " ###   " {
		t.Fatalf("filled row = %q, want %q", got, " ###   ")
	}
	frame.ClearRegion(NewRect(1, 2, 1, 1))
	if got := frameRow(frame, 1); got != " # #   " {
		t.Fatalf("cleared region row = %q, want %q", got, " # #   ")
	}

	box := NewFrame(Size{Rows: 3, Columns: 5})
	box.DrawBox(box.Bounds(), Style{})
	want := []string{"┌───┐", "│   │", "└───┘"}
	for row, expected := range want {
		if got := frameRow(box, row); got != expected {
			t.Fatalf("box row %d = %q, want %q", row, got, expected)
		}
	}
}

func TestFrameResizePreservesOverlapAndClips(t *testing.T) {
	frame := NewFrame(Size{Rows: 2, Columns: 3})
	frame.DrawText(0, 0, Style{}, "abc")
	frame.DrawText(1, 0, Style{}, "def")
	frame.Resize(Size{Rows: 3, Columns: 5})
	if got := frameRow(frame, 0); got != "abc  " {
		t.Fatalf("expanded row = %q, want %q", got, "abc  ")
	}
	if got := frameRow(frame, 1); got != "def  " {
		t.Fatalf("expanded second row = %q, want %q", got, "def  ")
	}
	frame.Resize(Size{Rows: 1, Columns: 2})
	if got := frameRow(frame, 0); got != "ab" {
		t.Fatalf("shrunk row = %q, want %q", got, "ab")
	}
	if got := frame.Size(); got != (Size{Rows: 1, Columns: 2}) {
		t.Fatalf("resized dimensions = %#v", got)
	}
}

func frameRow(frame *Frame, row int) string {
	var rowText []rune
	for column := 0; column < frame.Size().Columns; column++ {
		cell, _ := frame.Cell(row, column)
		if cell.Continuation {
			continue
		}
		rowText = append(rowText, cell.Rune)
	}
	return string(rowText)
}
