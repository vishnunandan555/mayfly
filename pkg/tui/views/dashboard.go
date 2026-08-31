package views

import (
	"fmt"

	"mayfly/pkg/tui/terminal"
	"mayfly/pkg/tui/widget"
)

func (s *Screens) reloadProjects() {
	projects, err := s.svc.Projects()
	if err != nil {
		s.SetStatus(fmt.Sprintf("Failed to load projects: %v", err))
		return
	}

	var currentProjID string
	if cur, err := s.svc.ResolveCurrentProject(s.currentDir); err == nil {
		currentProjID = cur.ID
	}

	var cards []widget.ProjectCard
	for _, p := range projects {
		secCount := 0
		if s.svc.IsUnlocked() {
			if list, err := s.svc.ListSecrets(p.ID); err == nil {
				secCount = len(list)
			}
		}
		cards = append(cards, widget.ProjectCard{
			Project:     p,
			SecretCount: secCount,
			IsCurrent:   p.ID == currentProjID,
		})
	}

	s.projectGrid.SetCards(cards)
}

func (s *Screens) drawDashboard(frame *terminal.Frame, bodyRect terminal.Rect) {
	s.projectGrid.Draw(frame, bodyRect)
}
