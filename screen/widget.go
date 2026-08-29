package screen

// Widget is the common contract for stateful pieces of a MayFly screen.
// Widgets receive already-decoded input events and render into a logical
// Frame; they do not read terminals, emit escape sequences, or access process
// global state.
type Widget interface {
	Bounds() Rect
	SetBounds(Rect)
	Handle(Event) bool
	Render(*Frame)
	Focusable() bool
	Focused() bool
	SetFocused(bool)
}

// WidgetState contains bounds and focus state for widgets that do not need
// specialized behavior. It is intended to be embedded in custom widgets.
type WidgetState struct {
	bounds   Rect
	focused  bool
	disabled bool
}

// Bounds returns the widget's assigned half-open rectangle.
func (s *WidgetState) Bounds() Rect {
	if s == nil {
		return Rect{}
	}
	return s.bounds
}

// SetBounds assigns a widget rectangle. It does not clip to a terminal; the
// Frame supplied to Render provides the final clipping boundary.
func (s *WidgetState) SetBounds(bounds Rect) {
	if s == nil {
		return
	}
	s.bounds = normalizeWidgetRect(bounds)
}

// Focused reports whether the widget currently owns focus.
func (s *WidgetState) Focused() bool {
	return s != nil && s.focused
}

// SetFocused changes focus state without changing any other widget state.
func (s *WidgetState) SetFocused(focused bool) {
	if s != nil {
		s.focused = focused && !s.disabled
	}
}

// Enabled reports whether the widget may receive focus. It is optional in the
// Widget interface but is honored by Application when present.
func (s *WidgetState) Enabled() bool {
	return s != nil && !s.disabled
}

// SetEnabled enables or disables a widget. Disabling also removes focus.
func (s *WidgetState) SetEnabled(enabled bool) {
	if s == nil {
		return
	}
	s.disabled = !enabled
	if !enabled {
		s.focused = false
	}
}

func normalizeWidgetRect(bounds Rect) Rect {
	if bounds.Max.Row < bounds.Min.Row {
		bounds.Max.Row = bounds.Min.Row
	}
	if bounds.Max.Column < bounds.Min.Column {
		bounds.Max.Column = bounds.Min.Column
	}
	return bounds
}

func widgetFrame(frame *Frame, bounds Rect) (Rect, bool) {
	if frame == nil {
		return Rect{}, false
	}
	area := normalizeWidgetRect(bounds).Intersect(frame.Bounds())
	area = normalizeWidgetRect(area)
	return area, !area.Empty()
}
