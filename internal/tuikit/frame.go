package tuikit

import "fmt"

// Rect is a resolved area of the terminal grid.
type Rect struct{ X, Y, W, H int }

// Inner shrinks r by the one-cell border ring; a too-small rect collapses.
func (r Rect) Inner() Rect {
	if r.W <= 2 || r.H <= 2 {
		return Rect{X: r.X, Y: r.Y}
	}
	return Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
}

// Span claims size along one axis: Fixed takes exactly n cells, Fill shares
// the flexible remainder with the other fills on that axis.
type Span interface{ span() }

type fixedSpan int

func (fixedSpan) span() {}

type fillSpan struct{}

func (fillSpan) span() {}

// Fixed reserves exactly n cells on its axis.
func Fixed(n int) Span { return fixedSpan(n) }

// Fill shares the flexible remainder of its axis with the other fills.
func Fill() Span { return fillSpan{} }

// Frame is a declarative stack of rows. A row without cells is chrome space
// the host fills; a row split with Cols hosts one region per cell.
type Frame struct {
	minW, minH int
	w, h       int
	rows       []*Row
	dirty      bool
}

// NewFrame declares the smallest terminal the frame accepts.
func NewFrame(minW, minH int) *Frame {
	return &Frame{minW: minW, minH: minH, dirty: true}
}

// Resize records the terminal size (from tea.WindowSizeMsg).
func (f *Frame) Resize(w, h int) {
	f.w, f.h = w, h
	f.dirty = true
}

// Fits reports whether the terminal meets the declared minimum.
func (f *Frame) Fits() bool { return f.w >= f.minW && f.h >= f.minH }

// TooSmall is the refusal notice for undersized terminals.
func (f *Frame) TooSmall() string {
	return fmt.Sprintf("terminal too small: need ≥ %d×%d, have %d×%d — resize me",
		f.minW, f.minH, f.w, f.h)
}

// Row appends a row with the given vertical span.
func (f *Frame) Row(span Span) *Row {
	r := &Row{frame: f, span: span}
	f.rows = append(f.rows, r)
	f.dirty = true
	return r
}

// Row is one horizontal band. Rows are resolved top to bottom; Fill rows
// share the height the fixed rows leave over.
type Row struct {
	frame *Frame
	span  Span
	y, h  int
	cells []*Cell
}

// Cols splits the row horizontally into cells. From here on the row is a
// region row: each cell renders one region, in order.
func (r *Row) Cols(spans ...Span) []*Cell {
	for _, s := range spans {
		r.cells = append(r.cells, &Cell{row: r, span: s})
	}
	r.frame.dirty = true
	return r.cells
}

// Rect is the row's resolved area of the terminal.
func (r *Row) Rect() Rect {
	r.frame.resolve()
	return Rect{X: 0, Y: r.y, W: r.frame.w, H: r.h}
}

// Cell is one region slot inside a row.
type Cell struct {
	row  *Row
	span Span
	rect Rect
}

// Rect is the cell's resolved area, border ring included.
func (c *Cell) Rect() Rect {
	c.row.frame.resolve()
	return c.rect
}

// resolve lays out rows and cells for the current terminal size. Layout is
// recomputed only after a structural change or a resize.
func (f *Frame) resolve() {
	if !f.dirty {
		return
	}
	f.dirty = false
	spans := make([]Span, len(f.rows))
	for i, r := range f.rows {
		spans[i] = r.span
	}
	heights := splitSpans(f.h, spans)
	y := 0
	for i, r := range f.rows {
		r.y, r.h = y, heights[i]
		y += r.h
		cellSpans := make([]Span, len(r.cells))
		for j, c := range r.cells {
			cellSpans[j] = c.span
		}
		widths := splitSpans(f.w, cellSpans)
		x := 0
		for j, c := range r.cells {
			c.rect = Rect{X: x, Y: r.y, W: widths[j], H: r.h}
			x += widths[j]
		}
	}
}

// splitSpans resolves one axis: fixed claims pass through, fills share what
// is left (earlier fills absorb the rounding remainder).
func splitSpans(total int, spans []Span) []int {
	fixed, fills := 0, 0
	for _, s := range spans {
		switch v := s.(type) {
		case fixedSpan:
			fixed += int(v)
		case fillSpan:
			fills++
		}
	}
	out := make([]int, len(spans))
	rem := max(total-fixed, 0)
	share, extra := 0, 0
	if fills > 0 {
		share, extra = rem/fills, rem%fills
	}
	for i, s := range spans {
		if _, ok := s.(fillSpan); ok {
			out[i] = share
			if extra > 0 {
				out[i]++
				extra--
			}
			continue
		}
		out[i] = int(s.(fixedSpan))
	}
	return out
}

// audit checks the frame against its declared minimum (the wiring audit's
// geometry half: unsolvable layouts are dev errors, caught before shipping).
func (f *Frame) audit(where string, out *[]Issue) {
	fixedH, fillRows := 0, 0
	for _, r := range f.rows {
		if s, ok := r.span.(fixedSpan); ok {
			fixedH += int(s)
		} else {
			fillRows++
		}
	}
	if fixedH > f.minH {
		*out = append(*out, Issue{SeverityError, where,
			fmt.Sprintf("fixed rows need %d lines, minimum height is %d", fixedH, f.minH)})
	}
	if fillRows == 0 {
		*out = append(*out, Issue{SeverityError, where,
			"no Fill row — the frame cannot grow into extra height"})
	}
	for i, r := range f.rows {
		if len(r.cells) == 0 {
			continue
		}
		fixedW, fillCells := 0, 0
		for _, c := range r.cells {
			if s, ok := c.span.(fixedSpan); ok {
				fixedW += int(s)
			} else {
				fillCells++
			}
		}
		if fixedW > f.minW {
			*out = append(*out, Issue{SeverityError, where,
				fmt.Sprintf("row %d: fixed columns need %d cols, minimum width is %d", i, fixedW, f.minW)})
		}
		if fillCells == 0 {
			*out = append(*out, Issue{SeverityError, where,
				fmt.Sprintf("row %d: no Fill column — the row cannot grow into extra width", i)})
		}
	}
}
