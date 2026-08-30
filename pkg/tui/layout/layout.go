package layout

import (
	"mayfly/pkg/tui/terminal"
)

type Direction int

const (
	DirVertical Direction = iota
	DirHorizontal
)

type ConstraintType int

const (
	ConstraintFixed ConstraintType = iota
	ConstraintPercentage
	ConstraintFlexible
)

type Constraint struct {
	Type  ConstraintType
	Value int
}

func Fixed(v int) Constraint {
	return Constraint{Type: ConstraintFixed, Value: v}
}

func Percentage(pct int) Constraint {
	return Constraint{Type: ConstraintPercentage, Value: pct}
}

func Flexible() Constraint {
	return Constraint{Type: ConstraintFlexible, Value: 1}
}

// Split divides a target bounding box across child constraints.
func Split(dir Direction, target terminal.Rect, constraints []Constraint) []terminal.Rect {
	if len(constraints) == 0 {
		return []terminal.Rect{target}
	}

	size := target.Size()
	var totalSpace int
	if dir == DirVertical {
		totalSpace = size.Rows
	} else {
		totalSpace = size.Columns
	}

	allocated := make([]int, len(constraints))
	remaining := totalSpace
	flexCount := 0

	// 1. Fixed and Percentage allocations
	for i, c := range constraints {
		switch c.Type {
		case ConstraintFixed:
			val := c.Value
			if val > remaining {
				val = remaining
			}
			allocated[i] = val
			remaining -= val
		case ConstraintPercentage:
			val := (totalSpace * c.Value) / 100
			if val > remaining {
				val = remaining
			}
			allocated[i] = val
			remaining -= val
		case ConstraintFlexible:
			flexCount++
		}
	}

	// 2. Distribute remainder among Flexible items
	if flexCount > 0 && remaining > 0 {
		flexShare := remaining / flexCount
		extra := remaining % flexCount
		for i, c := range constraints {
			if c.Type == ConstraintFlexible {
				share := flexShare
				if extra > 0 {
					share++
					extra--
				}
				allocated[i] = share
			}
		}
	}

	// 3. Construct Rectangles
	rects := make([]terminal.Rect, len(constraints))
	currRow := target.Min.Row
	currCol := target.Min.Column

	for i, alloc := range allocated {
		if dir == DirVertical {
			rects[i] = terminal.NewRect(currRow, currCol, alloc, size.Columns)
			currRow += alloc
		} else {
			rects[i] = terminal.NewRect(currRow, currCol, size.Rows, alloc)
			currCol += alloc
		}
	}

	return rects
}
