package screen

import "unicode"

// TextInput is a single-line editable text widget. Value returns plaintext by
// design for application use; Render never uses it directly when Password is
// enabled and the widget never logs or formats its value.
type TextInput struct {
	WidgetState
	value            []rune
	cursor           int
	scroll           int
	Placeholder      string
	Password         bool
	Style            Style
	PlaceholderStyle Style
	CursorStyle      Style
}

// NewTextInput creates an empty, focusable text input.
func NewTextInput() *TextInput {
	return &TextInput{CursorStyle: Style{Attributes: AttrReverse}}
}

// Focusable reports that text inputs participate in focus navigation.
func (*TextInput) Focusable() bool { return true }

// SetBounds assigns the input bounds and keeps its horizontal scroll valid.
func (t *TextInput) SetBounds(bounds Rect) {
	if t == nil {
		return
	}
	t.WidgetState.SetBounds(bounds)
	t.ensureCursorVisible()
}

// Value returns the current editable value. Callers should treat a password
// value as sensitive; the widget itself never includes it in output.
func (t *TextInput) Value() string {
	if t == nil {
		return ""
	}
	return string(t.value)
}

// SetValue replaces the value and places the cursor at the end.
func (t *TextInput) SetValue(value string) {
	if t == nil {
		return
	}
	t.value = sanitizeInputValue(value)
	t.cursor = len(t.value)
	t.scroll = 0
}

func sanitizeInputValue(value string) []rune {
	runes := []rune(value)
	result := make([]rune, 0, len(runes))
	for _, r := range runes {
		if RuneWidth(r) == 0 || unicode.IsControl(r) {
			continue
		}
		result = append(result, r)
	}
	return result
}

// Cursor returns the cursor's rune index between 0 and len(Value()).
func (t *TextInput) Cursor() int {
	if t == nil {
		return 0
	}
	return t.cursor
}

// SetCursor places the cursor at a clamped rune index.
func (t *TextInput) SetCursor(index int) {
	if t == nil {
		return
	}
	if index < 0 {
		index = 0
	}
	if index > len(t.value) {
		index = len(t.value)
	}
	t.cursor = index
	t.ensureCursorVisible()
}

// Handle updates the input for editing and cursor events. Enter, Escape, and
// navigation events not owned by the input are left for a parent controller.
func (t *TextInput) Handle(event Event) bool {
	if t == nil || !t.Focused() {
		return false
	}
	switch event.Type {
	case EventRune:
		if RuneWidth(event.Rune) == 0 || unicode.IsControl(event.Rune) {
			return false
		}
		t.value = append(t.value, 0)
		copy(t.value[t.cursor+1:], t.value[t.cursor:])
		t.value[t.cursor] = event.Rune
		t.cursor++
		t.ensureCursorVisible()
		return true
	case EventBackspace:
		if t.cursor == 0 {
			return true
		}
		t.value = append(t.value[:t.cursor-1], t.value[t.cursor:]...)
		t.cursor--
		t.ensureCursorVisible()
		return true
	case EventDelete:
		if t.cursor < len(t.value) {
			t.value = append(t.value[:t.cursor], t.value[t.cursor+1:]...)
		}
		return true
	case EventArrowLeft:
		if t.cursor > 0 {
			t.cursor--
		}
		t.ensureCursorVisible()
		return true
	case EventArrowRight:
		if t.cursor < len(t.value) {
			t.cursor++
		}
		t.ensureCursorVisible()
		return true
	case EventHome:
		t.cursor = 0
		t.ensureCursorVisible()
		return true
	case EventEnd:
		t.cursor = len(t.value)
		t.ensureCursorVisible()
		return true
	case EventCtrlU:
		t.value = t.value[t.cursor:]
		t.cursor = 0
		t.ensureCursorVisible()
		return true
	case EventCtrlW:
		start := t.cursor
		for start > 0 && unicode.IsSpace(t.value[start-1]) {
			start--
		}
		for start > 0 && !unicode.IsSpace(t.value[start-1]) {
			start--
		}
		t.value = append(t.value[:start], t.value[t.cursor:]...)
		t.cursor = start
		t.ensureCursorVisible()
		return true
	default:
		return false
	}
}

