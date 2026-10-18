package tuikit

import (
	"strings"
	"testing"
)

func TestLineRenderPads(t *testing.T) {
	if got := Plain("ab").Render(5); got != "ab   " {
		t.Errorf("Plain render = %q, want %q", got, "ab   ")
	}
	if got := Plain("abcde").Render(3); got != "abcde" {
		t.Errorf("overlong lines pass through, got %q", got)
	}
	if got := Lines("a", "b"); len(got) != 2 || got[1].Text != "b" {
		t.Errorf("Lines = %v", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("abcdef", 3); got != "ab…" {
		t.Errorf("Truncate = %q, want %q", got, "ab…")
	}
	if got := Truncate("ab", 5); got != "ab" {
		t.Errorf("short strings pass through, got %q", got)
	}
	if got := Truncate("abcdef", 1); got != "a" {
		t.Errorf("degenerate max = %q, want %q", got, "a")
	}
}

func TestClamp(t *testing.T) {
	cases := []struct{ v, lo, hi, want int }{
		{5, 0, 3, 3},
		{-1, 0, 3, 0},
		{2, 0, 3, 2},
		{2, 3, 1, 3}, // an inverted range collapses to lo
	}
	for _, c := range cases {
		if got := Clamp(c.v, c.lo, c.hi); got != c.want {
			t.Errorf("Clamp(%d,%d,%d) = %d, want %d", c.v, c.lo, c.hi, got, c.want)
		}
	}
}

func TestCenterPads(t *testing.T) {
	out := Center("hi", 10, 3, 1)
	rows := strings.Split(out, "\n")
	if len(rows) != 3 {
		t.Fatalf("centered %d rows, want 3", len(rows))
	}
	if !strings.Contains(out, "hi") {
		t.Errorf("centered body lost its content: %q", out)
	}
}
