package tuikit

import (
	"strings"
	"testing"
)

func TestFrameRowsSumToHeight(t *testing.T) {
	f := NewFrame(141, 28)
	f.Resize(145, 30)
	top := f.Row(Fixed(1))
	panes := f.Row(Fill())
	bottom := f.Row(Fixed(6))
	if got := top.Rect().H; got != 1 {
		t.Errorf("top row = %d, want 1", got)
	}
	if got := panes.Rect(); got.H != 23 || got.Y != 1 {
		t.Errorf("pane row = %+v, want y=1 h=23 (fills absorb the rest)", got)
	}
	if got := bottom.Rect(); got.H != 6 || got.Y != 24 {
		t.Errorf("bottom row = %+v, want y=24 h=6", got)
	}
}

func TestColsSplitFixedAndFill(t *testing.T) {
	f := NewFrame(141, 28)
	f.Resize(145, 30)
	cells := f.Row(Fill()).Cols(Fixed(44), Fill(), Fixed(54))
	if got := cells[0].Rect(); got.W != 44 || got.X != 0 {
		t.Errorf("left = %+v, want x=0 w=44", got)
	}
	if got := cells[1].Rect(); got.W != 47 || got.X != 44 {
		t.Errorf("center = %+v, want x=44 w=47", got)
	}
	if got := cells[2].Rect(); got.X != 91 || got.W != 54 {
		t.Errorf("right = %+v, want x=91 w=54", got)
	}
	if cells[0].Rect().H != 30 { // the lone Fill row takes the whole height
		t.Errorf("cells inherit the row height, got %d", cells[0].Rect().H)
	}
}

func TestFillsShareRemainder(t *testing.T) {
	f := NewFrame(1, 1)
	f.Resize(10, 11)
	a, b := f.Row(Fill()), f.Row(Fill())
	if a.Rect().H != 6 || b.Rect().H != 5 {
		t.Errorf("fills = %d/%d, want 6/5 (earlier fill absorbs the rounding)", a.Rect().H, b.Rect().H)
	}
}

func TestTooSmallAndFits(t *testing.T) {
	f := NewFrame(141, 28)
	f.Resize(80, 20)
	if f.Fits() {
		t.Fatal("80×20 must not fit a 141×28 minimum")
	}
	if s := f.TooSmall(); !strings.Contains(s, "141×28") || !strings.Contains(s, "80×20") {
		t.Errorf("notice = %q", s)
	}
	f.Resize(141, 28)
	if !f.Fits() {
		t.Fatal("the exact minimum must fit")
	}
}

func TestInnerShrinksByBorderRing(t *testing.T) {
	if got, want := (Rect{X: 2, Y: 3, W: 44, H: 23}).Inner(), (Rect{X: 3, Y: 4, W: 42, H: 21}); got != want {
		t.Errorf("inner = %+v, want %+v", got, want)
	}
	if got := (Rect{W: 2, H: 2}).Inner(); got.W != 0 || got.H != 0 {
		t.Errorf("degenerate inner = %+v, want zero size", got)
	}
}

func TestRegionRenderExactHeight(t *testing.T) {
	r := Region{Title: "program", Body: Lines("add_number r1, r0, 7", "halt")}
	out := r.Render(20, 8, DefaultStyles)
	if rows := strings.Split(out, "\n"); len(rows) != 8 {
		t.Fatalf("rendered %d rows, want exactly 8 (structural equal heights)", len(rows))
	}
	for _, want := range []string{"program", "add_number r1, r0, 7", "halt"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}

func TestRegionRenderClipsAndPads(t *testing.T) {
	var body []Line
	for i := 0; i < 50; i++ {
		body = append(body, Plain(strings.Repeat("x", 30)))
	}
	out := Region{Title: "p", Body: body}.Render(10, 4, DefaultStyles)
	if rows := strings.Split(out, "\n"); len(rows) != 4 {
		t.Fatalf("clipped to %d rows, want 4", len(rows))
	}
	out = Region{Title: "p"}.Render(10, 6, DefaultStyles)
	if rows := strings.Split(out, "\n"); len(rows) != 6 {
		t.Fatalf("padded to %d rows, want 6", len(rows))
	}
}
