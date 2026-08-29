package screen

// Panel is a non-focusable bordered container with an optional title and
// child widget. The child receives the inner bounds after the border and
// padding are removed.
type Panel struct {
	WidgetState
	Title       string
	BorderStyle Style
	TitleStyle  Style
	Padding     Padding
	Child       Widget
}

// NewPanel creates a panel containing child.
func NewPanel(title string, child Widget) *Panel {
	return &Panel{Title: title, Child: child}
}

// Box is an alias for Panel for callers that prefer the shorter name.
type Box = Panel

// NewBox creates a titled panel.
func NewBox(title string, child Widget) *Panel {
	return NewPanel(title, child)
}

func (*Panel) Focusable() bool { return false }

// Handle delegates input to the child, if present.
func (p *Panel) Handle(event Event) bool {
	if p == nil || p.Child == nil {
		return false
	}
	return p.Child.Handle(event)
}

// SetBounds assigns the panel bounds and updates the child's inner bounds.
func (p *Panel) SetBounds(bounds Rect) {
	if p == nil {
		return
	}
	p.WidgetState.SetBounds(bounds)
	p.updateChildBounds()
}

// Render draws the border/title and then the child.
func (p *Panel) Render(frame *Frame) {
	if p == nil || frame == nil {
		return
	}
	bounds := p.Bounds()
	frame.ClearRegion(bounds)
	frame.DrawBox(bounds, p.BorderStyle)
	if p.Title != "" && bounds.Size().Rows > 0 && bounds.Size().Columns > 2 {
		titleBounds := NewRect(bounds.Min.Row, bounds.Min.Column+1, 1, bounds.Size().Columns-2)
		frame.DrawTextIn(titleBounds, " "+p.Title+" ", TextOptions{Style: p.TitleStyle})
	}
	p.updateChildBounds()
	if p.Child != nil {
		p.Child.Render(frame)
	}
}

func (p *Panel) updateChildBounds() {
	if p == nil || p.Child == nil {
		return
	}
	inner := p.Bounds()
	if inner.Size().Rows >= 2 {
		inner.Min.Row++
		inner.Max.Row--
	}
	if inner.Size().Columns >= 2 {
		inner.Min.Column++
		inner.Max.Column--
	}
	inner = p.Padding.Inset(inner)
	p.Child.SetBounds(normalizeWidgetRect(inner))
}
