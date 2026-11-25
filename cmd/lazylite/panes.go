package main

// panes.go — the PANE renderers (pure per-frame derivations: a pane is a
// value built fresh from data + cursor, never a cached widget) and the
// pane keymaps (CURSOR movement + ACTION declarations).
//
// Body sizing follows the kit's contract: Region.Render spends
// 2 rows on the border and 1 on the title, so the body gets h-3-len(header).

import (
	"fmt"
	"strings"

	"ammaboard/internal/tuikit"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	stCursor = lipgloss.NewStyle().Background(lipgloss.Color("23")).Foreground(lipgloss.Color("255"))
	stDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	stGood   = lipgloss.NewStyle().Foreground(lipgloss.Color("35"))
	stWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	stMark   = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
)

// cursorKeys is the shared CURSOR keymap fragment: j/k move, g/G jump.
// The window always follows — Scroll.Window(track, ...) pins the cursor
// one third from the top, the kit's built-in follow mode.
func cursorKeys(st *cursorState, count func() int) tuikit.Keymap {
	move := func(d int) func() tea.Cmd {
		return func() tea.Cmd {
			st.cur = tuikit.Clamp(st.cur+d, 0, count()-1)
			return nil
		}
	}
	return tuikit.Keymap{
		tuikit.Bind("j", "down").Help("cursor down").Run(move(1)),
		tuikit.Bind("k", "up").Help("cursor up").Run(move(-1)),
		tuikit.Bind("g").Help("first item").Run(func() tea.Cmd { st.cur = 0; return nil }),
		tuikit.Bind("G").Help("last item").Run(func() tea.Cmd {
			if n := count(); n > 0 {
				st.cur = n - 1
			}
			return nil
		}),
	}
}

// ── PANE: files ───────────────────────────────────────────────────────────

func (m *Model) filesKeys() tuikit.Keymap {
	return tuikit.Merge(
		cursorKeys(&m.files, func() int { return len(m.data.files) }),
		tuikit.Keymap{
			// DETAIL: enter is the primary opener; '+' is the alias.
			tuikit.Bind("enter", "+").Help("detail").Run(m.openFileDetail),
			// ACTIONS: the write path, declared.
			tuikit.Bind("space").Help("stage / unstage").Run(m.toggleStage),
			tuikit.Bind("c").Help("commit staged").Run(m.commitStaged),
			// DANGER: never runs directly — pushes the confirm context.
			tuikit.Bind("d").Help("discard — confirm").Run(m.discardFile),
		},
	)
}

func (m *Model) filesPane(w, h int) tuikit.Region {
	header := []tuikit.Line{tuikit.Plain("st status  path")}
	return tuikit.Region{
		Title:       "pane: files",
		Header:      header,
		Body:        m.fileLines(h - 3 - len(header)),
		Constraints: tuikit.Constraints{MinW: 40, MinH: 10},
	}
}

func (m *Model) fileLines(bh int) []tuikit.Line {
	if bh <= 0 {
		return nil
	}
	n := len(m.data.files)
	top := m.files.scroll.Window(m.files.cur, n, bh)
	out := make([]tuikit.Line, 0, bh)
	for i := top; i < n && len(out) < bh; i++ {
		f := &m.data.files[i]
		mark := " "
		if f.staged {
			mark = "S"
		}
		text := fmt.Sprintf("%-2s %-3s %s", mark, f.status, f.path)
		if i == m.files.cur {
			out = append(out, tuikit.Styled(text, stCursor)) // the CURSOR line
		} else {
			out = append(out, tuikit.Plain(text))
		}
	}
	return out
}

// ── PANE: commits (tiled on status; full-width on log) ────────────────────

func (m *Model) commitsKeys() tuikit.Keymap {
	return tuikit.Merge(
		cursorKeys(&m.commits, func() int { return len(m.data.commits) }),
		tuikit.Keymap{
			tuikit.Bind("enter", "+").Help("detail").Run(m.openCommitDetail),
			tuikit.Bind("r").Help("revert — confirm").Run(m.revertCommit),
		},
	)
}

func (m *Model) commitsPane(w, h int) tuikit.Region {
	return tuikit.Region{
		Title:       "pane: commits",
		Body:        m.commitLines(h-3, false),
		Constraints: tuikit.Constraints{MinW: 40, MinH: 10},
	}
}

