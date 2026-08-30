package terminal

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// ANSI Attribute bitflags
const (
	AttrNone      uint8 = 0
	AttrBold      uint8 = 1 << 0
	AttrDim       uint8 = 1 << 1
	AttrItalic    uint8 = 1 << 2
	AttrUnderline uint8 = 1 << 3
	AttrReverse   uint8 = 1 << 4
)

// ANSI Color constants
const (
	ColorDefault       uint8 = 0
	ColorBlack         uint8 = 1
	ColorRed           uint8 = 2
	ColorGreen         uint8 = 3
	ColorYellow        uint8 = 4
	ColorBlue          uint8 = 5
	ColorMagenta       uint8 = 6
	ColorCyan          uint8 = 7
	ColorWhite         uint8 = 8
	ColorBrightBlack   uint8 = 9
	ColorBrightRed     uint8 = 10
	ColorBrightGreen   uint8 = 11
	ColorBrightYellow  uint8 = 12
	ColorBrightBlue    uint8 = 13
	ColorBrightMagenta uint8 = 14
	ColorBrightCyan    uint8 = 15
	ColorBrightWhite   uint8 = 16
)

type Style struct {
	Foreground uint8
	Background uint8
	Attributes uint8
}

type Point struct {
	Row    int
	Column int
}

type Size struct {
	Rows    int
	Columns int
}

type Rect struct {
	Min Point
	Max Point
}

func NewRect(row, col, rows, cols int) Rect {
	return Rect{
		Min: Point{Row: row, Column: col},
		Max: Point{Row: row + rows, Column: col + cols},
	}
}

func (r Rect) Size() Size {
	return Size{
		Rows:    r.Max.Row - r.Min.Row,
		Columns: r.Max.Column - r.Min.Column,
	}
}

func (r Rect) Contains(p Point) bool {
	return p.Row >= r.Min.Row && p.Row < r.Max.Row && p.Column >= r.Min.Column && p.Column < r.Max.Column
}

type Cell struct {
	Rune  rune
	Style Style
}

// Frame is a 2D grid of character cells.
type Frame struct {
	size  Size
	cells []Cell
}

func NewFrame(size Size) *Frame {
	if size.Rows <= 0 {
		size.Rows = 24
	}
	if size.Columns <= 0 {
		size.Columns = 80
	}
	f := &Frame{
		size:  size,
		cells: make([]Cell, size.Rows*size.Columns),
	}
	f.Clear(Style{})
	return f
}

func (f *Frame) Size() Size {
	return f.size
}

func (f *Frame) Bounds() Rect {
	return Rect{Min: Point{0, 0}, Max: Point{f.size.Rows, f.size.Columns}}
}

func (f *Frame) Clear(s Style) {
	for i := range f.cells {
		f.cells[i] = Cell{Rune: ' ', Style: s}
	}
}

func (f *Frame) SetCell(row, col int, cell Cell) {
	if row < 0 || row >= f.size.Rows || col < 0 || col >= f.size.Columns {
		return
	}
	f.cells[row*f.size.Columns+col] = cell
}

func (f *Frame) DrawText(row, col int, style Style, text string) {
	currCol := col
	for _, r := range text {
		if r == '\n' {
			row++
			currCol = col
			continue
		}
		if row >= f.size.Rows {
			break
		}
		w := RuneWidth(r)
		if w <= 0 {
			continue
		}
		if currCol < f.size.Columns {
			f.SetCell(row, currCol, Cell{Rune: r, Style: style})
		}
		currCol += w
	}
}

func (f *Frame) DrawBox(r Rect, style Style, title string) {
	if r.Max.Row <= r.Min.Row || r.Max.Column <= r.Min.Column {
		return
	}

	// Corners
	f.SetCell(r.Min.Row, r.Min.Column, Cell{Rune: '┌', Style: style})
	f.SetCell(r.Min.Row, r.Max.Column-1, Cell{Rune: '┐', Style: style})
	f.SetCell(r.Max.Row-1, r.Min.Column, Cell{Rune: '└', Style: style})
	f.SetCell(r.Max.Row-1, r.Max.Column-1, Cell{Rune: '┘', Style: style})

	// Horizontal borders
	for c := r.Min.Column + 1; c < r.Max.Column-1; c++ {
		f.SetCell(r.Min.Row, c, Cell{Rune: '─', Style: style})
		f.SetCell(r.Max.Row-1, c, Cell{Rune: '─', Style: style})
	}

	// Vertical borders
	for row := r.Min.Row + 1; row < r.Max.Row-1; row++ {
		f.SetCell(row, r.Min.Column, Cell{Rune: '│', Style: style})
		f.SetCell(row, r.Max.Column-1, Cell{Rune: '│', Style: style})
	}

	// Title if any
	if title != "" && r.Max.Column-r.Min.Column > 4 {
		titleStr := " " + title + " "
		maxLen := r.Max.Column - r.Min.Column - 4
		if len(titleStr) > maxLen {
			titleStr = titleStr[:maxLen]
		}
		f.DrawText(r.Min.Row, r.Min.Column+2, Style{Foreground: ColorBrightWhite, Attributes: AttrBold}, titleStr)
	}
}

