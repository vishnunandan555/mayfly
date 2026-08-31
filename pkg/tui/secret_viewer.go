package tui

import (
	"os"
	"os/signal"
	"strings"
	"time"

	"mayfly/pkg/tui/terminal"
)

// ShowSecretViewer launches an ephemeral, alternate-screen TUI viewer for a single secret.
// When closed, it restores the terminal and leaves zero plaintext in the scrollback history.
func ShowSecretViewer(projectName, secretName, secretValue string) error {
	rawState, err := terminal.EnableRaw(os.Stdin)
	if err != nil {
		return err
	}
	defer func() {
		_ = terminal.Restore(os.Stdin, rawState)
	}()

	sz, err := terminal.GetSize(os.Stdout)
	if err != nil {
		sz = terminal.Size{Rows: 24, Columns: 80}
	}

	term := terminal.NewTerminal(os.Stdout, sz)
	term.EnterAltScreen()
	defer term.ExitAltScreen()

	// Listen for window resize signals
	sigCh := make(chan os.Signal, 1)
	terminal.NotifyResize(sigCh)
	defer signal.Stop(sigCh)

	// Key reader
	keyCh := make(chan []terminal.KeyEvent, 16)
	errCh := make(chan error, 1)

	go func() {
		buf := make([]byte, 128)
		parser := terminal.NewParser()
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			if n > 0 {
				events := parser.Feed(buf[:n])
				if len(events) > 0 {
					keyCh <- events
				}
			}
		}
	}()

	var statusMsg string
	var statusTimer time.Time

	render := func() {
		frame := terminal.NewFrame(sz)
		DrawSecretViewerFrame(frame, sz, projectName, secretName, secretValue, statusMsg)
		_ = term.Render(frame)
	}

	render()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if statusMsg != "" && !statusTimer.IsZero() && time.Since(statusTimer) > 3*time.Second {
				statusMsg = ""
				render()
			}

		case <-sigCh:
			newSz, err := terminal.GetSize(os.Stdout)
			if err == nil {
				sz = newSz
				term = terminal.NewTerminal(os.Stdout, sz)
				render()
			}

		case events := <-keyCh:
			for _, ev := range events {
				switch ev.Type {
				case terminal.KeyEscape, terminal.KeyEnter:
					return nil
				case terminal.KeyRune:
					switch ev.Rune {
					case 'q', 'Q':
						return nil
					case 'c', 'C':
						_ = terminal.CopyToClipboard(secretValue, nil)
						statusMsg = "✓ Copied to clipboard!"
						statusTimer = time.Now()
						render()
					}
				}
			}

		case err := <-errCh:
			return err
		}
	}
}

// DrawSecretViewerFrame renders the visual secret modal onto a terminal frame.
func DrawSecretViewerFrame(frame *terminal.Frame, sz terminal.Size, projName, secName, secVal, statusMsg string) {
	// Fill background
	bgStyle := terminal.Style{Foreground: terminal.ColorDefault, Background: terminal.ColorDefault}
	frame.FillRect(terminal.NewRect(0, 0, sz.Rows, sz.Columns), ' ', bgStyle)

	// Calculate dialog dimensions
	valLines := strings.Split(secVal, "\n")
	contentHeight := len(valLines)
	if contentHeight > 10 {
		contentHeight = 10
	}

	boxHeight := 10 + contentHeight
	if boxHeight > sz.Rows-2 {
		boxHeight = sz.Rows - 2
	}
	if boxHeight < 9 {
		boxHeight = 9
	}

	boxWidth := 72
	if boxWidth > sz.Columns-4 {
		boxWidth = sz.Columns - 4
	}
	if boxWidth < 40 {
		boxWidth = 40
	}

	startRow := (sz.Rows - boxHeight) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (sz.Columns - boxWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	boxRect := terminal.NewRect(startRow, startCol, boxHeight, boxWidth)
	borderStyle := terminal.Style{Foreground: terminal.ColorCyan, Attributes: terminal.AttrBold}
	frame.DrawBox(boxRect, borderStyle, " MAYFLY SECRET VIEWER ")

	// Project & Key details
	curRow := startRow + 2
	labelStyle := terminal.Style{Foreground: terminal.ColorYellow, Attributes: terminal.AttrBold}
	valueStyle := terminal.Style{Foreground: terminal.ColorBrightWhite}

	if projName != "" {
		frame.DrawText(curRow, startCol+4, labelStyle, "Project : ")
		frame.DrawText(curRow, startCol+14, valueStyle, projName)
		curRow++
	}

	frame.DrawText(curRow, startCol+4, labelStyle, "Secret  : ")
	frame.DrawText(curRow, startCol+14, terminal.Style{Foreground: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}, secName)
	curRow += 2

	// Secret Value Display Area
	frame.DrawText(curRow, startCol+4, labelStyle, "Value   : ")
	valStyle := terminal.Style{Foreground: terminal.ColorBrightGreen, Attributes: terminal.AttrBold}

	maxValWidth := boxWidth - 18
	if maxValWidth < 10 {
		maxValWidth = 10
	}

	for i, line := range valLines {
		if i >= contentHeight {
			frame.DrawText(curRow, startCol+14, terminal.Style{Foreground: terminal.ColorBrightBlack, Attributes: terminal.AttrDim}, "... (truncated)")
			curRow++
			break
		}
		displayLine := line
		if len(displayLine) > maxValWidth {
			displayLine = displayLine[:maxValWidth-3] + "..."
		}
		frame.DrawText(curRow, startCol+14, valStyle, displayLine)
		curRow++
	}

	// Status badge if present
	if statusMsg != "" {
		curRow++
		badgeStyle := terminal.Style{Foreground: terminal.ColorBrightGreen, Attributes: terminal.AttrBold}
		frame.DrawText(curRow, startCol+4, badgeStyle, statusMsg)
	}

	// Bottom action helper bar
	footerRow := boxRect.Max.Row - 2
	footerStyle := terminal.Style{Foreground: terminal.ColorBrightBlack, Attributes: terminal.AttrDim}
	keyStyle := terminal.Style{Foreground: terminal.ColorYellow, Attributes: terminal.AttrBold}

	frame.DrawText(footerRow, startCol+4, keyStyle, "[C]")
	frame.DrawText(footerRow, startCol+8, footerStyle, "Copy to clipboard    ")
	frame.DrawText(footerRow, startCol+29, keyStyle, "[Q / Esc / Enter]")
	frame.DrawText(footerRow, startCol+47, footerStyle, "Close & Erase")

}
