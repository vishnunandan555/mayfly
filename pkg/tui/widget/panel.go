package widget

import (
	"mayfly/pkg/tui/terminal"
)

type Label struct {
	Text  string
	Style terminal.Style
}

func NewLabel(text string, style terminal.Style) *Label {
	return &Label{Text: text, Style: style}
}

func (l *Label) Draw(f *terminal.Frame, bounds terminal.Rect) {
	f.DrawText(bounds.Min.Row, bounds.Min.Column, l.Style, l.Text)
}

func (l *Label) HandleKey(event terminal.KeyEvent) bool {
	return false
}

type StatusBar struct {
	LeftText  string
	RightText string
}

func NewStatusBar(left, right string) *StatusBar {
	return &StatusBar{LeftText: left, RightText: right}
}

func (s *StatusBar) Draw(f *terminal.Frame, bounds terminal.Rect) {
	bgStyle := terminal.Style{Foreground: terminal.ColorBlack, Background: terminal.ColorBrightWhite}
	for c := bounds.Min.Column; c < bounds.Max.Column; c++ {
		f.SetCell(bounds.Min.Row, c, terminal.Cell{Rune: ' ', Style: bgStyle})
	}

	f.DrawText(bounds.Min.Row, bounds.Min.Column+1, bgStyle, s.LeftText)
	if s.RightText != "" {
		col := bounds.Max.Column - len(s.RightText) - 1
		if col > bounds.Min.Column+len(s.LeftText)+2 {
			f.DrawText(bounds.Min.Row, col, bgStyle, s.RightText)
		}
	}
}

func (s *StatusBar) HandleKey(event terminal.KeyEvent) bool {
	return false
}

type ConfirmDialog struct {
	Title   string
	Message string
	Confirm bool
	Active  bool
}

func NewConfirmDialog(title, message string) *ConfirmDialog {
	return &ConfirmDialog{
		Title:   title,
		Message: message,
		Confirm: false,
		Active:  false,
	}
}

func (c *ConfirmDialog) HandleKey(event terminal.KeyEvent) bool {
	if !c.Active {
		return false
	}
	switch event.Type {
	case terminal.KeyLeft, terminal.KeyRight, terminal.KeyTab:
		c.Confirm = !c.Confirm
		return true
	}
	return false
}

func (c *ConfirmDialog) Draw(f *terminal.Frame, bounds terminal.Rect) {
	if !c.Active {
		return
	}

	reqWidth := len(c.Message) + 8
	if len(c.Title)+8 > reqWidth {
		reqWidth = len(c.Title) + 8
	}
	if reqWidth < 46 {
		reqWidth = 46
	}
	maxWidth := bounds.Max.Column - bounds.Min.Column - 4
	if maxWidth < 20 {
		maxWidth = 20
	}
	if reqWidth > maxWidth {
		reqWidth = maxWidth
	}

	dialogWidth := reqWidth
	dialogHeight := 7
	startRow := bounds.Min.Row + (bounds.Max.Row-bounds.Min.Row-dialogHeight)/2
	startCol := bounds.Min.Column + (bounds.Max.Column-bounds.Min.Column-dialogWidth)/2

	modalRect := terminal.NewRect(startRow, startCol, dialogHeight, dialogWidth)

	// Cleanly fill the inside of the modal dialog so underlying cards don't bleed through
	f.FillRect(modalRect, ' ', terminal.Style{})

	boxStyle := terminal.Style{Foreground: terminal.ColorBrightRed, Attributes: terminal.AttrBold}
	f.DrawBox(modalRect, boxStyle, c.Title)

	msg := c.Message
	if len(msg) > dialogWidth-6 {
		msg = msg[:dialogWidth-6]
	}
	msgCol := modalRect.Min.Column + (dialogWidth-len(msg))/2
	f.DrawText(modalRect.Min.Row+2, msgCol, terminal.Style{Foreground: terminal.ColorBrightWhite}, msg)

	btnRow := modalRect.Min.Row + 4

	yesStyle := terminal.Style{Foreground: terminal.ColorBrightWhite}
	noStyle := terminal.Style{Foreground: terminal.ColorBrightWhite}

	if c.Confirm {
		yesStyle = terminal.Style{Foreground: terminal.ColorBlack, Background: terminal.ColorBrightRed, Attributes: terminal.AttrBold}
	} else {
		noStyle = terminal.Style{Foreground: terminal.ColorBlack, Background: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}
	}

	yesBtn := " [ Yes ] "
	cancelBtn := " [ Cancel ] "
	gap := 4
	totalBtnWidth := len(yesBtn) + len(cancelBtn) + gap
	btnCol := modalRect.Min.Column + (dialogWidth-totalBtnWidth)/2

	f.DrawText(btnRow, btnCol, yesStyle, yesBtn)
	f.DrawText(btnRow, btnCol+len(yesBtn)+gap, noStyle, cancelBtn)
}
