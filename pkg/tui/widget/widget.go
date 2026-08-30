package widget

import (
	"mayfly/pkg/tui/terminal"
)

// Widget is the core interface for all TUI components.
type Widget interface {
	Draw(frame *terminal.Frame, bounds terminal.Rect)
	HandleKey(event terminal.KeyEvent) bool
}
