// Package screen provides MayFly's terminal UI output and input foundation.
// Input decoding is explicit, and raw terminal mode is entered only when
// NewRawInput or RunRaw is called. Raw mode currently targets Linux termios;
// unsupported operating systems return ErrRawModeUnsupported. The parser
// targets common ANSI/VT-compatible terminals rather than every terminal
// protocol.
package screen

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

const escape = "\x1b["

// Size describes a terminal or frame in rows and columns.
//
// Rows and columns are counts, not maximum coordinates. A zero Size is an
// empty viewport. Frame and Terminal methods use zero-based coordinates; ANSI
// sequences are one-based and are generated internally by this package.
type Size struct {
	Rows    int
	Columns int
}

func (s Size) normalized() Size {
	if s.Rows < 0 {
		s.Rows = 0
	}
	if s.Columns < 0 {
		s.Columns = 0
	}
	return s
}

// Color is an ANSI SGR foreground color. ColorDefault means that no color is
// emitted for that part of a style.
type Color int

const (
	ColorDefault Color = iota
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

// Attributes is a set of ANSI SGR text attributes.
type Attributes uint16

const (
	AttrNone          Attributes = 0
	AttrBold          Attributes = 1 << 0
	AttrDim           Attributes = 1 << 1
	AttrItalic        Attributes = 1 << 2
	AttrUnderline     Attributes = 1 << 3
	AttrBlink         Attributes = 1 << 4
	AttrReverse       Attributes = 1 << 5
	AttrHidden        Attributes = 1 << 6
	AttrStrikethrough Attributes = 1 << 7
)

// Style describes the appearance of text in a frame.
type Style struct {
	Foreground Color
	Background Color
	Attributes Attributes
}

// IsZero reports whether the style leaves terminal text appearance unchanged.
func (s Style) IsZero() bool {
	return s.Foreground == ColorDefault && s.Background == ColorDefault && s.Attributes == AttrNone
}

// SGR returns the ANSI Select Graphic Rendition sequence for the style. It
// returns an empty string for the zero style.
func (s Style) SGR() string {
	if s.IsZero() {
		return ""
	}

	params := make([]string, 0, 10)
	for bit, code := range []int{1, 2, 3, 4, 5, 7, 8, 9} {
		if s.Attributes&(1<<bit) != 0 {
			params = append(params, fmt.Sprintf("%d", code))
		}
	}
	if s.Foreground != ColorDefault {
		params = append(params, fmt.Sprintf("%d", int(s.Foreground)+29))
	}
	if s.Background != ColorDefault {
		params = append(params, fmt.Sprintf("%d", int(s.Background)+39))
	}
	return escape + strings.Join(params, ";") + "m"
}

// Terminal writes ANSI control sequences and rendered frames to an injected
// writer. Output is buffered until Flush is called.
type Terminal struct {
	out           *bufio.Writer
	viewport      Size
	previousFrame Size
}

// NewTerminal creates a buffered terminal writer with an explicit viewport.
// The viewport is intentionally supplied by the caller: this package neither
// assumes a conventional terminal size nor invokes an external command to
// discover one.
func NewTerminal(w io.Writer, viewport Size) *Terminal {
	if w == nil {
		w = io.Discard
	}
	return &Terminal{out: bufio.NewWriter(w), viewport: viewport.normalized()}
}

// Viewport returns the terminal size used for clipping rendered frames.
func (t *Terminal) Viewport() Size {
	return t.viewport
}

// SetViewport changes the clipping viewport for subsequent operations.
func (t *Terminal) SetViewport(viewport Size) {
	t.viewport = viewport.normalized()
}

// Flush writes all buffered output to the injected writer.
func (t *Terminal) Flush() error {
	return t.out.Flush()
}

func (t *Terminal) write(s string) error {
	_, err := t.out.WriteString(s)
	return err
}

func csi(command string) string {
	return escape + command
}

// ClearScreen clears the display and moves the cursor to the logical origin.
func (t *Terminal) ClearScreen() error {
	return t.write(csi("2J") + csi("H"))
}

// ClearLine clears the current physical line without moving the cursor.
func (t *Terminal) ClearLine() error {
	return t.write(csi("2K"))
}

// MoveCursor moves to a zero-based row and column. Negative coordinates are
// clipped to zero, and positive coordinates are clipped to the configured
// viewport. Thus this method cannot address outside the controlled viewport.
func (t *Terminal) MoveCursor(row, column int) error {
	row, column = t.clipPoint(row, column)
	return t.write(csi(fmt.Sprintf("%d;%dH", row+1, column+1)))
}

func (t *Terminal) clipPoint(row, column int) (int, int) {
	if row < 0 {
		row = 0
	}
	if column < 0 {
		column = 0
	}
	if t.viewport.Rows > 0 && row >= t.viewport.Rows {
		row = t.viewport.Rows - 1
	}
	if t.viewport.Columns > 0 && column >= t.viewport.Columns {
		column = t.viewport.Columns - 1
	}
	return row, column
}

// HideCursor hides the terminal cursor.
func (t *Terminal) HideCursor() error {
	return t.write(csi("?25l"))
}

// ShowCursor shows the terminal cursor.
func (t *Terminal) ShowCursor() error {
	return t.write(csi("?25h"))
}

// SaveCursor saves the cursor position using the ANSI save-cursor sequence.
func (t *Terminal) SaveCursor() error {
	return t.write(csi("s"))
}

// RestoreCursor restores the cursor position saved by SaveCursor.
func (t *Terminal) RestoreCursor() error {
	return t.write(csi("u"))
}

// Reset restores common terminal text attributes and cursor visibility.
func (t *Terminal) Reset() error {
	return t.write(csi("0m") + csi("?25h"))
}

// WriteStyled writes text wrapped in the style's SGR sequence and a reset.
// Text is written as supplied. Frame rendering is the safe bounded API for
// arbitrary UI text because it emits only printable cell contents.
func (t *Terminal) WriteStyled(style Style, text string) error {
	if text == "" {
		return nil
	}
	if style.IsZero() {
		return t.write(text)
	}
	return t.write(style.SGR() + text + csi("0m"))
}

// Render draws the frame's intersection with the terminal viewport. Each
// visible line is cleared before its cells are written, so shorter updates do
// not leave stale characters behind. The cursor is returned to (0, 0) and is
// not allowed to move beyond the clipped region.
func (t *Terminal) Render(frame *Frame) error {
	if frame == nil {
		return fmt.Errorf("screen: cannot render a nil frame")
	}

	visible := frame.size
	if visible.Rows > t.viewport.Rows {
		visible.Rows = t.viewport.Rows
	}
	if visible.Columns > t.viewport.Columns {
		visible.Columns = t.viewport.Columns
	}
	visible = visible.normalized()

	if visible.Rows == 0 || visible.Columns == 0 {
		for row := 0; row < t.previousFrame.Rows; row++ {
			if err := t.MoveCursor(row, 0); err != nil {
				return err
			}
			if err := t.ClearLine(); err != nil {
				return err
			}
		}
		if t.previousFrame.Rows > 0 {
			if err := t.MoveCursor(0, 0); err != nil {
				return err
			}
		}
		t.previousFrame = Size{}
		return nil
	}

	if err := t.HideCursor(); err != nil {
		return err
	}
	for row := 0; row < visible.Rows; row++ {
		if err := t.MoveCursor(row, 0); err != nil {
			return err
		}
		if err := t.ClearLine(); err != nil {
			return err
		}
		if err := t.writeRow(frame, row, visible.Columns); err != nil {
			return err
		}
	}
	for row := visible.Rows; row < t.previousFrame.Rows; row++ {
		if err := t.MoveCursor(row, 0); err != nil {
			return err
		}
		if err := t.ClearLine(); err != nil {
			return err
		}
	}
	if err := t.MoveCursor(0, 0); err != nil {
		return err
	}
	t.previousFrame = visible
	return nil
}

func (t *Terminal) writeRow(frame *Frame, row, columns int) error {
	for column := 0; column < columns; {
		style := frame.cells[row][column].Style
		end := column + 1
		for end < columns && frame.cells[row][end].Style == style {
			end++
		}

		var text strings.Builder
		for index := column; index < end; index++ {
			cell := frame.cells[row][index]
			if cell.Continuation {
				continue
			}
			if RuneWidth(cell.Rune) == 2 && index+1 >= columns {
				text.WriteRune(' ')
				continue
			}
			if cell.Rune == 0 {
				text.WriteRune(' ')
				continue
			}
			text.WriteRune(cell.Rune)
		}
		if err := t.WriteStyled(style, text.String()); err != nil {
			return err
		}
		column = end
	}
	return nil
}
