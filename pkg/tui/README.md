# 🖥️ MayFly Terminal UI Engine (`pkg/tui`)

[![Zero Dependencies](https://img.shields.io/badge/Dependencies-0%20(Pure%20Stdlib)-brightgreen)](file:///home/vishnunandan555/Projects/mayfly/STDLIB.md)
[![Package Killer](https://img.shields.io/badge/Replaces-bubbletea%20%7C%20tview%20%7C%20termenv-red)](file:///home/vishnunandan555/Projects/mayfly/HACKATHON_WINNING_STRATEGY.md)
[![Architecture](https://img.shields.io/badge/Engine-Double--Buffered%202D%20Canvas-blue)](file:///home/vishnunandan555/Projects/mayfly/pkg/tui/terminal/terminal.go)

`pkg/tui` is a **complete, double-buffered, zero-dependency Terminal User Interface framework** built from first principles with raw POSIX `termios` system calls and standard Go primitives.

---

## 📦 Packages Replaced (Package Killer)

| Third-Party Package | Typical Downloads | Standard Library Replacement in `pkg/tui` |
|---|---|---|
| `github.com/charmbracelet/bubbletea` | **2M+/week** | Low-level event loop with double-buffered frame diff engine. |
| `github.com/rivo/tview` / `termbox-go` | **1M+/week** | Double-buffered 2D cell matrix with clean ANSI SGR attribute caching. |
| `golang.org/x/term` | **8M+/week** | Direct `syscall.SYS_IOCTL`, `syscall.TCGETS`, `syscall.TCSETS` (Linux/macOS) and Windows Console API. |
| `github.com/muesli/termenv` | **1M+/week** | Hand-rolled ANSI escape sequence builders and East Asian character width math (`RuneWidth`). |

---

## 📐 Subpackage Architecture

```
pkg/tui/
├── terminal/   # Raw OS termios syscalls, AltScreen controller, double-buffered ANSI diff renderer
├── layout/     # Flexible constraint solver (Fixed, Flexible, Min, Percent)
├── widget/     # Reusable UI widgets (ProjectCardGrid, List, TextInput, ConfirmDialog, StatusBar)
├── views/      # Stateful interactive screens & modal dialog manager
└── tui.go      # Main event loop & signal/ticker dispatcher
```

---

## 🎨 Core Engine Highlights

1. **Direct OS System Calls (`pkg/tui/terminal`):**
   * **Linux/macOS:** Direct `syscall.SYS_IOCTL` manipulate `termios` line discipline into raw non-canonical mode, disabling `ECHO` and `ICANON`.
   * **Windows:** Direct `syscall.SyscallN` calls to `GetConsoleMode` and `SetConsoleMode` with virtual terminal processing (`ENABLE_VIRTUAL_TERMINAL_INPUT`, `ENABLE_VIRTUAL_TERMINAL_PROCESSING`).
2. **Double-Buffered Frame Diff Engine:**
   * Keeps two 2D cell grids in memory (`curFrame` and `nextFrame`).
   * Only outputs ANSI escape codes for cells that changed between renders, eliminating terminal flicker.
3. **Modal Dialog Background Isolation:**
   * `FillRect` fills modal interior boundaries to prevent underlying text bleed-through.
4. **Live Time-Based Tick System:**
   * 500ms background ticker drives real-time auto-hide countdowns without user key input.

---

## 🚀 API Reference & Usage

### 1. Launching the Full TUI Session
```go
package main

import (
	"mayfly/pkg/application"
	"mayfly/pkg/tui"
)

func main() {
	svc := application.NewService(nil, nil, nil, nil)

	// Launches full-screen alternate-screen interactive dashboard
	err := tui.Run(svc, tui.Options{
		CurrentDir: "/home/user/my-project",
	})
	if err != nil {
		panic(err)
	}
}
```

### 2. Using the 2D Layout Constraint Solver
```go
package main

import (
	"fmt"
	"mayfly/pkg/tui/layout"
	"mayfly/pkg/tui/terminal"
)

func main() {
	screenBounds := terminal.NewRect(0, 0, 24, 80) // 24 rows, 80 columns

	// Split screen vertically into Header (3 rows), Body (Flexible), Footer (1 row)
	sections := layout.Split(layout.DirVertical, screenBounds, []layout.Constraint{
		layout.Fixed(3),
		layout.Flexible(),
		layout.Fixed(1),
	})

	fmt.Printf("Header Rect: %+v\n", sections[0])
	fmt.Printf("Body Rect:   %+v\n", sections[1])
	fmt.Printf("Footer Rect: %+v\n", sections[2])
}
```

---

## 🧪 Testing & Verification

Run the complete TUI subsystem tests:
```bash
go test -race -v ./pkg/tui/...
```