func (m *Model) logPane(w, h int) tuikit.Region {
	return tuikit.Region{
		Title:       "pane: log",
		Body:        m.commitLines(h-3, true),
		Constraints: tuikit.Constraints{MinW: 60, MinH: 10},
	}
}

func (m *Model) commitLines(bh int, withWhen bool) []tuikit.Line {
	if bh <= 0 {
		return nil
	}
	n := len(m.data.commits)
	top := m.commits.scroll.Window(m.commits.cur, n, bh)
	out := make([]tuikit.Line, 0, bh)
	for i := top; i < n && len(out) < bh; i++ {
		c := &m.data.commits[i]
		text := fmt.Sprintf("%s  %s", c.hash, c.subject)
		if withWhen {
			text = fmt.Sprintf("%s  %-7s %-5s %s", c.hash, c.when, c.author, c.subject)
		}
		if i == m.commits.cur {
			out = append(out, tuikit.Styled(text, stCursor))
		} else {
			out = append(out, tuikit.Plain(text))
		}
	}
	return out
}

// ── PANE: branches + worktree (the cursorless pane) ───────────────────────

func (m *Model) branchesKeys() tuikit.Keymap {
	return tuikit.Merge(
		cursorKeys(&m.branches, func() int { return len(m.data.branches) }),
		tuikit.Keymap{
			tuikit.Bind("enter", "+").Help("detail").Run(m.openBranchDetail),
			tuikit.Bind("space").Help("checkout — confirm").Run(m.checkoutBranch),
			tuikit.Bind("x").Help("delete — confirm").Run(m.deleteBranch),
		},
	)
}

func (m *Model) branchesPane(w, h int) tuikit.Region {
	return tuikit.Region{
		Title:       "pane: branches",
		Body:        m.branchLines(h - 3),
		Constraints: tuikit.Constraints{MinW: 30, MinH: 10},
	}
}

func (m *Model) branchLines(bh int) []tuikit.Line {
	if bh <= 0 {
		return nil
	}
	n := len(m.data.branches)
	top := m.branches.scroll.Window(m.branches.cur, n, bh)
	out := make([]tuikit.Line, 0, bh)
	for i := top; i < n && len(out) < bh; i++ {
		b := &m.data.branches[i]
		mark, extra := " ", ""
		if b.current {
			mark, extra = "*", "  (checked out)"
		}
		text := fmt.Sprintf("%s %-22s +%-3d -%-3d%s", mark, b.name, b.ahead, b.behind, extra)
		if i == m.branches.cur {
			out = append(out, tuikit.Styled(text, stCursor))
		} else {
			out = append(out, tuikit.Plain(text))
		}
	}
	return out
}

// worktreePane is a PANE WITHOUT A CURSOR: static status, no bindings.
// The verbs degrade around it — proof that cursor-owning is optional.
func (m *Model) worktreePane(w, h int) tuikit.Region {
	body := []tuikit.Line{
		tuikit.Plain(fmt.Sprintf("branch     %s (checked out)", m.currentBranch())),
		tuikit.Plain(fmt.Sprintf("staged     %d file(s)", m.data.stagedCount())),
		tuikit.Plain(fmt.Sprintf("dirty      %d file(s)", len(m.data.files))),
		tuikit.Plain(fmt.Sprintf("commits    %d on main", len(m.data.commits))),
		tuikit.Line{},
		tuikit.Styled("this pane has no cursor:", stDim),
		tuikit.Styled("the palette here lists", stDim),
		tuikit.Styled("screen-scope actions only", stDim),
	}
	return tuikit.Region{
		Title:       "pane: worktree",
		Body:        clipLines(body, h-3),
		Constraints: tuikit.Constraints{MinW: 40, MinH: 10},
	}
}

func (m *Model) currentBranch() string {
	for i := range m.data.branches {
		if m.data.branches[i].current {
			return m.data.branches[i].name
		}
	}
	return "?"
}

// ── DETAIL (the stack) ────────────────────────────────────────────────────

