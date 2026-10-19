// Package testrender turns the kit's rendered frames into inspectable cell
// grids and golden files — the capture → decode → compare pipeline for visual
// testing. No terminal, no screenshots; runs inside `go test`.
//
// Need-to-develop fence: no image or OSC capture, no pixel or percentage
// diffs, no waiting/retry primitives, no terminal matrices, no kitty/graphics
// support (Attr is the slot it would grow into), no ANSI generation — we only
// decode the vocabulary the kit itself emits.
//
// The three fidelity levels:
//
//	1 — chars: StripANSI over the frame; a pure UTF-8 grid.
//	2 — attrs: Decode walks the SGR spans and emits the old text-mode pair —
//	    one attribute letter + one character per cell.
//	3 — kitty: a placeholder; Cell.Attr is the slot it would use.
//
// Known limitation, stated on purpose: cells are runes — east-asian wide
// glyphs would break column alignment. We render none, so we solve none.
package testrender

import "strings"

// Attr is one palette slot of the decoded grid — the "attribute byte" of old
// text mode, as a letter so golden files stay readable text.
type Attr rune

const (
	// Normal is the unstyled slot every palette must register for the zero
	// style key.
	Normal Attr = 'N'

	// Unknown marks a style the palette does not cover; Decode reports it as
	// an error as well, so it never passes silently.
	Unknown Attr = '?'
)

// Cell is one decoded cell: the character and its palette slot.
type Cell struct {
	R rune
	A Attr
}

// Grid is a decoded frame: H rows of W cells, row-major.
type Grid struct {
	W, H  int
	Cells []Cell
}

// Cell returns the cell at x, y.
func (g Grid) Cell(x, y int) Cell { return g.Cells[y*g.W+x] }

// Chars renders the level-1 view: one string per row, attributes dropped.
func (g Grid) Chars() []string {
	out := make([]string, g.H)
	for y := 0; y < g.H; y++ {
		runes := make([]rune, g.W)
		for x := 0; x < g.W; x++ {
			runes[x] = g.Cell(x, y).R
		}
		out[y] = string(runes)
	}
	return out
}

// String renders the level-2 view: an attr line paired over each char line.
func (g Grid) String() string {
	var b strings.Builder
	for y := 0; y < g.H; y++ {
		attrs := make([]rune, g.W)
		chars := make([]rune, g.W)
		for x := 0; x < g.W; x++ {
			c := g.Cell(x, y)
			attrs[x] = rune(c.A)
			chars[x] = c.R
		}
		b.WriteString("attr: " + string(attrs) + "\n")
		b.WriteString("char: " + string(chars) + "\n")
	}
	return b.String()
}
