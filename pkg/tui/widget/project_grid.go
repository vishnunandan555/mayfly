package widget

import (
	"fmt"
	"path/filepath"

	"mayfly/pkg/domain"
	"mayfly/pkg/tui/terminal"
)

type ProjectCard struct {
	Project     domain.Project
	SecretCount int
	IsCurrent   bool
}

type ProjectCardGrid struct {
	Title    string
	Cards    []ProjectCard
	Selected int
	Focused  bool
}

func NewProjectCardGrid(title string) *ProjectCardGrid {
	return &ProjectCardGrid{
		Title:   title,
		Focused: true,
	}
}

func (g *ProjectCardGrid) SetCards(cards []ProjectCard) {
	g.Cards = cards
	currentIdx := -1
	for i, c := range cards {
		if c.IsCurrent {
			currentIdx = i
			break
		}
	}
	if currentIdx != -1 {
		g.Selected = currentIdx
	} else {
		g.Selected = -1
	}
}

func (g *ProjectCardGrid) SelectedCard() *ProjectCard {
	if len(g.Cards) == 0 || g.Selected < 0 || g.Selected >= len(g.Cards) {
		return nil
	}
	return &g.Cards[g.Selected]
}

func (g *ProjectCardGrid) HandleKey(event terminal.KeyEvent) bool {
	if !g.Focused || len(g.Cards) == 0 {
		return false
	}

	cols := 2 // 2-column grid layout

	if g.Selected < 0 {
		switch event.Type {
		case terminal.KeyLeft, terminal.KeyRight, terminal.KeyUp, terminal.KeyDown, terminal.KeyHome, terminal.KeyEnd:
			g.Selected = 0
			return true
		}
		return false
	}

	switch event.Type {
	case terminal.KeyLeft:
		if g.Selected%cols > 0 {
			g.Selected--
			return true
		}
	case terminal.KeyRight:
		if g.Selected%cols < cols-1 && g.Selected < len(g.Cards)-1 {
			g.Selected++
			return true
		}
	case terminal.KeyUp:
		if g.Selected-cols >= 0 {
			g.Selected -= cols
			return true
		}
	case terminal.KeyDown:
		if g.Selected+cols < len(g.Cards) {
			g.Selected += cols
			return true
		}
	case terminal.KeyHome:
		g.Selected = 0
		return true
	case terminal.KeyEnd:
		g.Selected = len(g.Cards) - 1
		return true
	}

	return false
}

func (g *ProjectCardGrid) Draw(f *terminal.Frame, bounds terminal.Rect) {
	if bounds.Max.Row <= bounds.Min.Row || bounds.Max.Column <= bounds.Min.Column {
		return
	}

	headerStyle := terminal.Style{Foreground: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}
	f.DrawText(bounds.Min.Row, bounds.Min.Column+2, headerStyle, fmt.Sprintf("PROJECT DIRECTORIES (%d registered)", len(g.Cards)))

	if len(g.Cards) == 0 {
		emptyBox := terminal.NewRect(bounds.Min.Row+2, bounds.Min.Column+2, 6, bounds.Max.Column-bounds.Min.Column-4)
		f.DrawBox(emptyBox, terminal.Style{Foreground: terminal.ColorBrightBlack}, "No Projects")
		f.DrawText(emptyBox.Min.Row+2, emptyBox.Min.Column+4, terminal.Style{Foreground: terminal.ColorBrightWhite}, "No projects registered in MayFly yet.")
		f.DrawText(emptyBox.Min.Row+3, emptyBox.Min.Column+4, terminal.Style{Foreground: terminal.ColorBrightBlack}, "Navigate to a project folder and run: 'mayfly init' or press [N] to add.")
		return
	}

	totalWidth := bounds.Max.Column - bounds.Min.Column - 4
	cardWidth := (totalWidth - 2) / 2
	if cardWidth < 28 {
		cardWidth = totalWidth
	}
	cardHeight := 5

	availableHeight := bounds.Max.Row - bounds.Min.Row - 3
	visibleRows := availableHeight / (cardHeight + 1)
	if visibleRows < 1 {
		visibleRows = 1
	}

	selectedRow := g.Selected / 2
	scrollRow := 0
	if selectedRow >= visibleRows {
		scrollRow = selectedRow - visibleRows + 1
	}

	startRow := bounds.Min.Row + 2

	for i, card := range g.Cards {
		rowIdx := (i / 2) - scrollRow
		if rowIdx < 0 || rowIdx >= visibleRows {
			continue
		}

		colIdx := i % 2
		cardRow := startRow + rowIdx*(cardHeight+1)
		cardCol := bounds.Min.Column + 2 + colIdx*(cardWidth+2)

		if cardRow+cardHeight > bounds.Max.Row-1 {
			break
		}

		cardRect := terminal.NewRect(cardRow, cardCol, cardHeight, cardWidth)
		isSelected := i == g.Selected

		boxStyle := terminal.Style{Foreground: terminal.ColorBrightBlack}
		if isSelected && g.Focused {
			boxStyle = terminal.Style{Foreground: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}
		}

		projName := filepath.Base(card.Project.CanonicalPath)
		if projName == "/" || projName == "." {
			projName = card.Project.CanonicalPath
		}

		badge := fmt.Sprintf("[%d secrets]", card.SecretCount)
		title := fmt.Sprintf("%s %s", projName, badge)

		f.DrawBox(cardRect, boxStyle, title)

		// Inside card details
		pathStr := card.Project.CanonicalPath
		if len(pathStr) > cardWidth-6 {
			pathStr = "..." + pathStr[len(pathStr)-(cardWidth-9):]
		}

		pathStyle := terminal.Style{Foreground: terminal.ColorBrightWhite}
		if isSelected {
			pathStyle.Foreground = terminal.ColorBrightCyan
		}
		f.DrawText(cardRect.Min.Row+1, cardRect.Min.Column+2, pathStyle, "Path: "+pathStr)

		idStr := card.Project.ID
		if len(idStr) > 22 {
			idStr = idStr[:22] + "..."
		}
		f.DrawText(cardRect.Min.Row+2, cardRect.Min.Column+2, terminal.Style{Foreground: terminal.ColorBrightBlack}, "ID:   "+idStr)

		if card.IsCurrent {
			f.DrawText(cardRect.Min.Row+3, cardRect.Min.Column+2, terminal.Style{Foreground: terminal.ColorBrightGreen, Attributes: terminal.AttrBold}, "[Current Working Directory]")
		} else {
			f.DrawText(cardRect.Min.Row+3, cardRect.Min.Column+2, terminal.Style{Foreground: terminal.ColorBrightBlack}, "Press [Enter] to open secrets")
		}
	}
}
