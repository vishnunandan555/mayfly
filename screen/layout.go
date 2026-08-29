package screen

// Layout is a stateless layout node. MinSize is the smallest useful size for
// the node; PreferredSize is the size it would like when space is available.
// Arrange calls layout.Arrange with a new terminal rectangle, so terminal
// resize handling requires no mutable layout state.
type Layout interface {
	MinSize() Size
	PreferredSize() Size
}

// Placement is the clipped rectangle assigned to one leaf layout node.
// Placements are returned in draw order; for overlays, later placements are
// on top of earlier placements.
type Placement struct {
	Node   Layout
	Bounds Rect
}

// Region is a named leaf layout. Name is application metadata and is not
// interpreted by the layout engine.
type Region struct {
	Name      string
	Minimum   Size
	Preferred Size
}

// NewRegion creates a named leaf layout. Preferred dimensions below minimum
// are raised to the corresponding minimum dimension.
func NewRegion(name string, minimum, preferred Size) *Region {
	minimum = minimum.normalized()
	preferred = preferred.normalized()
	preferred.Rows = max(preferred.Rows, minimum.Rows)
	preferred.Columns = max(preferred.Columns, minimum.Columns)
	return &Region{Name: name, Minimum: minimum, Preferred: preferred}
}

// MinSize returns the region's minimum size.
func (r *Region) MinSize() Size {
	if r == nil {
		return Size{}
	}
	return r.Minimum.normalized()
}

// PreferredSize returns the region's preferred size.
func (r *Region) PreferredSize() Size {
	if r == nil {
		return Size{}
	}
	preferred := r.Preferred.normalized()
	minimum := r.MinSize()
	preferred.Rows = max(preferred.Rows, minimum.Rows)
	preferred.Columns = max(preferred.Columns, minimum.Columns)
	return preferred
}

// Fixed returns a node with an exact requested size. Its child is arranged at
// the fixed region's top-left and clipped if its parent is smaller.
func Fixed(size Size, child Layout) Layout {
	return &fixedLayout{size: size.normalized(), child: child}
}

type fixedLayout struct {
	size  Size
	child Layout
}

func (l *fixedLayout) MinSize() Size       { return l.size }
func (l *fixedLayout) PreferredSize() Size { return l.size }

func (l *fixedLayout) arrange(bounds Rect, placements *[]Placement) {
	area := clipRect(NewRect(bounds.Min.Row, bounds.Min.Column, l.size.Rows, l.size.Columns), bounds)
	arrangeNode(l.child, area, placements)
}

// Flexible marks a child as eligible to consume remaining stack space after
// minimum and preferred sizes have been satisfied.
func Flexible(child Layout) Layout {
	return &flexibleLayout{child: child}
}

type flexibleLayout struct {
	child Layout
}

func (l *flexibleLayout) MinSize() Size {
	return minSizeOf(l.child)
}

func (l *flexibleLayout) PreferredSize() Size {
	return preferredSizeOf(l.child)
}

func (l *flexibleLayout) arrange(bounds Rect, placements *[]Placement) {
	arrangeNode(l.child, bounds, placements)
}

// Padded insets a child by padding. Padding is applied inside the assigned
// rectangle and is clipped naturally when the available area is tiny.
func Padded(padding Padding, child Layout) Layout {
	return &paddedLayout{padding: padding, child: child}
}

type paddedLayout struct {
	padding Padding
	child   Layout
}

func (l *paddedLayout) MinSize() Size {
	return addPadding(minSizeOf(l.child), l.padding)
}

func (l *paddedLayout) PreferredSize() Size {
	return addPadding(preferredSizeOf(l.child), l.padding)
}

func (l *paddedLayout) arrange(bounds Rect, placements *[]Placement) {
	arrangeNode(l.child, clipRect(l.padding.Inset(bounds), bounds), placements)
}

// Centered centers a child using its preferred size, clipping it when the
// assigned rectangle is smaller than the child.
func Centered(child Layout) Layout {
	return &centeredLayout{child: child}
}

type centeredLayout struct {
	child Layout
}

func (l *centeredLayout) MinSize() Size       { return minSizeOf(l.child) }
func (l *centeredLayout) PreferredSize() Size { return preferredSizeOf(l.child) }

func (l *centeredLayout) arrange(bounds Rect, placements *[]Placement) {
	preferred := preferredSizeOf(l.child)
	available := bounds.Size()
	rows := min(preferred.Rows, available.Rows)
	columns := min(preferred.Columns, available.Columns)
	row := bounds.Min.Row + (available.Rows-rows)/2
	column := bounds.Min.Column + (available.Columns-columns)/2
	arrangeNode(l.child, clipRect(NewRect(row, column, rows, columns), bounds), placements)
}

// Vertical creates a top-to-bottom stack. Children first receive their
// minimum sizes, then grow toward preferred sizes; remaining space goes to
// Flexible children in deterministic round-robin order.
func Vertical(children ...Layout) Layout {
	return &stackLayout{vertical: true, children: append([]Layout(nil), children...)}
}

// Horizontal creates a left-to-right stack with the same sizing rules as
// Vertical.
func Horizontal(children ...Layout) Layout {
	return &stackLayout{children: append([]Layout(nil), children...)}
}

type stackLayout struct {
	vertical bool
	children []Layout
}

func (l *stackLayout) MinSize() Size {
	return l.measure(false)
}

func (l *stackLayout) PreferredSize() Size {
	return l.measure(true)
}

