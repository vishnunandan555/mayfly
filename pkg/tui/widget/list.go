package widget

import (
	"fmt"

	"mayfly/pkg/tui/terminal"
)

type ListItem struct {
	Primary   string
	Secondary string
	Extra     string
	Data      interface{}
}

type List struct {
	Title    string
	Items    []ListItem
	Selected int
	Offset   int
	Focused  bool
}

func NewList(title string) *List {
	return &List{
		Title:   title,
		Focused: true,
	}
}

func (l *List) SetItems(items []ListItem) {
	l.Items = items
	if l.Selected >= len(items) {
		l.Selected = len(items) - 1
	}
	if l.Selected < 0 && len(items) > 0 {
		l.Selected = 0
	}
}

func (l *List) SelectedItem() *ListItem {
	if len(l.Items) == 0 || l.Selected < 0 || l.Selected >= len(l.Items) {
		return nil
	}
	return &l.Items[l.Selected]
}

func (l *List) HandleKey(event terminal.KeyEvent) bool {
	if !l.Focused || len(l.Items) == 0 {
		return false
	}

	switch event.Type {
	case terminal.KeyUp:
		if l.Selected > 0 {
			l.Selected--
			return true
		}
	case terminal.KeyDown:
		if l.Selected < len(l.Items)-1 {
			l.Selected++
			return true
		}
	case terminal.KeyHome:
		l.Selected = 0
		return true
	case terminal.KeyEnd:
		l.Selected = len(l.Items) - 1
		return true
	}

	return false
}

func (l *List) Draw(f *terminal.Frame, bounds terminal.Rect) {
	if bounds.Max.Row <= bounds.Min.Row || bounds.Max.Column <= bounds.Min.Column {
		return
	}

	boxStyle := terminal.Style{Foreground: terminal.ColorBrightBlack}
	if l.Focused {
		boxStyle.Foreground = terminal.ColorBrightCyan
	}

	titleText := l.Title
	if len(l.Items) > 0 {
		titleText = fmt.Sprintf("%s (%d)", l.Title, len(l.Items))
	}
	f.DrawBox(bounds, boxStyle, titleText)

	visibleRows := bounds.Max.Row - bounds.Min.Row - 2
	if visibleRows <= 0 {
		return
	}

	// Keep selection within viewport
	if l.Selected < l.Offset {
		l.Offset = l.Selected
	} else if l.Selected >= l.Offset+visibleRows {
		l.Offset = l.Selected - visibleRows + 1
	}

	if l.Offset < 0 {
		l.Offset = 0
	}
	if len(l.Items) > 0 && l.Offset >= len(l.Items) {
		l.Offset = len(l.Items) - 1
	}

	contentWidth := bounds.Max.Column - bounds.Min.Column - 4
	if contentWidth <= 0 {
		return
	}

	if len(l.Items) == 0 {
		f.DrawText(bounds.Min.Row+2, bounds.Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightBlack, Attributes: terminal.AttrDim}, "No items found.")
		return
	}

	for i := 0; i < visibleRows; i++ {
		itemIdx := l.Offset + i
		if itemIdx >= len(l.Items) {
			break
		}

		item := l.Items[itemIdx]
		row := bounds.Min.Row + 1 + i
		col := bounds.Min.Column + 2

		isSelected := itemIdx == l.Selected

		prefix := "  "
		itemStyle := terminal.Style{Foreground: terminal.ColorBrightWhite}
		secStyle := terminal.Style{Foreground: terminal.ColorBrightBlack}

		if isSelected && l.Focused {
			prefix = "▶ "
			itemStyle = terminal.Style{Foreground: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}
			secStyle = terminal.Style{Foreground: terminal.ColorCyan}
		}

		f.DrawText(row, col, itemStyle, prefix+item.Primary)

		if item.Secondary != "" {
			secCol := col + len(prefix) + len(item.Primary) + 3
			if secCol < bounds.Max.Column-2 {
				f.DrawText(row, secCol, secStyle, item.Secondary)
			}
		}

		if item.Extra != "" {
			extraLen := len(item.Extra)
			extraCol := bounds.Max.Column - extraLen - 3
			if extraCol > col+len(item.Primary)+len(item.Secondary)+6 {
				f.DrawText(row, extraCol, terminal.Style{Foreground: terminal.ColorBrightYellow}, item.Extra)
			}
		}
	}
}
