package screen

// List is a focusable, single-column selectable list. Items are copied on
// input so callers can safely reuse their source slice.
type List struct {
	WidgetState
	items         []string
	selected      int
	offset        int
	ItemStyle     Style
	SelectedStyle Style
	EmptyText     string
	EmptyStyle    Style
}

// NewList creates a list. The initial selection is the first item, or -1 for
// an empty list.
func NewList(items []string) *List {
	list := &List{SelectedStyle: Style{Attributes: AttrReverse}, EmptyText: "(empty)"}
	list.SetItems(items)
	return list
}

// Focusable reports that lists participate in focus navigation.
func (*List) Focusable() bool { return true }

// SetBounds assigns the list bounds and immediately keeps its viewport
// aligned with the current selection.
func (l *List) SetBounds(bounds Rect) {
	if l == nil {
		return
	}
	l.WidgetState.SetBounds(bounds)
	l.ensureVisible()
}

// Items returns a copy of the current item slice.
func (l *List) Items() []string {
	if l == nil {
		return nil
	}
	return append([]string(nil), l.items...)
}

// SetItems replaces list content and clamps selection and scrolling.
func (l *List) SetItems(items []string) {
	if l == nil {
		return
	}
	l.items = append([]string(nil), items...)
	if len(l.items) == 0 {
		l.selected = -1
		l.offset = 0
		return
	}
	if l.selected < 0 {
		l.selected = 0
	}
	if l.selected >= len(l.items) {
		l.selected = len(l.items) - 1
	}
	l.ensureVisible()
}

// SelectedIndex returns the selected index, or -1 for an empty list.
func (l *List) SelectedIndex() int {
	if l == nil {
		return -1
	}
	return l.selected
}

// Selected returns the selected item and whether one exists.
func (l *List) Selected() (string, bool) {
	if l == nil || l.selected < 0 || l.selected >= len(l.items) {
		return "", false
	}
	return l.items[l.selected], true
}

// ScrollOffset returns the first item row currently visible.
func (l *List) ScrollOffset() int {
	if l == nil {
		return 0
	}
	return l.offset
}

// Handle processes list navigation while focused.
func (l *List) Handle(event Event) bool {
	if l == nil || !l.Focused() {
		return false
	}
	if len(l.items) == 0 {
		return event.Type == EventArrowUp || event.Type == EventArrowDown || event.Type == EventHome || event.Type == EventEnd || event.Type == EventPageUp || event.Type == EventPageDown
	}
	switch event.Type {
	case EventArrowUp:
		l.selectIndex(l.selected - 1)
	case EventArrowDown:
		l.selectIndex(l.selected + 1)
	case EventHome:
		l.selectIndex(0)
	case EventEnd:
		l.selectIndex(len(l.items) - 1)
	case EventPageUp:
		l.selectIndex(l.selected - max(1, l.visibleRows()))
	case EventPageDown:
		l.selectIndex(l.selected + max(1, l.visibleRows()))
	default:
		return false
	}
	return true
}

// Render draws visible items and the empty-list message without writing
// outside the frame or the widget bounds.
func (l *List) Render(frame *Frame) {
	if l == nil || frame == nil {
		return
	}
	bounds, ok := widgetFrame(frame, l.Bounds())
	if !ok {
		return
	}
	rows, columns := bounds.Size().Rows, bounds.Size().Columns
	if rows == 0 || columns == 0 {
		return
	}
	frame.ClearRegion(bounds)
	if len(l.items) == 0 {
		frame.DrawTextIn(bounds, l.EmptyText, TextOptions{Style: l.EmptyStyle})
		return
	}
	l.ensureVisible()
	for row := 0; row < rows && l.offset+row < len(l.items); row++ {
		style := l.ItemStyle
		if l.offset+row == l.selected {
			style = l.SelectedStyle
		}
		frame.DrawText(bounds.Min.Row+row, bounds.Min.Column, style, l.items[l.offset+row])
	}
}

func (l *List) selectIndex(index int) {
	if len(l.items) == 0 {
		l.selected = -1
		l.offset = 0
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(l.items) {
		index = len(l.items) - 1
	}
	l.selected = index
	l.ensureVisible()
}

func (l *List) visibleRows() int {
	rows := l.Bounds().Size().Rows
	if rows < 1 {
		return 1
	}
	return rows
}

func (l *List) ensureVisible() {
	if len(l.items) == 0 {
		l.offset = 0
		return
	}
	rows := l.visibleRows()
	if l.selected < l.offset {
		l.offset = l.selected
	}
	if l.selected >= l.offset+rows {
		l.offset = l.selected - rows + 1
	}
	maxOffset := len(l.items) - rows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if l.offset > maxOffset {
		l.offset = maxOffset
	}
	if l.offset < 0 {
		l.offset = 0
	}
}
