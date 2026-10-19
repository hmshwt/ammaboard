package tuikit

import (
	"testing"

	"ammaboard/internal/tuikit/testrender"
	"charm.land/lipgloss/v2"
)

// kitPalette mirrors DefaultStyles: the visual contract, as data.
func kitPalette() testrender.Palette {
	return testrender.Palette{
		{}:                                       'N',
		{Fg: [3]int{106, 115, 125}, HasFg: true}: 'D', // dim border
		{Fg: [3]int{79, 166, 255}, HasFg: true}:  'A', // focused border
		{Fg: [3]int{79, 166, 255}, HasFg: true, Bold: true}: 'T', // titles
	}
}

// The kit's own render, decoded end-to-end: this is the proof that golden
// testing will have real cells to look at.
func TestRegionRenderDecodesToGrid(t *testing.T) {
	frame := lipgloss.JoinHorizontal(lipgloss.Top,
		Region{Title: "left"}.Render(10, 5, DefaultStyles),
		Region{Title: "right", Focused: true}.Render(10, 5, DefaultStyles),
	)
	grid, err := testrender.Decode(frame, kitPalette())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if grid.W != 20 || grid.H != 5 {
		t.Fatalf("grid = %dx%d, want 20×5", grid.W, grid.H)
	}
	if got := grid.Cell(0, 0).A; got != 'D' {
		t.Errorf("unfocused border = %c, want D", got)
	}
	if got := grid.Cell(10, 0).A; got != 'A' {
		t.Errorf("focused border = %c, want A", got)
	}
	if got := grid.Cell(1, 1); got.A != 'T' || got.R != 'l' {
		t.Errorf("title cell = %+v, want T/l", got)
	}
	if got := grid.Cell(1, 3).A; got != 'N' {
		t.Errorf("empty body = %c, want N", got)
	}
}
