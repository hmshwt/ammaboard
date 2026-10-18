package tuikit

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Region is one boxed pane: title, fixed header, scrollable body. A Region is
// a value built fresh each frame; Render pads or clips it to EXACTLY h rows,
// so equal pane heights are structural, not discipline.
type Region struct {
	Title   string
	Header  []Line
	Body    []Line
	Focused bool
}

// Styles carries the palette the frame furniture needs. The kit ships
// DefaultStyles; the app overrides fields to re-skin — no theming engine.
type Styles struct {
	Border        lipgloss.Style
	BorderFocused lipgloss.Style
	Title         lipgloss.Style
	Dim           lipgloss.Style
}

// DefaultStyles is the kit's own palette.
var DefaultStyles = Styles{
	Border:        lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6a737d")),
	BorderFocused: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#4FA6FF")),
	Title:         lipgloss.NewStyle().Foreground(lipgloss.Color("#4FA6FF")).Bold(true),
	Dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("#6a737d")),
}

// Render boxes the region to exactly w×h columns/rows, borders included:
// title, then header, then body — padded or clipped to the inner height.
func (r Region) Render(w, h int, st Styles) string {
	if w < 2 || h < 2 {
		return ""
	}
	inner := h - 2
	rows := make([]Line, 0, inner)
	rows = append(rows, Line{Text: r.Title, Style: &st.Title})
	rows = append(rows, r.Header...)
	rows = append(rows, r.Body...)
	for len(rows) < inner {
		rows = append(rows, Line{})
	}
	rows = rows[:inner]
	out := make([]string, len(rows))
	for i, l := range rows {
		out[i] = l.Render(w - 2)
	}
	border := st.Border
	if r.Focused {
		border = st.BorderFocused
	}
	return border.Render(strings.Join(out, "\n"))
}
