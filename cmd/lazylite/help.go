package main

import (
	"strings"

	"ammaboard/internal/tuikit"

	"charm.land/lipgloss/v2"
)

// helpModal is a real MODAL: transient, view-only, keyboard captured by the
// kit (esc pops it; '?' — a global verb — toggles it closed again). Its
// content is GENERATED from the same keymaps that route keys, and it names
// where you are. Help cannot lie because it is a derivation, not a document.
type helpModal struct{ m *Model }

func (hm helpModal) View(w, h int) string {
	m := hm.m
	km := m.Keys
	if cur := m.Screens.Current(); cur != nil {
		km = tuikit.Merge(km, cur.Keys)
		if cur.RegionKeys != nil {
			km = tuikit.Merge(km, cur.RegionKeys(m.focusIdx()))
		}
	}

	var b strings.Builder
	b.WriteString("WHERE AM I\n")
	b.WriteString("  " + strings.TrimLeft(m.statusLine(), " ") + "\n\n")
	b.WriteString("screens switch · panes tile · cursors point · details stack · modals capture\n\n")
	b.WriteString("KEYS HERE — generated from the routing data, so it cannot drift\n")
	for _, l := range km.Help() {
		b.WriteString("  " + l.Text + "\n")
	}

	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("81")).
		Padding(0, 1).
		Render(strings.TrimRight(b.String(), "\n"))

	width := w - 6
	if width > 92 {
		width = 92
	}
	if width < 24 {
		width = 24
	}
	return tuikit.Center(body, width, h, 2)
}
