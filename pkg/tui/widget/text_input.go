package widget

import (
	"mayfly/pkg/tui/terminal"
)

type TextInput struct {
	Value       string
	Cursor      int
	Masked      bool
	MaskRune    rune
	Placeholder string
	Focused     bool
	Label       string
}

func NewTextInput(label, placeholder string, masked bool) *TextInput {
	return &TextInput{
		Label:       label,
		Placeholder: placeholder,
		Masked:      masked,
		MaskRune:    '•',
	}
}

func (t *TextInput) SetFocused(focused bool) {
	t.Focused = focused
}

func (t *TextInput) SetValue(v string) {
	t.Value = v
	t.Cursor = len([]rune(v))
}

func (t *TextInput) Clear() {
	t.Value = ""
	t.Cursor = 0
}

func (t *TextInput) HandleKey(event terminal.KeyEvent) bool {
	if !t.Focused {
		return false
	}

	runes := []rune(t.Value)

	switch event.Type {
	case terminal.KeyLeft:
		if t.Cursor > 0 {
			t.Cursor--
		}
		return true
	case terminal.KeyRight:
		if t.Cursor < len(runes) {
			t.Cursor++
		}
		return true
	case terminal.KeyHome:
		t.Cursor = 0
		return true
	case terminal.KeyEnd:
		t.Cursor = len(runes)
		return true
	case terminal.KeyBackspace:
		if t.Cursor > 0 {
			runes = append(runes[:t.Cursor-1], runes[t.Cursor:]...)
			t.Value = string(runes)
			t.Cursor--
		}
		return true
	case terminal.KeyDelete:
		if t.Cursor < len(runes) {
			runes = append(runes[:t.Cursor], runes[t.Cursor+1:]...)
			t.Value = string(runes)
		}
		return true
	case terminal.KeyRune:
		runes = append(runes[:t.Cursor], append([]rune{event.Rune}, runes[t.Cursor:]...)...)
		t.Value = string(runes)
		t.Cursor++
		return true
	}

	return false
}

func (t *TextInput) Draw(f *terminal.Frame, bounds terminal.Rect) {
	if bounds.Max.Row <= bounds.Min.Row || bounds.Max.Column <= bounds.Min.Column {
		return
	}

	labelStyle := terminal.Style{Foreground: terminal.ColorBrightWhite, Attributes: terminal.AttrBold}
	if t.Focused {
		labelStyle.Foreground = terminal.ColorBrightCyan
	}

	boxStyle := terminal.Style{Foreground: terminal.ColorBrightBlack}
	if t.Focused {
		boxStyle.Foreground = terminal.ColorBrightCyan
	}

	f.DrawBox(bounds, boxStyle, t.Label)

	innerRow := bounds.Min.Row + (bounds.Max.Row-bounds.Min.Row)/2
	innerCol := bounds.Min.Column + 2
	maxCols := bounds.Max.Column - innerCol - 2
	if maxCols <= 0 {
		return
	}

	runes := []rune(t.Value)
	displayRunes := make([]rune, len(runes))
	for i, r := range runes {
		if t.Masked {
			displayRunes[i] = t.MaskRune
		} else {
			displayRunes[i] = r
		}
	}

	if len(displayRunes) == 0 && !t.Focused && t.Placeholder != "" {
		f.DrawText(innerRow, innerCol, terminal.Style{Foreground: terminal.ColorBrightBlack, Attributes: terminal.AttrDim}, t.Placeholder)
		return
	}

	// Calculate scroll offset to keep cursor visible
	offset := 0
	if t.Cursor >= maxCols {
		offset = t.Cursor - maxCols + 1
	}

	visibleRunes := displayRunes[offset:]
	if len(visibleRunes) > maxCols {
		visibleRunes = visibleRunes[:maxCols]
	}

	textStyle := terminal.Style{Foreground: terminal.ColorBrightWhite}
	f.DrawText(innerRow, innerCol, textStyle, string(visibleRunes))

	// Draw cursor if focused
	if t.Focused {
		cursorCol := innerCol + (t.Cursor - offset)
		if cursorCol < bounds.Max.Column-1 {
			curRune := ' '
			if t.Cursor < len(displayRunes) {
				curRune = displayRunes[t.Cursor]
			}
			f.SetCell(innerRow, cursorCol, terminal.Cell{
				Rune:  curRune,
				Style: terminal.Style{Foreground: terminal.ColorBlack, Background: terminal.ColorBrightWhite},
			})
		}
	}
}
