package views

import (
	"fmt"

	"mayfly/pkg/tui/layout"
	"mayfly/pkg/tui/terminal"
)

func (s *Screens) drawFirstRunSetup(frame *terminal.Frame, bodyRect terminal.Rect) {
	inputRects := layout.Split(layout.DirVertical, bodyRect, []layout.Constraint{
		layout.Fixed(1),
		layout.Fixed(1),
		layout.Fixed(3),
		layout.Fixed(1),
		layout.Fixed(3),
		layout.Flexible(),
	})
	frame.DrawText(inputRects[0].Min.Row, inputRects[0].Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightYellow, Attributes: terminal.AttrBold}, "FIRST-TIME SETUP: CREATE VAULT MASTER PASSWORD")
	frame.DrawText(inputRects[1].Min.Row, inputRects[1].Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightBlack, Attributes: terminal.AttrDim}, "Choose a password to encrypt your local secrets vault (~/.mayfly/vault.enc)")
	s.passInput.Draw(frame, inputRects[2])
	s.confirmPass.Draw(frame, inputRects[4])
}

func (s *Screens) drawUnlock(frame *terminal.Frame, bodyRect terminal.Rect) {
	inputRects := layout.Split(layout.DirVertical, bodyRect, []layout.Constraint{
		layout.Fixed(3),
		layout.Fixed(3),
		layout.Flexible(),
	})
	s.passInput.Draw(frame, inputRects[1])
}

func (s *Screens) drawEditSecret(frame *terminal.Frame, bodyRect terminal.Rect) {
	inputRects := layout.Split(layout.DirVertical, bodyRect, []layout.Constraint{
		layout.Fixed(2),
		layout.Fixed(3),
		layout.Fixed(1),
		layout.Fixed(3),
		layout.Flexible(),
	})
	title := "ADD NEW SECRET"
	if s.editOrigName != "" {
		title = fmt.Sprintf("EDIT SECRET '%s'", s.editOrigName)
	}
	frame.DrawText(inputRects[0].Min.Row+1, inputRects[0].Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}, title)
	s.secretName.Draw(frame, inputRects[1])
	s.secretValue.Draw(frame, inputRects[3])
}

func (s *Screens) drawDeleteConfirm(frame *terminal.Frame, bodyRect terminal.Rect) {
	s.secretsList.Draw(frame, bodyRect)
	s.confirmDlg.Draw(frame, bodyRect)
}

func (s *Screens) drawBackup(frame *terminal.Frame, bodyRect terminal.Rect) {
	inputRects := layout.Split(layout.DirVertical, bodyRect, []layout.Constraint{
		layout.Fixed(2),
		layout.Fixed(3),
		layout.Flexible(),
	})
	frame.DrawText(inputRects[0].Min.Row+1, inputRects[0].Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}, "EXPORT ENCRYPTED VAULT BACKUP")
	s.backupPath.Draw(frame, inputRects[1])
}

func (s *Screens) drawInitProject(frame *terminal.Frame, bodyRect terminal.Rect, bounds terminal.Rect) {
	s.projectGrid.Draw(frame, bodyRect)
	dialogWidth := 64
	if dialogWidth > bounds.Max.Column-bounds.Min.Column-6 {
		dialogWidth = bounds.Max.Column - bounds.Min.Column - 6
	}
	dialogHeight := 13
	dialogTop := bodyRect.Min.Row + (bodyRect.Max.Row-bodyRect.Min.Row-dialogHeight)/2
	dialogLeft := bodyRect.Min.Column + (bodyRect.Max.Column-bodyRect.Min.Column-dialogWidth)/2
	dialogRect := terminal.NewRect(dialogTop, dialogLeft, dialogHeight, dialogWidth)

	frame.FillRect(dialogRect, ' ', terminal.Style{})
	frame.DrawBox(dialogRect, terminal.Style{Foreground: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}, "INITIALIZE NEW PROJECT VAULT")
	frame.DrawText(dialogRect.Min.Row+2, dialogRect.Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightWhite}, "Select directory to initialize with MayFly:")

	opt1Style := terminal.Style{Foreground: terminal.ColorBrightWhite}
	opt1Prefix := "  [1] "
	if s.initChoice == 0 {
		opt1Style = terminal.Style{Foreground: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}
		opt1Prefix = "► [1] "
	}
	currDisp := s.currentDir
	if len(currDisp) > dialogWidth-26 {
		currDisp = "..." + currDisp[len(currDisp)-(dialogWidth-29):]
	}
	frame.DrawText(dialogRect.Min.Row+4, dialogRect.Min.Column+3, opt1Style, fmt.Sprintf("%sCurrent Directory: %s", opt1Prefix, currDisp))

	opt2Style := terminal.Style{Foreground: terminal.ColorBrightWhite}
	opt2Prefix := "  [2] "
	if s.initChoice == 1 {
		opt2Style = terminal.Style{Foreground: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}
		opt2Prefix = "► [2] "
	}
	frame.DrawText(dialogRect.Min.Row+6, dialogRect.Min.Column+3, opt2Style, opt2Prefix+"Custom Directory Path:")

	inputRect := terminal.NewRect(dialogRect.Min.Row+7, dialogRect.Min.Column+7, 3, dialogRect.Max.Column-dialogRect.Min.Column-10)
	s.customInitPath.Draw(frame, inputRect)

	frame.DrawText(dialogRect.Min.Row+11, dialogRect.Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightBlack}, "[Tab/1/2] Select · [Enter] Confirm · [Esc] Cancel")
}

func (s *Screens) drawDeleteProjectConfirm(frame *terminal.Frame, bodyRect terminal.Rect) {
	s.projectGrid.Draw(frame, bodyRect)
	s.confirmDlg.Draw(frame, bodyRect)
}

func (s *Screens) drawDeleteProjectPassword(frame *terminal.Frame, bodyRect terminal.Rect, bounds terminal.Rect) {
	s.projectGrid.Draw(frame, bodyRect)
	dialogWidth := len(s.deleteProjName) + 38
	if dialogWidth < 56 {
		dialogWidth = 56
	}
	if dialogWidth > bounds.Max.Column-bounds.Min.Column-6 {
		dialogWidth = bounds.Max.Column - bounds.Min.Column - 6
	}
	dialogHeight := 8
	dialogTop := bodyRect.Min.Row + (bodyRect.Max.Row-bodyRect.Min.Row-dialogHeight)/2
	dialogLeft := bodyRect.Min.Column + (bodyRect.Max.Column-bodyRect.Min.Column-dialogWidth)/2
	passBox := terminal.NewRect(dialogTop, dialogLeft, dialogHeight, dialogWidth)

	frame.FillRect(passBox, ' ', terminal.Style{})
	frame.DrawBox(passBox, terminal.Style{Foreground: terminal.ColorBrightRed, Attributes: terminal.AttrBold}, "CONFIRM PROJECT VAULT DELETION")
	frame.DrawText(passBox.Min.Row+1, passBox.Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightYellow}, fmt.Sprintf("Enter master password to delete '%s':", s.deleteProjName))
	inputRect := terminal.NewRect(passBox.Min.Row+3, passBox.Min.Column+3, 3, passBox.Max.Column-passBox.Min.Column-6)
	s.deletePassInput.Draw(frame, inputRect)
	frame.DrawText(passBox.Min.Row+6, passBox.Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightBlack}, "[Enter] Confirm Delete · [Esc] Cancel")
}
