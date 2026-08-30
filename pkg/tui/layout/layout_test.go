package layout

import (
	"testing"

	"mayfly/pkg/tui/terminal"
)

func TestLayoutSplitVertical(t *testing.T) {
	target := terminal.NewRect(0, 0, 20, 80)
	rects := Split(DirVertical, target, []Constraint{
		Fixed(3),
		Flexible(),
		Fixed(1),
	})

	if len(rects) != 3 {
		t.Fatalf("expected 3 rects, got %d", len(rects))
	}

	if rects[0].Size().Rows != 3 {
		t.Fatalf("expected rect[0] height 3, got %d", rects[0].Size().Rows)
	}

	if rects[2].Size().Rows != 1 {
		t.Fatalf("expected rect[2] height 1, got %d", rects[2].Size().Rows)
	}

	if rects[1].Size().Rows != 16 {
		t.Fatalf("expected rect[1] height 16 (20-3-1), got %d", rects[1].Size().Rows)
	}
}
