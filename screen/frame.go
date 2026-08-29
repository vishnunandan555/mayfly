package screen

import (
	"unicode"
	"unicode/utf8"
)

// Point is a zero-based row and column coordinate.
type Point struct {
	Row    int
	Column int
}

// Rect is a half-open rectangle: Min is included and Max is excluded. This
// makes Rect{Max: Point{Rows: 3, Columns: 4}} a 3-by-4 area at the origin.
type Rect struct {
	Min Point
	Max Point
}

// NewRect creates a rectangle from an origin and row/column counts. Negative
// counts produce an empty rectangle.
func NewRect(row, column, rows, columns int) Rect {
	if rows < 0 {
		rows = 0
	}
	if columns < 0 {
		columns = 0
	}
	return Rect{Min: Point{Row: row, Column: column}, Max: Point{Row: row + rows, Column: column + columns}}
}

// Size returns the rectangle's row and column counts.
func (r Rect) Size() Size {
	rows, columns := r.Max.Row-r.Min.Row, r.Max.Column-r.Min.Column
	if rows < 0 {
		rows = 0
	}
	if columns < 0 {
		columns = 0
	}
	return Size{Rows: rows, Columns: columns}
}

// Empty reports whether the rectangle has no area.
func (r Rect) Empty() bool {
	return r.Size().Rows == 0 || r.Size().Columns == 0
}

// Contains reports whether p is inside the half-open rectangle.
func (r Rect) Contains(p Point) bool {
	return p.Row >= r.Min.Row && p.Row < r.Max.Row && p.Column >= r.Min.Column && p.Column < r.Max.Column
}

// Intersect returns the area shared by r and other.
func (r Rect) Intersect(other Rect) Rect {
	if r.Min.Row < other.Min.Row {
		r.Min.Row = other.Min.Row
	}
	if r.Min.Column < other.Min.Column {
		r.Min.Column = other.Min.Column
	}
	if r.Max.Row > other.Max.Row {
		r.Max.Row = other.Max.Row
	}
	if r.Max.Column > other.Max.Column {
		r.Max.Column = other.Max.Column
	}
	return r
}

// Padding describes the inset between a region's edge and its content.
type Padding struct {
	Top    int
	Right  int
	Bottom int
	Left   int
}

// Inset removes padding from a rectangle. Excessive padding produces an
// empty rectangle rather than inverted coordinates.
func (p Padding) Inset(r Rect) Rect {
	r.Min.Row += max(0, p.Top)
	r.Min.Column += max(0, p.Left)
	r.Max.Row -= max(0, p.Bottom)
	r.Max.Column -= max(0, p.Right)
	if r.Max.Row < r.Min.Row {
		r.Max.Row = r.Min.Row
	}
	if r.Max.Column < r.Min.Column {
		r.Max.Column = r.Min.Column
	}
	return r
}

// HorizontalAlignment controls placement of each text line within a region.
type HorizontalAlignment uint8

const (
	AlignLeft HorizontalAlignment = iota
	AlignCenter
	AlignRight
)

// VerticalAlignment controls placement of multiline text within a region.
type VerticalAlignment uint8

const (
	AlignTop VerticalAlignment = iota
	AlignMiddle
	AlignBottom
)

// TextOptions controls DrawTextIn. The zero value draws unstyled text at the
// top-left with no padding.
type TextOptions struct {
	Style      Style
	Horizontal HorizontalAlignment
	Vertical   VerticalAlignment
	Padding    Padding
}

// Cell is one logical terminal cell. A wide rune occupies its cell and the
// following cell, which is marked Continuation. Blank cells contain a space.
type Cell struct {
	Rune         rune
	Style        Style
	Continuation bool
}

// StyledCell is an alternate descriptive name for Cell.
type StyledCell = Cell