// Render draws the visible value or placeholder and a reverse cursor cell
// while focused. Password content is converted to bullets before it reaches
// the frame.
func (t *TextInput) Render(frame *Frame) {
	if t == nil || frame == nil {
		return
	}
	bounds, ok := widgetFrame(frame, t.Bounds())
	if !ok {
		return
	}
	width := bounds.Size().Columns
	if width == 0 || bounds.Size().Rows == 0 {
		return
	}
	frame.ClearRegion(bounds)
	display, cursorColumn, style := t.display(width)
	frame.DrawText(bounds.Min.Row, bounds.Min.Column, style, display)
	if !t.Focused() || cursorColumn < 0 || cursorColumn >= width {
		return
	}
	cell, exists := frame.Cell(bounds.Min.Row, bounds.Min.Column+cursorColumn)
	if exists && cell.Continuation {
		return
	}
	cell.Style = mergeCursorStyle(cell.Style, t.CursorStyle)
	if cell.Rune == 0 {
		cell.Rune = ' '
	}
	frame.SetCell(bounds.Min.Row, bounds.Min.Column+cursorColumn, cell)
}

func (t *TextInput) display(width int) (string, int, Style) {
	if len(t.value) == 0 && t.Placeholder != "" {
		return visibleText(t.Placeholder, 0, width), 0, t.PlaceholderStyle
	}
	text := t.Value()
	if t.Password {
		text = MaskSecret(text)
	}
	textWidth := TextWidth(text)
	cursorColumn := TextWidth(string(t.value[:t.cursor]))
	t.scroll = clampScroll(t.scroll, cursorColumn, width, textWidth)
	t.scroll = normalizeScroll(text, t.scroll)
	visible := visibleText(text, t.scroll, width)
	return visible, cursorColumn - t.scroll, t.Style
}

func (t *TextInput) ensureCursorVisible() {
	width := t.Bounds().Size().Columns
	if width <= 0 {
		return
	}
	text := t.Value()
	if t.Password {
		text = MaskSecret(text)
	}
	t.scroll = clampScroll(t.scroll, TextWidth(string(t.value[:t.cursor])), width, TextWidth(text))
	t.scroll = normalizeScroll(text, t.scroll)
}

func clampScroll(scroll, cursor, width, textWidth int) int {
	if width <= 0 {
		return 0
	}
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+width {
		scroll = cursor - width + 1
	}
	// Keep one cell available for the insertion cursor when it is at the end
	// of the value, so the cursor remains visible after long input.
	maxScroll := textWidth - width + 1
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	return scroll
}

func visibleText(text string, offset, width int) string {
	if width <= 0 {
		return ""
	}
	result := make([]rune, 0, width)
	position := 0
	for _, r := range []rune(text) {
		runeWidth := RuneWidth(r)
		if runeWidth == 0 {
			continue
		}
		if position+runeWidth <= offset {
			position += runeWidth
			continue
		}
		if position < offset {
			position += runeWidth
			continue
		}
		if position < offset || position+runeWidth > offset+width {
			break
		}
		result = append(result, r)
		position += runeWidth
	}
	return string(result)
}

func normalizeScroll(text string, offset int) int {
	if offset <= 0 {
		return 0
	}
	position := 0
	for _, r := range []rune(text) {
		width := RuneWidth(r)
		if width == 0 {
			continue
		}
		if position < offset && offset < position+width {
			return position + width
		}
		position += width
	}
	return offset
}

func mergeCursorStyle(base, cursor Style) Style {
	if cursor.Foreground != ColorDefault {
		base.Foreground = cursor.Foreground
	}
	if cursor.Background != ColorDefault {
		base.Background = cursor.Background
	}
	base.Attributes |= cursor.Attributes
	return base
}
