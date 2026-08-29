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
	frame.DrawBox(frame.Bounds(), screen.Style{})
	frame.DrawTextIn(screen.NewRect(1, 1, 3, 46), "MayFly — Phase 4\nStyled logical screen", screen.TextOptions{
		Horizontal: screen.AlignCenter,
		Vertical:   screen.AlignMiddle,
		Style: screen.Style{
			Foreground: screen.ColorCyan,
			Attributes: screen.AttrBold,
		},
	})
	frame.DrawTextIn(screen.NewRect(4, 2, 2, 44), "ANSI styling · UTF-8 cells · clipped regions", screen.TextOptions{
		Horizontal: screen.AlignCenter,
		Padding:    screen.Padding{Left: 1, Right: 1},
	})
	frame.DrawText(6, 2, screen.Style{
		Foreground: screen.ColorCyan,
	}, "This static demo exits after rendering.")
	frame.DrawText(7, 2, screen.Style{}, "OPENAI_API_KEY")
	frame.DrawMaskedText(7, 18, screen.Style{Foreground: screen.ColorBrightYellow}, "display-only-secret")

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
