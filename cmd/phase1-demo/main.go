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
	header := screen.NewRegion("header", screen.Size{Rows: 1, Columns: 1}, screen.Size{Rows: 1, Columns: 44})
	body := screen.NewRegion("body", screen.Size{Rows: 1, Columns: 1}, screen.Size{Rows: 3, Columns: 44})
	footer := screen.NewRegion("footer", screen.Size{Rows: 1, Columns: 1}, screen.Size{Rows: 1, Columns: 44})
	for _, placement := range screen.Arrange(screen.Vertical(header, screen.Flexible(body), footer), screen.NewRect(1, 1, 6, 46)) {
		region := placement.Node.(*screen.Region)
		switch region.Name {
		case "header":
			frame.DrawTextIn(placement.Bounds, "MayFly — Phase 5\nDeterministic layout", screen.TextOptions{
				Horizontal: screen.AlignCenter,
				Vertical:   screen.AlignMiddle,
				Style: screen.Style{
					Foreground: screen.ColorCyan,
					Attributes: screen.AttrBold,
				},
			})
		case "body":
			frame.DrawTextIn(placement.Bounds, "ANSI styling · UTF-8 cells · clipped regions", screen.TextOptions{
				Horizontal: screen.AlignCenter,
				Vertical:   screen.AlignMiddle,
				Padding:    screen.Padding{Left: 1, Right: 1},
			})
		case "footer":
			frame.DrawText(placement.Bounds.Min.Row, placement.Bounds.Min.Column, screen.Style{}, "OPENAI_API_KEY")
			frame.DrawMaskedText(placement.Bounds.Min.Row, placement.Bounds.Min.Column+16, screen.Style{Foreground: screen.ColorBrightYellow}, "display-only-secret")
		}
	}

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
