package tui

// shell.go — the bottom block: a boxed prompt with its recent replies and the
// generated keyhints line under it. The box border lights up when the shell
// is focused — the same accent the focused pane gets.

import (
	"strings"

	"ammaboard/internal/tuikit"

	"charm.land/lipgloss/v2"
)

// shellBlock renders the 6-row bottom block: box (5) + keyhints (1). Hints
// are generated from the global + current screen keymaps — help cannot lie.
func (m *Model) shellBlock(w int) string {
	st := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).
		BorderForeground(cDim).Width(w - 2)
	if m.input.Focused() {
		st = st.BorderForeground(cAccent)
	}
	rows := []string{m.input.View()}
	for _, o := range m.shellOut[max(0, len(m.shellOut)-2):] {
		rows = append(rows, stDim.Render("  "+o))
	}
	for len(rows) < 3 {
		rows = append(rows, "")
	}
	box := st.Render(strings.Join(rows[:3], "\n"))
	hints := tuikit.Merge(m.App.Keys, m.App.Screens.Current().Keys).Hints()
	return lipgloss.JoinVertical(lipgloss.Left, box, stBottom.Render(hints))
}
