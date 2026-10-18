package tuikit

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Line is one rendered row: plain text plus an optional style. Padding happens
// at Render time so styled backgrounds span the full region width.
type Line struct {
	Text  string
	Style *lipgloss.Style
}

// Plain is a Line without style.
func Plain(s string) Line { return Line{Text: s} }

// Styled is a Line rendered with st.
func Styled(s string, st lipgloss.Style) Line { return Line{Text: s, Style: &st} }

// Lines wraps plain strings.
func Lines(texts ...string) []Line {
	out := make([]Line, len(texts))
	for i, s := range texts {
		out[i] = Plain(s)
	}
	return out
}

// Render pads the line to exactly width columns, then applies the style.
func (l Line) Render(width int) string {
	s := padRight(l.Text, width)
	if l.Style == nil {
		return s
	}
	return l.Style.Render(s)
}

// Truncate caps s to max characters with an ellipsis.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max > 1 {
		return s[:max-1] + "…"
	}
	return s[:max]
}

// Clamp pins v to [lo, hi]; an inverted range collapses to lo.
func Clamp(v, lo, hi int) int {
	if hi < lo || v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}
