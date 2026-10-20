package tui

// help.go — the help modal. The grammar is generated from the keymaps (global
// / screen / focused pane) plus the shell's command surface — docs cannot
// drift from behavior, because there is no second copy of the truth.

import (
	"strings"

	"ammaboard/internal/tuikit"
)

type helpModal struct {
	m *Model
}

// View renders the modal into the w×h region area the kit hands it; the kit
// replaces only the region rows, so status bar and shell stay visible.
func (hm helpModal) View(w, h int) string {
	m := hm.m

	var lines []string
	add := func(section string, ls []tuikit.Line) {
		lines = append(lines, "", section)
		for _, l := range ls {
			lines = append(lines, l.Text)
		}
	}
	lines = append(lines, "AmmaBoard TUI")
	add("global", m.App.Keys.Help())
	add("board", m.App.Screens.Current().Keys.Help())
	focus := m.App.Screens.Current().FocusRing.Index()
	if 0 <= focus && focus < len(m.panes) {
		add("panes (tab cycles)", m.panes[focus].keys().Help())
	}
	lines = append(lines, "", "shell", "  run · step [n] · reset · speed n · quit · help")

	return tuikit.Center(stHelp.Render(strings.Join(lines, "\n")), w, h, 2)
}
