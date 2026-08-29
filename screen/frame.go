package screen

import (
	"unicode"
	"unicode/utf8"
)

// Cell is one logical terminal cell. This foundation treats each decoded
// Unicode code point as one cell; full-width and combining-character display
// width handling is intentionally deferred to a later phase.
type Cell struct {
	Rune  rune
	Style Style
}

// Frame is a bounded, logical canvas. Its origin is (0, 0), and all writes
// outside its rows and columns are clipped. A frame is independent of the
// terminal until Render is called.
type Frame struct {
	size  Size
	cells [][]Cell
}

// NewFrame creates a blank frame with the given row and column counts.
func NewFrame(size Size) *Frame {
	size = size.normalized()
	frame := &Frame{size: size, cells: make([][]Cell, size.Rows)}
	for row := range frame.cells {
		frame.cells[row] = make([]Cell, size.Columns)
		for column := range frame.cells[row] {
			frame.cells[row][column].Rune = ' '
		}
	}
	return frame
}

// Size returns the frame's row and column counts.
func (f *Frame) Size() Size {
	if f == nil {
		return Size{}
	}
	return f.size
}

// Clear replaces every cell with a blank, unstyled cell.
func (f *Frame) Clear() {
	if f == nil {
		return
	}
	for row := range f.cells {
		for column := range f.cells[row] {
			f.cells[row][column] = Cell{Rune: ' '}
		}
	}
}

// SetCell sets one cell and reports whether the coordinate was inside the
// frame. Out-of-bounds coordinates are ignored.
func (f *Frame) SetCell(row, column int, cell Cell) bool {
	if f == nil || row < 0 || row >= f.size.Rows || column < 0 || column >= f.size.Columns {
		return false
	}
	if unicode.IsControl(cell.Rune) {
		cell.Rune = ' '
	}
	f.cells[row][column] = cell
	return true
}

// Cell returns a cell and reports whether the coordinate was inside the
// frame.
func (f *Frame) Cell(row, column int) (Cell, bool) {
	if f == nil || row < 0 || row >= f.size.Rows || column < 0 || column >= f.size.Columns {
		return Cell{}, false
	}
	return f.cells[row][column], true
}

// DrawText writes text beginning at a zero-based coordinate. Newlines advance
// to the next row at the original starting column. Text outside the frame is
// clipped, including text that starts at a negative coordinate.
func (f *Frame) DrawText(row, column int, style Style, text string) {
	if f == nil {
		return
	}
	startColumn := column
	for len(text) > 0 {
		runeValue, width := utf8.DecodeRuneInString(text)
		text = text[width:]
		switch runeValue {
		case '\n':
			row++
			column = startColumn
			continue
		case '\r':
			column = startColumn
			continue
		case '\t':
			runeValue = ' '
		}
		f.SetCell(row, column, Cell{Rune: runeValue, Style: style})
		column++
	}
}

// SetLine clears a frame row and writes text from its first column. Text is
// clipped to the frame width and a newline is treated as a space.
func (f *Frame) SetLine(row int, style Style, text string) {
	if f == nil || row < 0 || row >= f.size.Rows {
		return
	}
	for column := range f.cells[row] {
		f.cells[row][column] = Cell{Rune: ' '}
	}
	column := 0
	for len(text) > 0 && column < f.size.Columns {
		runeValue, width := utf8.DecodeRuneInString(text)
		text = text[width:]
		f.SetCell(row, column, Cell{Rune: runeValue, Style: style})
		column++
	}
}
