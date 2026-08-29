package main

import (
	"fmt"
	"os"

	"mayfly/screen"
)

func main() {
	// The viewport is supplied by the application. A later phase can obtain
	// the real terminal size without changing the screen package API.
	terminal := screen.NewTerminal(os.Stdout, screen.Size{Rows: 8, Columns: 48})
	frame := screen.NewFrame(screen.Size{Rows: 8, Columns: 48})
	frame.DrawText(1, 2, screen.Style{
		Foreground: screen.ColorCyan,
		Attributes: screen.AttrBold,
	}, "MayFly — Phase 1")
	frame.DrawText(3, 2, screen.Style{}, "Terminal output foundation")
	frame.DrawText(4, 2, screen.Style{}, "Rendering is clipped to the configured viewport.")

	if err := terminal.ClearScreen(); err != nil {
		fail(err)
	}
	if err := terminal.Render(frame); err != nil {
		fail(err)
	}
	if err := terminal.Reset(); err != nil {
		fail(err)
	}
	if err := terminal.Flush(); err != nil {
		fail(err)
	}
}

func fail(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