func (m *Model) openFileDetail() tea.Cmd {
	if m.fileAt() == nil {
		return nil
	}
	m.pushDetail(detail{kind: dFile, idx: m.files.cur})
	return nil
}

func (m *Model) openCommitDetail() tea.Cmd {
	if m.commitAt() == nil {
		return nil
	}
	m.pushDetail(detail{kind: dCommit, idx: m.commits.cur})
	return nil
}

func (m *Model) openBranchDetail() tea.Cmd {
	if m.branchAt() == nil {
		return nil
	}
	m.pushDetail(detail{kind: dBranch, idx: m.branches.cur})
	return nil
}

func (m *Model) pushDetail(d detail) {
	d.origin = m.screenName()
	m.details = append(m.details, d)
	m.Screens.Switch("detail")
}

func (m *Model) detailPane(w, h int) tuikit.Region {
	d := m.detailTop()
	if d == nil {
		return tuikit.Region{Title: "detail — nothing open",
			Body: []tuikit.Line{tuikit.Plain("esc returns to where you were")}}
	}
	switch d.kind {
	case dFile:
		f := &m.data.files[d.idx]
		header := []tuikit.Line{tuikit.Plain(fmt.Sprintf("path %s · status %s · staged %v",
			f.path, f.status, f.staged))}
		return tuikit.Region{Title: "detail — file: " + f.path, Header: header,
			Body: clipLines(diffLines(f.diff), h-3-len(header)), Constraints: tuikit.Constraints{MinW: 60, MinH: 12}}
	case dCommit:
		c := &m.data.commits[d.idx]
		header := []tuikit.Line{tuikit.Plain(fmt.Sprintf("commit %s · %s · %s", c.hash, c.author, c.when))}
		body := []tuikit.Line{tuikit.Styled(c.subject, stMark), tuikit.Line{}}
		for _, l := range c.body {
			body = append(body, tuikit.Plain("  "+l))
		}
		body = append(body, tuikit.Line{}, tuikit.Plain("files touched:"))
		for _, f := range c.files {
			body = append(body, tuikit.Plain("  "+f))
		}
		return tuikit.Region{Title: "detail — commit: " + c.hash, Header: header,
			Body: clipLines(body, h-3-len(header)), Constraints: tuikit.Constraints{MinW: 60, MinH: 12}}
	default:
		b := &m.data.branches[d.idx]
		header := []tuikit.Line{tuikit.Plain(fmt.Sprintf("branch %s · %s",
			b.name, map[bool]string{true: "checked out", false: "not checked out"}[b.current]))}
		body := []tuikit.Line{
			tuikit.Plain(fmt.Sprintf("ahead %d, behind %d (vs origin)", b.ahead, b.behind)),
		}
		return tuikit.Region{Title: "detail — branch: " + b.name, Header: header,
			Body: clipLines(body, h-3-len(header)), Constraints: tuikit.Constraints{MinW: 60, MinH: 12}}
	}
}

func diffLines(diff []string) []tuikit.Line {
	out := make([]tuikit.Line, 0, len(diff))
	for _, l := range diff {
		switch {
		case strings.HasPrefix(l, "+"):
			out = append(out, tuikit.Styled(l, stGood))
		case strings.HasPrefix(l, "-"):
			out = append(out, tuikit.Styled(l, stWarn))
		case strings.HasPrefix(l, "@@"):
			out = append(out, tuikit.Styled(l, stDim))
		default:
			out = append(out, tuikit.Plain(l))
		}
	}
	return out
}

// ── CONFIRM (the DANGER context) ──────────────────────────────────────────

func (m *Model) confirmPane(w, h int) tuikit.Region {
	if m.confirm == nil {
		return tuikit.Region{Title: "confirm — nothing pending"}
	}
	body := []tuikit.Line{tuikit.Styled("DANGER — this mutates the fixture", stWarn), tuikit.Line{}}
	for _, l := range m.confirm.preview {
		body = append(body, tuikit.Plain("  "+l))
	}
	body = append(body, tuikit.Line{},
		tuikit.Plain("enter/y confirm · n cancel · esc back (unwind)"))
	return tuikit.Region{Title: "confirm — " + m.confirm.title,
		Body: clipLines(body, h-3), Constraints: tuikit.Constraints{MinW: 50, MinH: 10}}
}