// RuneWidth returns this package's conservative terminal display-width model:
// controls, combining marks, variation selectors, format characters, and
// zero-width joiners are width zero; a documented subset of East Asian and
// emoji code points is width two; other printable runes are width one.
//
// This is not a complete Unicode grapheme or terminal-width implementation.
// Combining marks and zero-width characters are ignored by DrawText, emoji
// sequences joined by ZWJ are not shaped as one glyph, and ambiguous-width
// characters follow the one-cell default. Full-width handling is limited to
// the ranges in isWideRune.
func RuneWidth(r rune) int {
	if r == 0 || unicode.IsControl(r) || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Cf, r) {
		return 0
	}
	if isWideRune(r) {
		return 2
	}
	return 1
}

// TextWidth returns the maximum display width of any line in text according
// to RuneWidth. Newlines separate lines and do not contribute to width.
func TextWidth(text string) int {
	width, maximum := 0, 0
	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		text = text[size:]
		if r == '\n' {
			if width > maximum {
				maximum = width
			}
			width = 0
			continue
		}
		width += RuneWidth(r)
	}
	if width > maximum {
		maximum = width
	}
	return maximum
}

// Frame is a bounded logical canvas. Its origin is (0, 0), and all writes
// outside its rows and columns are clipped. A frame is independent of the
// terminal until Terminal.Render is called.
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
			frame.cells[row][column] = blankCell()
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

// Bounds returns the frame's half-open rectangle.
func (f *Frame) Bounds() Rect {
	if f == nil {
		return Rect{}
	}
	return NewRect(0, 0, f.size.Rows, f.size.Columns)
}

// Resize changes the frame dimensions and preserves the overlapping cells.
// Content that no longer fits is clipped. A wide rune clipped at the new
// right edge is replaced with a blank cell.
func (f *Frame) Resize(size Size) {
	if f == nil {
		return
	}
	size = size.normalized()
	resized := NewFrame(size)
	rows, columns := min(f.size.Rows, size.Rows), min(f.size.Columns, size.Columns)
	for row := 0; row < rows; row++ {
		copy(resized.cells[row][:columns], f.cells[row][:columns])
	}
	resized.normalizeWideCells()
	f.size, f.cells = resized.size, resized.cells
}

// Clear replaces every cell with a blank, unstyled cell.
func (f *Frame) Clear() {
	if f == nil {
		return
	}
	f.ClearRegion(f.Bounds())
}

// ClearRegion clears the intersection of rect and the frame bounds.
func (f *Frame) ClearRegion(rect Rect) {
	f.Fill(rect, blankCell())
}

// Fill fills the intersection of rect and the frame bounds with one-cell
// content. Wide or zero-width fill runes are conservatively replaced by a
// styled blank because a region fill must not create dangling wide cells.
func (f *Frame) Fill(rect Rect, cell Cell) {
	if f == nil {
		return
	}
	area := rect.Intersect(f.Bounds())
	if area.Empty() {
		return
	}
	if RuneWidth(cell.Rune) != 1 || cell.Continuation {
		cell.Rune = ' '
		cell.Continuation = false
	}
	for row := area.Min.Row; row < area.Max.Row; row++ {
		for column := area.Min.Column; column < area.Max.Column; column++ {
			f.clearCellAt(row, column)
			f.cells[row][column] = cell
		}
	}
}

// SetCell sets one coordinate and reports whether it was inside the frame.
// A wide rune occupies two cells when there is room; a zero-width/control
// rune becomes a blank cell.
func (f *Frame) SetCell(row, column int, cell Cell) bool {
	if f == nil || row < 0 || row >= f.size.Rows || column < 0 || column >= f.size.Columns {
		return false
	}
	if cell.Continuation {
		f.clearCellAt(row, column)
		cell = Cell{Rune: ' ', Style: cell.Style}
	}
	width := RuneWidth(cell.Rune)
	if width != 1 && width != 2 {
		cell.Rune = ' '
	}
	return f.putRune(row, column, cell.Rune, cell.Style)
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
// clipped. Combining and other zero-width characters are intentionally
// ignored according to RuneWidth.
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
		runeWidth := RuneWidth(runeValue)
		if runeWidth == 0 {
			continue
		}
		f.putRune(row, column, runeValue, style)
		column += runeWidth
	}
}

