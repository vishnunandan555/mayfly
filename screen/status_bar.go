package screen

// StatusBar displays a short message on the left and optional hints on the
// right. It is non-focusable.
type StatusBar struct {
	WidgetState
	Message    string
	Hints      string
	Style      Style
	HintsStyle Style
}

// NewStatusBar creates an empty status bar.
func NewStatusBar() *StatusBar {
	return &StatusBar{}
}

func (*StatusBar) Focusable() bool { return false }

func (*StatusBar) Handle(Event) bool { return false }

// Render draws message and hints in one row. The two fields are clipped if
// their combined width exceeds the widget width.
func (s *StatusBar) Render(frame *Frame) {
	if s == nil || frame == nil {
		return
	}
	bounds, ok := widgetFrame(frame, s.Bounds())
	if !ok || bounds.Size().Rows == 0 || bounds.Size().Columns == 0 {
		return
	}
	frame.ClearRegion(bounds)
	frame.DrawText(bounds.Min.Row, bounds.Min.Column, s.Style, s.Message)
	if s.Hints == "" {
		return
	}
	hintsWidth := TextWidth(s.Hints)
	if hintsWidth > bounds.Size().Columns {
		hintsWidth = bounds.Size().Columns
	}
	frame.DrawTextIn(NewRect(bounds.Min.Row, bounds.Max.Column-hintsWidth, 1, hintsWidth), s.Hints, TextOptions{Style: s.HintsStyle, Horizontal: AlignRight})
}