func RuneWidth(r rune) int {
	if r < 32 || (r >= 127 && r < 160) {
		return 0
	}
	// Basic East Asian wide detection
	if r >= 0x1100 &&
		(r <= 0x115f || r == 0x2329 || r == 0x232a ||
			(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
			(r >= 0xac00 && r <= 0xd7a3) ||
			(r >= 0xf900 && r <= 0xfaff) ||
			(r >= 0xfe10 && r <= 0xfe19) ||
			(r >= 0xfe30 && r <= 0xfe6f) ||
			(r >= 0xff00 && r <= 0xff60) ||
			(r >= 0xffe0 && r <= 0xffe6) ||
			(r >= 0x20000 && r <= 0x2fffd) ||
			(r >= 0x30000 && r <= 0x3fffd)) {
		return 2
	}
	return 1
}

// Terminal writes rendered ANSI diffs directly to the output stream.
type Terminal struct {
	out      io.Writer
	size     Size
	curFrame *Frame
	noColor  bool
}

func NewTerminal(out io.Writer, initialSize Size) *Terminal {
	if out == nil {
		out = os.Stdout
	}
	return &Terminal{
		out:      out,
		size:     initialSize,
		noColor:  os.Getenv("NO_COLOR") != "",
	}
}

func (t *Terminal) EnterAltScreen() {
	_, _ = io.WriteString(t.out, "\x1b[?1049h\x1b[H\x1b[?25l")
}

func (t *Terminal) ExitAltScreen() {
	_, _ = io.WriteString(t.out, "\x1b[?25h\x1b[?1049l\x1b[0m")
}

func (t *Terminal) ClearScreen() {
	_, _ = io.WriteString(t.out, "\x1b[2J\x1b[H")
}

func (t *Terminal) Render(f *Frame) error {
	var buf bytes.Buffer
	buf.WriteString("\x1b[H") // Move to 1,1

	var lastStyle Style
	styleActive := false

	for r := 0; r < f.size.Rows; r++ {
		for c := 0; c < f.size.Columns; c++ {
			cell := f.cells[r*f.size.Columns+c]
			if !t.noColor && (!styleActive || cell.Style != lastStyle) {
				buf.WriteString(formatSGR(cell.Style))
				lastStyle = cell.Style
				styleActive = true
			}
			if cell.Rune == 0 {
				buf.WriteByte(' ')
			} else {
				buf.WriteRune(cell.Rune)
			}
		}
		if r < f.size.Rows-1 {
			buf.WriteString("\r\n")
		}
	}

	if styleActive {
		buf.WriteString("\x1b[0m")
	}

	_, err := t.out.Write(buf.Bytes())
	return err
}

func formatSGR(s Style) string {
	var codes []string
	codes = append(codes, "0") // reset

	if s.Attributes&AttrBold != 0 {
		codes = append(codes, "1")
	}
	if s.Attributes&AttrDim != 0 {
		codes = append(codes, "2")
	}
	if s.Attributes&AttrItalic != 0 {
		codes = append(codes, "3")
	}
	if s.Attributes&AttrUnderline != 0 {
		codes = append(codes, "4")
	}
	if s.Attributes&AttrReverse != 0 {
		codes = append(codes, "7")
	}

	if s.Foreground >= ColorBlack && s.Foreground <= ColorWhite {
		codes = append(codes, fmt.Sprintf("%d", 29+s.Foreground))
	} else if s.Foreground >= ColorBrightBlack && s.Foreground <= ColorBrightWhite {
		codes = append(codes, fmt.Sprintf("%d", 81+s.Foreground))
	}

	if s.Background >= ColorBlack && s.Background <= ColorWhite {
		codes = append(codes, fmt.Sprintf("%d", 39+s.Background))
	} else if s.Background >= ColorBrightBlack && s.Background <= ColorBrightWhite {
		codes = append(codes, fmt.Sprintf("%d", 91+s.Background))
	}

	return "\x1b[" + strings.Join(codes, ";") + "m"
}
