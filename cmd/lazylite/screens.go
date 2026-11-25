package main

// screens.go — the SCREEN registry: six named screens, flat, atomically
// switched. Every screen declares the same chrome shape (one status row,
// one hints row) and its own GRID: rows × slots, Fixed or Fill spans.
//
//	status   → two panes: files | commits        (the tile)
//	log      → one pane: commits, full width
//	branches → two panes: branches | worktree    (worktree has NO cursor)
//	detail   → one pane: the stacked DETAIL of the cursor's target
//	confirm  → one pane: the DANGER confirm context
//	palette  → one pane: the ':' action palette
//
// Transitions are host policy (goScreen / openPalette / pushConfirm /
// unwind); the kit only switches.

import (
	"ammaboard/internal/tuikit"

	tea "charm.land/bubbletea/v2"
)

func (m *Model) newScreens() {
	m.Screens.Register(m.screenStatus())
	m.Screens.Register(m.screenLog())
	m.Screens.Register(m.screenBranches())
	m.Screens.Register(m.screenDetail())
	m.Screens.Register(m.screenConfirm())
	m.Screens.Register(m.screenPalette())
}

func (m *Model) screenStatus() *tuikit.Screen {
	scr := tuikit.NewScreen("status", 100, 20, 2)
	scr.Frame.Row(tuikit.Fixed(1)) // chrome: status line
	cells := scr.Frame.Row(tuikit.Fill()).Cols(tuikit.Fixed(44), tuikit.Fill())
	scr.Frame.Row(tuikit.Fixed(1)) // chrome: hints line
	scr.Layout = func() []tuikit.Region {
		l, r := cells[0].Rect(), cells[1].Rect()
		return []tuikit.Region{m.filesPane(l.W, l.H), m.commitsPane(r.W, r.H)}
	}
	scr.RegionKeys = func(focus int) tuikit.Keymap {
		if focus == 0 {
			return m.filesKeys()
		}
		return m.commitsKeys()
	}
	return scr
}

func (m *Model) screenLog() *tuikit.Screen {
	scr := tuikit.NewScreen("log", 100, 20, 1)
	scr.Frame.Row(tuikit.Fixed(1))
	cell := scr.Frame.Row(tuikit.Fill()).Cols(tuikit.Fill())[0]
	scr.Frame.Row(tuikit.Fixed(1))
	scr.Layout = func() []tuikit.Region {
		r := cell.Rect()
		return []tuikit.Region{m.logPane(r.W, r.H)}
	}
	scr.RegionKeys = func(int) tuikit.Keymap { return m.commitsKeys() }
	return scr
}

func (m *Model) screenBranches() *tuikit.Screen {
	scr := tuikit.NewScreen("branches", 100, 20, 2)
	scr.Frame.Row(tuikit.Fixed(1))
	cells := scr.Frame.Row(tuikit.Fill()).Cols(tuikit.Fixed(36), tuikit.Fill())
	scr.Frame.Row(tuikit.Fixed(1))
	scr.Layout = func() []tuikit.Region {
		l, r := cells[0].Rect(), cells[1].Rect()
		return []tuikit.Region{m.branchesPane(l.W, l.H), m.worktreePane(r.W, r.H)}
	}
	// focus 1 is the worktree pane: a PANE WITHOUT A CURSOR. It claims no
	// keys, so while it is focused the verbs degrade gracefully — the
	// palette lists screen-scope actions only, and nothing else answers.
	scr.RegionKeys = func(focus int) tuikit.Keymap {
		if focus == 0 {
			return m.branchesKeys()
		}
		return tuikit.Keymap{}
	}
	return scr
}

func (m *Model) screenDetail() *tuikit.Screen {
	scr := tuikit.NewScreen("detail", 90, 22, 1)
	scr.Frame.Row(tuikit.Fixed(1))
	cell := scr.Frame.Row(tuikit.Fill()).Cols(tuikit.Fill())[0]
	scr.Frame.Row(tuikit.Fixed(1))
	scr.Layout = func() []tuikit.Region {
		r := cell.Rect()
		return []tuikit.Region{m.detailPane(r.W, r.H)}
	}
	// A DETAIL is a read place: no keys of its own. esc (global) pops it.
	return scr
}

func (m *Model) screenConfirm() *tuikit.Screen {
	scr := tuikit.NewScreen("confirm", 80, 18, 1)
	scr.Frame.Row(tuikit.Fixed(1))
	cell := scr.Frame.Row(tuikit.Fill()).Cols(tuikit.Fill())[0]
	scr.Frame.Row(tuikit.Fixed(1))
	scr.Layout = func() []tuikit.Region {
		r := cell.Rect()
		return []tuikit.Region{m.confirmPane(r.W, r.H)}
	}
	scr.RegionKeys = func(int) tuikit.Keymap {
		return tuikit.Keymap{
			tuikit.Bind("enter", "y").Help("confirm").Run(m.runConfirm),
			tuikit.Bind("n").Help("cancel").Run(m.cancelConfirm),
		}
	}
	return scr
}

func (m *Model) screenPalette() *tuikit.Screen {
	scr := tuikit.NewScreen("palette", 80, 18, 1)
	scr.Frame.Row(tuikit.Fixed(1))
	cell := scr.Frame.Row(tuikit.Fill()).Cols(tuikit.Fill())[0]
	scr.Frame.Row(tuikit.Fixed(1))
	scr.Layout = func() []tuikit.Region {
		r := cell.Rect()
		return []tuikit.Region{m.palettePane(r.W, r.H)}
	}
	// Typing is grabbed by Handle; the palette routes only its menu keys.
	scr.RegionKeys = func(int) tuikit.Keymap {
		return tuikit.Keymap{
			tuikit.Bind("up").Help("select up").Run(func() tea.Cmd { m.paletteSel(-1); return nil }),
			tuikit.Bind("down").Help("select down").Run(func() tea.Cmd { m.paletteSel(1); return nil }),
			tuikit.Bind("enter").Help("run selected").Run(m.runPaletteSel),
		}
	}
	return scr
}