func (m *Model) runConfirm() tea.Cmd {
	if m.confirm == nil {
		return nil
	}
	c := m.confirm
	m.confirm = nil
	m.Screens.Switch(m.origin)
	c.run()
	return nil
}

func (m *Model) cancelConfirm() tea.Cmd {
	if m.confirm == nil {
		return nil
	}
	m.confirm = nil
	m.setNotice("cancelled — nothing changed")
	m.Screens.Switch(m.origin)
	return nil
}

// ── ACTIONS (the declared write path) ─────────────────────────────────────

func (m *Model) toggleStage() tea.Cmd {
	if f := m.fileAt(); f != nil {
		f.staged = !f.staged
		m.setNotice(fmt.Sprintf("%s %s", map[bool]string{true: "staged", false: "unstaged"}[f.staged], f.path))
	}
	return nil
}

func (m *Model) commitStaged() tea.Cmd {
	var paths []string
	for i := range m.data.files {
		if m.data.files[i].staged {
			paths = append(paths, m.data.files[i].path)
		}
	}
	if len(paths) == 0 {
		m.setNotice("nothing staged — space stages a file")
		return nil
	}
	c := commitItem{
		hash:    m.data.fakeHash(),
		subject: fmt.Sprintf("wip: %d staged file(s)", len(paths)),
		author:  "you", when: "now",
		body:  []string{"Committed from the lazylite palette/pane action."},
		files: paths,
	}
	m.data.commits = append([]commitItem{c}, m.data.commits...)
	for i := range m.data.files {
		m.data.files[i].staged = false
	}
	m.commits.cur = 0
	m.commits.scroll.Reset()
	m.setNotice(fmt.Sprintf("committed %s · %d file(s)", c.hash, len(paths)))
	return nil
}

func (m *Model) discardFile() tea.Cmd {
	f := m.fileAt()
	if f == nil {
		return nil
	}
	path := f.path
	m.pushConfirm("discard "+path, []string{
		"$ git checkout -- " + path,
		"all uncommitted changes to " + path + " are lost",
	}, func() {
		m.data.removeFile(path)
		m.files.cur = tuikit.Clamp(m.files.cur, 0, len(m.data.files)-1)
		m.files.scroll.Reset()
		m.setNotice("discarded " + path)
	})
	return nil
}

func (m *Model) revertCommit() tea.Cmd {
	if c := m.commitAt(); c != nil {
		m.pushConfirm("revert "+c.hash, []string{
			"$ git revert " + c.hash,
			"creates an inverse commit on main",
		}, func() {
			m.setNotice("reverted " + c.hash + " (fixture: nothing actually changed)")
		})
	}
	return nil
}

func (m *Model) checkoutBranch() tea.Cmd {
	if b := m.branchAt(); b != nil {
		if b.current {
			m.setNotice("already on " + b.name)
			return nil
		}
		name := b.name
		m.pushConfirm("checkout "+name, []string{
			"$ git switch " + name,
			"uncommitted files travel with you",
		}, func() {
			for i := range m.data.branches {
				m.data.branches[i].current = m.data.branches[i].name == name
			}
			m.setNotice("switched to " + name)
		})
	}
	return nil
}

func (m *Model) deleteBranch() tea.Cmd {
	if b := m.branchAt(); b != nil {
		if b.current {
			m.setNotice("cannot delete the checked-out branch")
			return nil
		}
		name := b.name
		m.pushConfirm("delete "+name, []string{
			"$ git branch -D " + name,
			"unmerged commits on this branch are lost",
		}, func() {
			m.data.removeBranch(name)
			m.branches.cur = tuikit.Clamp(m.branches.cur, 0, len(m.data.branches)-1)
			m.branches.scroll.Reset()
			m.setNotice("deleted " + name)
		})
	}
	return nil
}

// ── PALETTE (':' — actions available here) ────────────────────────────────

