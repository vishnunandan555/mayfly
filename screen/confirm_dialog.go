package screen

// DialogResult is the state of a ConfirmDialog.
type DialogResult uint8

const (
	DialogPending DialogResult = iota
	DialogYes
	DialogNo
	DialogCancelled
)

// ConfirmDialog is a focusable yes/no dialog. It does not perform any action;
// callers inspect Result after handling events.
type ConfirmDialog struct {
	WidgetState
	Title         string
	Message       string
	YesLabel      string
	NoLabel       string
	Result        DialogResult
	YesSelected   bool
	BorderStyle   Style
	TitleStyle    Style
	MessageStyle  Style
	SelectedStyle Style
}

// NewConfirmDialog creates a dialog with Yes selected by default.
func NewConfirmDialog(title, message string) *ConfirmDialog {
	return &ConfirmDialog{
		Title:         title,
		Message:       message,
		YesLabel:      "Yes",
		NoLabel:       "No",
		Result:        DialogPending,
		YesSelected:   true,
		SelectedStyle: Style{Attributes: AttrReverse},
	}
}

func (*ConfirmDialog) Focusable() bool { return true }

// Reset returns the dialog to its initial pending/Yes state.
func (d *ConfirmDialog) Reset() {
	if d == nil {
		return
	}
	d.Result = DialogPending
	d.YesSelected = true
}

// Handle changes the selection or resolves the dialog. Events are consumed
// only while the dialog is focused and pending.
func (d *ConfirmDialog) Handle(event Event) bool {
	if d == nil || !d.Focused() || d.Result != DialogPending {
		return false
	}
	switch event.Type {
	case EventArrowLeft, EventArrowUp:
		if d.NoLabel != "" {
			d.YesSelected = true
		}
	case EventArrowRight, EventArrowDown:
		if d.NoLabel != "" {
			d.YesSelected = false
		}
	case EventRune:
		if event.Rune == ' ' {
			if d.YesSelected || d.NoLabel == "" {
				d.Result = DialogYes
			} else {
				d.Result = DialogNo
			}
			return true
		}
		switch event.Rune {
		case 'y', 'Y':
			d.YesSelected = true
		case 'n', 'N':
			if d.NoLabel != "" {
				d.YesSelected = false
			}
		default:
			if d.NoLabel == "" {
				d.Result = DialogYes
				return true
			}
			return false
		}
	case EventEnter:
		if d.YesSelected || d.NoLabel == "" {
			d.Result = DialogYes
		} else {
			d.Result = DialogNo
		}
	case EventEscape:
		d.Result = DialogCancelled
	default:
		return false
	}
	return true
}

// Render draws the dialog contents inside its assigned bounds. Small bounds
// are handled by clipping through Frame's bounded drawing methods.
func (d *ConfirmDialog) Render(frame *Frame) {
	if d == nil || frame == nil {
		return
	}
	bounds, ok := widgetFrame(frame, d.Bounds())
	if !ok {
		return
	}
	frame.ClearRegion(bounds)
	frame.DrawBox(bounds, d.BorderStyle)
	if d.Title != "" && bounds.Size().Rows > 0 && bounds.Size().Columns > 2 {
		frame.DrawTextIn(NewRect(bounds.Min.Row, bounds.Min.Column+1, 1, bounds.Size().Columns-2), " "+d.Title+" ", TextOptions{Style: d.TitleStyle})
	}
	inner := bounds
	if inner.Size().Rows > 2 {
		inner.Min.Row++
		inner.Max.Row--
	}
	if inner.Size().Columns > 2 {
		inner.Min.Column++
		inner.Max.Column--
	}
	if inner.Empty() {
		return
	}
	messageRows := inner.Size().Rows
	if messageRows > 2 {
		messageRows -= 1
	}
	frame.DrawTextIn(NewRect(inner.Min.Row, inner.Min.Column, messageRows, inner.Size().Columns), d.Message, TextOptions{Style: d.MessageStyle, Vertical: AlignMiddle})
	if inner.Size().Rows == 0 {
		return
	}
	buttonRow := inner.Max.Row - 1
	var buttons string
	if d.NoLabel != "" {
		buttons = "[ " + d.YesLabel + " ]   [ " + d.NoLabel + " ]"
	} else {
		buttons = "[ " + d.YesLabel + " ]"
	}
	frame.DrawTextIn(NewRect(buttonRow, inner.Min.Column, 1, inner.Size().Columns), buttons, TextOptions{Horizontal: AlignCenter})
	// Repaint the selected button with its style. This deliberately uses fixed
	// label positions, keeping selection behavior deterministic.
	if d.YesSelected || d.NoLabel == "" {
		start := inner.Min.Column + (inner.Size().Columns-TextWidth(buttons))/2 + 2
		frame.DrawText(buttonRow, start, d.SelectedStyle, d.YesLabel)
	} else {
		start := inner.Min.Column + (inner.Size().Columns-TextWidth(buttons))/2 + 2 + TextWidth(d.YesLabel) + 7
		frame.DrawText(buttonRow, start, d.SelectedStyle, d.NoLabel)
	}
}