// SetLine clears a frame row and writes text from its first column. Newlines
// are treated as spaces and text is clipped to the frame width.
func (f *Frame) SetLine(row int, style Style, text string) {
	if f == nil || row < 0 || row >= f.size.Rows {
		return
	}
	f.ClearRegion(NewRect(row, 0, 1, f.size.Columns))
	column := 0
	for len(text) > 0 && column < f.size.Columns {
		runeValue, width := utf8.DecodeRuneInString(text)
		text = text[width:]
		if runeValue == '\n' || runeValue == '\r' || RuneWidth(runeValue) == 0 {
			runeValue = ' '
		}
		runeWidth := RuneWidth(runeValue)
		f.putRune(row, column, runeValue, style)
		column += runeWidth
	}
}

// DrawTextIn draws unwrapped multiline text in a bounded region with padding
// and horizontal/vertical alignment. Lines longer than the content region are
// clipped; text is never written outside the frame.
func (f *Frame) DrawTextIn(rect Rect, text string, options TextOptions) {
	if f == nil {
		return
	}
	area := rect.Intersect(f.Bounds())
	content := options.Padding.Inset(area)
	if content.Empty() {
		return
	}
	lines := splitLines(text)
	if len(lines) == 0 {
		return
	}
	lineStart := 0
	if len(lines) < content.Size().Rows {
		switch options.Vertical {
		case AlignMiddle:
			lineStart = (content.Size().Rows - len(lines)) / 2
		case AlignBottom:
			lineStart = content.Size().Rows - len(lines)
		}
	}
	for index, line := range lines {
		row := content.Min.Row + lineStart + index
		if row >= content.Max.Row {
			break
		}
		lineWidth := runesWidth(line)
		columnOffset := 0
		if lineWidth < content.Size().Columns {
			switch options.Horizontal {
			case AlignCenter:
				columnOffset = (content.Size().Columns - lineWidth) / 2
			case AlignRight:
				columnOffset = content.Size().Columns - lineWidth
			}
		}
		f.drawRunes(row, content.Min.Column+columnOffset, content.Max.Column, options.Style, line)
	}
}

// DrawBox draws a single-line box around rect. The rectangle is half-open and
// is clipped safely when it extends beyond the frame.
func (f *Frame) DrawBox(rect Rect, style Style) {
	if f == nil || rect.Empty() {
		return
	}
	width, height := rect.Size().Columns, rect.Size().Rows
	if width == 1 && height == 1 {
		f.putRune(rect.Min.Row, rect.Min.Column, '┌', style)
		return
	}
	if height == 1 {
		for column := rect.Min.Column; column < rect.Max.Column; column++ {
			f.putRune(rect.Min.Row, column, '─', style)
		}
		return
	}
	if width == 1 {
		for row := rect.Min.Row; row < rect.Max.Row; row++ {
			f.putRune(row, rect.Min.Column, '│', style)
		}
		return
	}
	for column := rect.Min.Column + 1; column < rect.Max.Column-1; column++ {
		f.putRune(rect.Min.Row, column, '─', style)
		f.putRune(rect.Max.Row-1, column, '─', style)
	}
	for row := rect.Min.Row + 1; row < rect.Max.Row-1; row++ {
		f.putRune(row, rect.Min.Column, '│', style)
		f.putRune(row, rect.Max.Column-1, '│', style)
	}
	f.putRune(rect.Min.Row, rect.Min.Column, '┌', style)
	f.putRune(rect.Min.Row, rect.Max.Column-1, '┐', style)
	f.putRune(rect.Max.Row-1, rect.Min.Column, '└', style)
	f.putRune(rect.Max.Row-1, rect.Max.Column-1, '┘', style)
}

