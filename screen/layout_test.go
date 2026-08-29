package screen

import "testing"

func TestVerticalHeaderBodyFooterLayout(t *testing.T) {
	header := NewRegion("header", Size{Rows: 1, Columns: 1}, Size{Rows: 1, Columns: 20})
	body := NewRegion("body", Size{Rows: 1, Columns: 1}, Size{Rows: 3, Columns: 20})
	footer := NewRegion("footer", Size{Rows: 1, Columns: 1}, Size{Rows: 1, Columns: 20})

	placements := Arrange(Vertical(header, Flexible(body), footer), NewRect(0, 0, 8, 20))
	assertRegionBounds(t, placements, header, NewRect(0, 0, 1, 20))
	assertRegionBounds(t, placements, body, NewRect(1, 0, 6, 20))
	assertRegionBounds(t, placements, footer, NewRect(7, 0, 1, 20))
}

func TestHorizontalSidebarAndMainLayout(t *testing.T) {
	sidebar := NewRegion("sidebar", Size{Rows: 1, Columns: 4}, Size{Rows: 10, Columns: 4})
	main := NewRegion("main", Size{Rows: 1, Columns: 1}, Size{Rows: 10, Columns: 3})

	placements := Arrange(Horizontal(sidebar, Flexible(main)), NewRect(0, 0, 5, 20))
	assertRegionBounds(t, placements, sidebar, NewRect(0, 0, 5, 4))
	assertRegionBounds(t, placements, main, NewRect(0, 4, 5, 16))
}

func TestNestedPaddingAndCenteredLayout(t *testing.T) {
	content := NewRegion("content", Size{Rows: 2, Columns: 4}, Size{Rows: 2, Columns: 4})
	root := Centered(Padded(Padding{Top: 1, Right: 2, Bottom: 1, Left: 2}, content))

	placements := Arrange(root, NewRect(0, 0, 8, 20))
	assertRegionBounds(t, placements, content, NewRect(3, 8, 2, 4))
}

func TestFixedAndFlexibleChildren(t *testing.T) {
	fixed := NewRegion("fixed", Size{}, Size{Rows: 2, Columns: 3})
	body := NewRegion("body", Size{Rows: 1, Columns: 1}, Size{Rows: 1, Columns: 4})

	placements := Arrange(Vertical(Fixed(Size{Rows: 2, Columns: 3}, fixed), Flexible(body)), NewRect(0, 0, 6, 10))
	assertRegionBounds(t, placements, fixed, NewRect(0, 0, 2, 3))
	assertRegionBounds(t, placements, body, NewRect(2, 0, 4, 10))
}

func TestOverlayPreservesOrderAndBounds(t *testing.T) {
	back := NewRegion("back", Size{}, Size{Rows: 1, Columns: 1})
	front := NewRegion("front", Size{}, Size{Rows: 1, Columns: 1})

	placements := Arrange(Overlay(back, Centered(front)), NewRect(2, 3, 5, 7))
	if len(placements) != 2 {
		t.Fatalf("placement count = %d, want 2", len(placements))
	}
	assertRegionBounds(t, placements, back, NewRect(2, 3, 5, 7))
	assertRegionBounds(t, placements, front, NewRect(4, 6, 1, 1))
	if placements[0].Node != back || placements[1].Node != front {
		t.Fatalf("overlay order = %#v, want back then front", placements)
	}
}

func TestInsufficientTerminalSizeClipsMinimumsDeterministically(t *testing.T) {
	first := NewRegion("first", Size{Rows: 2, Columns: 1}, Size{Rows: 2, Columns: 1})
	second := NewRegion("second", Size{Rows: 3, Columns: 1}, Size{Rows: 3, Columns: 1})
	third := NewRegion("third", Size{Rows: 1, Columns: 1}, Size{Rows: 1, Columns: 1})

	placements := Arrange(Vertical(first, second, third), NewRect(0, 0, 3, 4))
	assertRegionBounds(t, placements, first, NewRect(0, 0, 2, 4))
	assertRegionBounds(t, placements, second, NewRect(2, 0, 1, 4))
	assertRegionBounds(t, placements, third, NewRect(3, 0, 0, 4))
	for _, placement := range placements {
		if placement.Bounds.Size().Rows < 0 || placement.Bounds.Size().Columns < 0 {
			t.Fatalf("negative placement bounds: %#v", placement.Bounds)
		}
	}
}

func TestZeroSizeAndOddDimensionLayouts(t *testing.T) {
	zero := NewRegion("zero", Size{}, Size{})
	if placements := Arrange(zero, NewRect(0, 0, 0, 0)); len(placements) != 1 || !placements[0].Bounds.Empty() {
		t.Fatalf("zero-size placement = %#v", placements)
	}
	if placements := Arrange(nil, NewRect(0, 0, 4, 4)); placements != nil {
		t.Fatalf("nil root placements = %#v, want nil", placements)
	}

	left := NewRegion("left", Size{Rows: 1, Columns: 1}, Size{Rows: 1, Columns: 1})
	right := NewRegion("right", Size{Rows: 1, Columns: 1}, Size{Rows: 1, Columns: 1})
	placements := Arrange(Horizontal(Flexible(left), Flexible(right)), NewRect(0, 0, 1, 7))
	assertRegionBounds(t, placements, left, NewRect(0, 0, 1, 4))
	assertRegionBounds(t, placements, right, NewRect(0, 4, 1, 3))
}

func TestLayoutRearrangesForResize(t *testing.T) {
	header := NewRegion("header", Size{Rows: 1, Columns: 1}, Size{Rows: 1, Columns: 1})
	body := NewRegion("body", Size{Rows: 1, Columns: 1}, Size{Rows: 1, Columns: 1})
	root := Vertical(header, Flexible(body))

	first := Arrange(root, NewRect(0, 0, 4, 10))
	second := Arrange(root, NewRect(0, 0, 7, 13))
	assertRegionBounds(t, first, header, NewRect(0, 0, 1, 10))
	assertRegionBounds(t, first, body, NewRect(1, 0, 3, 10))
	assertRegionBounds(t, second, header, NewRect(0, 0, 1, 13))
	assertRegionBounds(t, second, body, NewRect(1, 0, 6, 13))
}

func TestLayoutMinimumAndPreferredSizes(t *testing.T) {
	header := NewRegion("header", Size{Rows: 1, Columns: 2}, Size{Rows: 2, Columns: 4})
	footer := NewRegion("footer", Size{Rows: 1, Columns: 3}, Size{Rows: 1, Columns: 5})
	root := Vertical(header, footer)

	if got, want := root.MinSize(), (Size{Rows: 2, Columns: 3}); got != want {
		t.Fatalf("minimum size = %#v, want %#v", got, want)
	}
	if got, want := root.PreferredSize(), (Size{Rows: 3, Columns: 5}); got != want {
		t.Fatalf("preferred size = %#v, want %#v", got, want)
	}
}

func assertRegionBounds(t *testing.T, placements []Placement, region *Region, want Rect) {
	t.Helper()
	for _, placement := range placements {
		if placement.Node == region {
			if placement.Bounds != want {
				t.Fatalf("region %q bounds = %#v, want %#v", region.Name, placement.Bounds, want)
			}
			return
		}
	}
	t.Fatalf("region %q was not placed", region.Name)
}
