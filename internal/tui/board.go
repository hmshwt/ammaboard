package tui

// board.go — the main screen: status row, three panes, shell block. The frame
// is declared once here (Fixed chrome, Fill panes, Cols widths); regions are
// derived fresh every frame in Layout — the kit renders them into the cell
// rects and marks the focused one. Adding a pane is one component file plus
// one entry in Model.panes and one column below.

import (
	"fmt"

	"ammaboard/internal/tuikit"

	tea "charm.land/bubbletea/v2"
)

// pane is one focusable region of the board screen: a pure view over machine
// state plus its own scroll. Panes never mutate the machine.
type pane interface {
	region(m *Model, w, h int) tuikit.Region
	keys() tuikit.Keymap // the region scope: pane keys stay documented because they are data
	reset()              // machine reset fans out to the view state
}

// newBoardScreen declares the board screen: geometry, key scopes, and the
// per-frame layout closure. Registration happens in New.
func newBoardScreen(m *Model) *tuikit.Screen {
	scr := tuikit.NewScreen("board", minWidth, minHeight, len(m.panes))
	scr.Frame.Row(tuikit.Fixed(1)) // status chrome
	cells := scr.Frame.Row(tuikit.Fill()).
		Cols(tuikit.Fixed(leftW), tuikit.Fill(), tuikit.Fixed(rightW))
	scr.Frame.Row(tuikit.Fixed(bottomH)) // shell box + keyhints chrome

	scr.Keys = tuikit.Keymap{
		tuikit.Bind("space").Help("run/pause").Run(m.toggleRun),
		tuikit.Bind("s").Help("step").Run(m.stepOne),
		tuikit.Bind("+", "=").Help("speed ×2").Run(m.faster),
		tuikit.Bind("-").Help("speed ÷2").Run(m.slower),
	}
	scr.RegionKeys = func(focus int) tuikit.Keymap { return m.panes[focus].keys() }
	scr.Layout = func() []tuikit.Region {
		regions := make([]tuikit.Region, len(cells))
		for i, c := range cells {
			r := c.Rect()
			regions[i] = m.panes[i].region(m, r.W, r.H)
		}
		return regions
	}
	return scr
}

// globalKeys is the global scope — the only scope that may claim reserved
// keys. tab/shift+tab are kit-owned (the ring cycles in App.Update); the
// bindings are display-only so the generated help stays truthful.
func (m *Model) globalKeys() tuikit.Keymap {
	return tuikit.Keymap{
		tuikit.Bind("tab", "shift+tab").Help("cycle pane focus").Run(nil),
		tuikit.Bind(":").Help("focus the shell (esc leaves)").Run(m.focusShell),
		tuikit.Bind("?").Help("toggle help").Run(m.toggleHelp),
		tuikit.Bind("q", "ctrl+c").Help("quit").Run(func() tea.Cmd { return tea.Quit }),
	}
}

// chrome fills the frame's cell-less rows in frame order: the status bar and
// the shell block. The kit splits on newlines and pads/clips to row heights,
// so one entry per chrome row is the shape the audit checks.
func (m *Model) chrome(w, _ int) []string {
	status := stTop.Width(w).Render(fmt.Sprintf(
		" AmmaBoard-2 · %s · major %d · %s · ×%d ", m.name, m.board.Major, m.runState(), m.speed))
	return []string{status, m.shellBlock(w)}
}