func (f *Frame) drawRunes(row, column, endColumn int, style Style, runes []rune) {
	for _, runeValue := range runes {
		width := RuneWidth(runeValue)
		if width == 0 {
			continue
		}
		if column >= endColumn || (width == 2 && column+1 >= endColumn) {
			break
		}
		f.putRune(row, column, runeValue, style)
		column += width
	}
}

func (f *Frame) putRune(row, column int, runeValue rune, style Style) bool {
	if row < 0 || row >= f.size.Rows || column < 0 || column >= f.size.Columns {
		return false
	}
	width := RuneWidth(runeValue)
	if width == 0 {
		return false
	}
	if width == 2 && column+1 >= f.size.Columns {
		return false
	}
	f.clearCellAt(row, column)
	if width == 2 {
		f.clearCellAt(row, column+1)
	}
	f.cells[row][column] = Cell{Rune: runeValue, Style: style}
	if width == 2 {
		f.cells[row][column+1] = Cell{Style: style, Continuation: true}
	}
	return true
}

func (f *Frame) clearCellAt(row, column int) {
	if row < 0 || row >= f.size.Rows || column < 0 || column >= f.size.Columns {
		return
	}
	cell := f.cells[row][column]
	if cell.Continuation && column > 0 && RuneWidth(f.cells[row][column-1].Rune) == 2 {
		f.cells[row][column-1] = blankCell()
	}
	if RuneWidth(cell.Rune) == 2 && column+1 < f.size.Columns {
		f.cells[row][column+1] = blankCell()
	}
	f.cells[row][column] = blankCell()
}

func (f *Frame) normalizeWideCells() {
	for row := range f.cells {
		for column := range f.cells[row] {
			cell := f.cells[row][column]
			if RuneWidth(cell.Rune) == 2 {
				if column+1 >= f.size.Columns {
					f.cells[row][column] = blankCell()
					continue
				}
				f.cells[row][column+1] = Cell{Style: cell.Style, Continuation: true}
			} else if cell.Continuation && (column == 0 || RuneWidth(f.cells[row][column-1].Rune) != 2) {
				f.cells[row][column] = blankCell()
			}
		}
	}
}

func splitLines(text string) [][]rune {
	lines := [][]rune{{}}
	for len(text) > 0 {
		runeValue, width := utf8.DecodeRuneInString(text)
		text = text[width:]
		if runeValue == '\n' {
			lines = append(lines, []rune{})
			continue
		}
		if runeValue == '\r' {
			continue
		}
		if RuneWidth(runeValue) == 0 {
			continue
		}
		lines[len(lines)-1] = append(lines[len(lines)-1], runeValue)
	}
	return lines
}

func runesWidth(runes []rune) int {
	width := 0
	for _, r := range runes {
		width += RuneWidth(r)
	}
	return width
}

func isWideRune(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115f:
		return true
	case r == 0x2329 || r == 0x232a:
		return true
	case r >= 0x2e80 && r <= 0xa4cf && r != 0x303f:
		return !(r >= 0x2500 && r <= 0x27bf)
	case r >= 0xac00 && r <= 0xd7a3:
		return true
	case r >= 0xf900 && r <= 0xfaff:
		return true
	case r >= 0xfe10 && r <= 0xfe19:
		return true
	case r >= 0xfe30 && r <= 0xfe6f:
		return true
	case r >= 0xff01 && r <= 0xff60:
		return true
	case r >= 0xffe0 && r <= 0xffe6:
		return true
	case r >= 0x1f300 && r <= 0x1f64f:
		return true
	case r >= 0x1f680 && r <= 0x1faff:
		return true
	case r >= 0x20000 && r <= 0x3fffd:
		return true
	default:
		return false
	}
}

func blankCell() Cell {
	return Cell{Rune: ' '}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
