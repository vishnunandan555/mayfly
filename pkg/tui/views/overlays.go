package views

import (
	"context"
	"fmt"

	"mayfly/pkg/tui/terminal"
	"mayfly/pkg/tui/widget"
)

func (s *Screens) runScan() {
	ctx := context.Background()
	dir := s.currentDir
	if s.selProject.CanonicalPath != "" {
		dir = s.selProject.CanonicalPath
	}

	findings, err := s.svc.Scan(ctx, dir)
	if err != nil {
		s.SetStatus(fmt.Sprintf("Scan error: %v", err))
		return
	}

	var items []widget.ListItem
	for _, f := range findings {
		items = append(items, widget.ListItem{
			Primary:   fmt.Sprintf("[%s] %s:%d", f.Severity, f.Path, f.Line),
			Secondary: f.Message,
		})
	}

	s.scanList.SetItems(items)
	if len(findings) == 0 {
		s.SetStatus("[OK] No plaintext credential leaks detected.")
	}
}

func (s *Screens) loadAudit() {
	ctx := context.Background()
	events, err := s.svc.AuditTrail(ctx)
	if err != nil {
		s.SetStatus(fmt.Sprintf("Audit error: %v", err))
		return
	}

	var items []widget.ListItem
	for i := len(events) - 1; i >= 0; i-- { // Most recent first
		ev := events[i]
		items = append(items, widget.ListItem{
			Primary:   fmt.Sprintf("#%d %s", ev.Sequence, ev.Action),
			Secondary: ev.At.Format("15:04:05") + " " + ev.Secret + " " + ev.Command,
			Extra:     ev.Hash[:8] + "...",
		})
	}
	s.auditList.SetItems(items)
}

func (s *Screens) drawScan(frame *terminal.Frame, bodyRect terminal.Rect) {
	s.scanList.Draw(frame, bodyRect)
}

func (s *Screens) drawAudit(frame *terminal.Frame, bodyRect terminal.Rect) {
	s.auditList.Draw(frame, bodyRect)
}
