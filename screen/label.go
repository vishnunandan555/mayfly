package screen

// Label is a non-focusable static text widget.
type Label struct {
	WidgetState
	Text       string
	Style      Style
	Horizontal HorizontalAlignment
	Vertical   VerticalAlignment
}

// NewLabel creates a label with default top-left alignment.
func NewLabel(text string) *Label {
	return &Label{Text: text}
}

// Focusable reports that labels do not receive keyboard focus.
func (*Label) Focusable() bool { return false }

// Handle ignores input and reports it was not consumed.
func (*Label) Handle(Event) bool { return false }

// Render draws the label text inside its assigned bounds.
func (l *Label) Render(frame *Frame) {
	if l == nil || frame == nil {
		return
	}
	frame.ClearRegion(l.Bounds())
	frame.DrawTextIn(l.Bounds(), l.Text, TextOptions{
		Style:      l.Style,
		Horizontal: l.Horizontal,
		Vertical:   l.Vertical,
	})
}
