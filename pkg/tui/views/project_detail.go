package views

import (
	"fmt"
	"path/filepath"

	"mayfly/pkg/tui/terminal"
	"mayfly/pkg/tui/widget"
)

func (s *Screens) reloadSecrets() {
	if s.selProject.ID == "" {
		return
	}

	list, err := s.svc.ListSecrets(s.selProject.ID)
	if err != nil {
		s.SetStatus(fmt.Sprintf("Error: %v", err))
		return
	}
	s.secrets = list

	var items []widget.ListItem
	for _, sec := range list {
		maskedVal := "••••••••••••••••"
		if s.revealValue {
			maskedVal = sec.Value
		}
		items = append(items, widget.ListItem{
			Primary:   string(sec.Name),
			Secondary: maskedVal,
			Extra:     "[C: Copy]",
			Data:      sec,
		})
	}

	projName := filepath.Base(s.selProject.CanonicalPath)
	s.secretsList.Title = fmt.Sprintf("Secrets for: %s", projName)
	s.secretsList.SetItems(items)
}

func (s *Screens) drawProjectDetail(frame *terminal.Frame, bodyRect terminal.Rect) {
	s.secretsList.Draw(frame, bodyRect)
}