func (l *stackLayout) measure(preferred bool) Size {
	var result Size
	for _, child := range l.children {
		size := minSizeOf(child)
		if preferred {
			size = preferredSizeOf(child)
		}
		if l.vertical {
			result.Rows += size.Rows
			result.Columns = max(result.Columns, size.Columns)
		} else {
			result.Rows = max(result.Rows, size.Rows)
			result.Columns += size.Columns
		}
	}
	return result
}

func (l *stackLayout) arrange(bounds Rect, placements *[]Placement) {
	available := bounds.Size()
	mainAvailable := available.Columns
	if l.vertical {
		mainAvailable = available.Rows
	}
	sizes := allocateMainSizes(l.children, mainAvailable, l.vertical)
	position := bounds.Min.Row
	if !l.vertical {
		position = bounds.Min.Column
	}
	for index, child := range l.children {
		main := sizes[index]
		var area Rect
		if l.vertical {
			area = NewRect(position, bounds.Min.Column, main, available.Columns)
			position += main
		} else {
			area = NewRect(bounds.Min.Row, position, available.Rows, main)
			position += main
		}
		arrangeNode(child, clipRect(area, bounds), placements)
	}
}

// Overlay assigns the complete available rectangle to each child in order.
// Later children are intended to be rendered above earlier children.
func Overlay(children ...Layout) Layout {
	return &overlayLayout{children: append([]Layout(nil), children...)}
}

type overlayLayout struct {
	children []Layout
}

func (l *overlayLayout) MinSize() Size       { return l.measure(false) }
func (l *overlayLayout) PreferredSize() Size { return l.measure(true) }

func (l *overlayLayout) measure(preferred bool) Size {
	var result Size
	for _, child := range l.children {
		size := minSizeOf(child)
		if preferred {
			size = preferredSizeOf(child)
		}
		result.Rows = max(result.Rows, size.Rows)
		result.Columns = max(result.Columns, size.Columns)
	}
	return result
}

func (l *overlayLayout) arrange(bounds Rect, placements *[]Placement) {
	for _, child := range l.children {
		arrangeNode(child, bounds, placements)
	}
}

// Arrange computes clipped leaf placements for bounds. It is deterministic,
// stateless, and safe for zero-sized or smaller-than-minimum bounds. When
// space is insufficient for all minimum sizes, earlier stack children receive
// space first and later children may receive zero rows or columns.
func Arrange(root Layout, bounds Rect) []Placement {
	if root == nil {
		return nil
	}
	bounds = normalizeRect(bounds)
	placements := make([]Placement, 0)
	arrangeNode(root, bounds, &placements)
	return placements
}

type arranger interface {
	arrange(Rect, *[]Placement)
}

func arrangeNode(node Layout, bounds Rect, placements *[]Placement) {
	if node == nil {
		return
	}
	if layout, ok := node.(arranger); ok {
		layout.arrange(normalizeRect(bounds), placements)
		return
	}
	*placements = append(*placements, Placement{Node: node, Bounds: normalizeRect(bounds)})
}

func allocateMainSizes(children []Layout, available int, vertical bool) []int {
	sizes := make([]int, len(children))
	if available <= 0 {
		return sizes
	}

	minimums := make([]int, len(children))
	preferreds := make([]int, len(children))
	minimumTotal := 0
	for index, child := range children {
		minimums[index] = mainSize(minSizeOf(child), vertical)
		preferreds[index] = max(minimums[index], mainSize(preferredSizeOf(child), vertical))
		minimumTotal += minimums[index]
	}
	if minimumTotal > available {
		remaining := available
		for index, minimum := range minimums {
			sizes[index] = min(minimum, remaining)
			remaining -= sizes[index]
			if remaining == 0 {
				break
			}
		}
		return sizes
	}

	copy(sizes, minimums)
	extra := available - minimumTotal
	for index := range children {
		growth := min(extra, preferreds[index]-sizes[index])
		sizes[index] += growth
		extra -= growth
		if extra == 0 {
			return sizes
		}
	}

	flexible := make([]int, 0)
	for index, child := range children {
		if isFlexible(child) {
			flexible = append(flexible, index)
		}
	}
	for extra > 0 && len(flexible) > 0 {
		share := extra / len(flexible)
		if share == 0 {
			share = 1
		}
		for _, index := range flexible {
			if extra == 0 {
				break
			}
			amount := min(share, extra)
			sizes[index] += amount
			extra -= amount
		}
	}
	return sizes
}

func isFlexible(node Layout) bool {
	_, ok := node.(*flexibleLayout)
	return ok
}

func mainSize(size Size, vertical bool) int {
	if vertical {
		return size.Rows
	}
	return size.Columns
}

func minSizeOf(node Layout) Size {
	if node == nil {
		return Size{}
	}
	return node.MinSize().normalized()
}

func preferredSizeOf(node Layout) Size {
	if node == nil {
		return Size{}
	}
	preferred := node.PreferredSize().normalized()
	minimum := minSizeOf(node)
	preferred.Rows = max(preferred.Rows, minimum.Rows)
	preferred.Columns = max(preferred.Columns, minimum.Columns)
	return preferred
}

func addPadding(size Size, padding Padding) Size {
	return Size{
		Rows:    size.Rows + max(0, padding.Top) + max(0, padding.Bottom),
		Columns: size.Columns + max(0, padding.Left) + max(0, padding.Right),
	}.normalized()
}

func normalizeRect(rect Rect) Rect {
	if rect.Max.Row < rect.Min.Row {
		rect.Max.Row = rect.Min.Row
	}
	if rect.Max.Column < rect.Min.Column {
		rect.Max.Column = rect.Min.Column
	}
	return rect
}

func clipRect(rect, bounds Rect) Rect {
	return normalizeRect(normalizeRect(rect).Intersect(normalizeRect(bounds)))
}