// actionsHere derives the palette from the place: universal verbs plus the
// CURSOR-scoped actions. A cursorless pane gets the universal set only.
func (m *Model) actionsHere() []action {
	var acts []action
	switch m.screenName() {
	case "status":
		if m.focusIdx() == 0 {
			if f := m.fileAt(); f != nil {
				acts = append(acts,
					action{"enter", "open detail: " + f.path, false, m.openFileDetail},
					action{"space", "stage / unstage: " + f.path, false, m.toggleStage},
					action{"c", "commit staged files", false, m.commitStaged},
					action{"d", "discard: " + f.path, true, m.discardFile},
				)
			}
		} else if c := m.commitAt(); c != nil {
			acts = append(acts,
				action{"enter", "open detail: " + c.hash, false, m.openCommitDetail},
				action{"r", "revert: " + c.hash, true, m.revertCommit},
			)
		}
	case "log":
		if c := m.commitAt(); c != nil {
			acts = append(acts,
				action{"enter", "open detail: " + c.hash, false, m.openCommitDetail},
				action{"r", "revert: " + c.hash, true, m.revertCommit},
			)
		}
	case "branches":
		if m.focusIdx() == 0 {
			if b := m.branchAt(); b != nil {
				acts = append(acts,
					action{"enter", "open detail: " + b.name, false, m.openBranchDetail},
					action{"space", "checkout: " + b.name, true, m.checkoutBranch},
					action{"x", "delete: " + b.name, true, m.deleteBranch},
				)
			}
		}
	}
	return append(acts,
		action{"1", "go: status screen", false, func() tea.Cmd { m.closePalette(); return m.goScreen("status") }},
		action{"2", "go: log screen", false, func() tea.Cmd { m.closePalette(); return m.goScreen("log") }},
		action{"3", "go: branches screen", false, func() tea.Cmd { m.closePalette(); return m.goScreen("branches") }},
		action{"?", "toggle help", false, func() tea.Cmd { m.closePalette(); return m.toggleHelp() }},
	)
}

func (m *Model) filteredActions() []action {
	if m.palette == nil {
		return nil
	}
	q := strings.ToLower(m.palette.query)
	var out []action
	for _, a := range m.palette.items {
		if q == "" ||
			strings.Contains(strings.ToLower(a.name), q) ||
			strings.Contains(strings.ToLower(a.keys), q) {
			out = append(out, a)
		}
	}
	return out
}

func (m *Model) paletteSel(d int) tea.Cmd {
	if m.palette == nil {
		return nil
	}
	n := len(m.filteredActions())
	m.palette.sel = tuikit.Clamp(m.palette.sel+d, 0, n-1)
	return nil
}

func (m *Model) runPaletteSel() tea.Cmd {
	if m.palette == nil {
		return nil
	}
	filtered := m.filteredActions()
	if m.palette.sel >= len(filtered) {
		return nil
	}
	a := filtered[m.palette.sel]
	m.closePalette() // always leave the palette before acting
	return a.run()
}

func (m *Model) palettePane(w, h int) tuikit.Region {
	if m.palette == nil {
		return tuikit.Region{Title: "palette — nothing open",
			Body: []tuikit.Line{tuikit.Plain("(: opens the actions available at the cursor)")}}
	}
	filtered := m.filteredActions()
	header := []tuikit.Line{tuikit.Plain(fmt.Sprintf(
		"filter %q · %d/%d actions · type to filter · up/down select · enter run",
		m.palette.query, len(filtered), len(m.palette.items)))}
	var body []tuikit.Line
	for i, a := range filtered {
		mark := a.keys
		if a.danger {
			mark = "!" + a.keys
		}
		text := fmt.Sprintf("%-7s %s", mark, a.name)
		if i == m.palette.sel {
			body = append(body, tuikit.Styled(text, stCursor))
		} else if a.danger {
			body = append(body, tuikit.Styled(text, stWarn))
		} else {
			body = append(body, tuikit.Plain(text))
		}
	}
	if len(body) == 0 {
		body = []tuikit.Line{tuikit.Styled("(no actions match)", stDim)}
	}
	return tuikit.Region{Title: "palette — actions here", Header: header,
		Body: clipLines(body, h-3-len(header)), Constraints: tuikit.Constraints{MinW: 50, MinH: 10}}
}

// ── small helpers ─────────────────────────────────────────────────────────

func clipLines(lines []tuikit.Line, max int) []tuikit.Line {
	if max < 0 {
		max = 0
	}
	if len(lines) > max {
		return lines[:max]
	}
	return lines
}
