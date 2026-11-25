// Command lazylite is a lite lazygit whose job is to make the tuikit
// vocabulary concrete. Every noun in the naming table is a real thing you
// can touch here:
//
//	Screens switch   1 status · 2 log · 3 branches    (flat registry, atomic)
//	Panes tile       tab cycles focus                 (boxed areas in the grid)
//	Cursors point    j/k/g/G move it                  (the app knows what you're on)
//	Details stack    enter or + opens, esc pops       (parameterized views)
//	Modals capture   ? toggles generated help         (view-only; esc pops)
//	Palette          : lists actions at the cursor    (type to filter)
//	Actions          space mutates · d/r/x are DANGER → confirm context
//	Chrome           the top line always names screen · pane · cursor
//	Audit            ammaboard AssertCleanWiring runs in the tests
//
// The fixture is a deterministic snapshot of a repo — no git, no network.
// What is demonstrated is the architecture, not version control.
//
//	ammaboard run  → the machine.        lazylite  → the vocabulary.
package main

import (
	"fmt"
	"os"

	"ammaboard/internal/tuikit"

	tea "charm.land/bubbletea/v2"
)

// cursorState is one PANE's pointing state: the CURSOR (an index into the
// pane's data) plus the kit's Scroll, which windows the body. The window
// always follows the cursor — Scroll.Window(track, ...) is the follow mode
// the kit already ships.
type cursorState struct {
	cur    int
	scroll tuikit.Scroll
}

// detail is one stacked DETAIL: which item, and where to pop back to.
// Screens switch; details stack.
type detail struct {
	kind   detailKind
	idx    int
	origin string
}

type detailKind int

const (
	dFile detailKind = iota
	dCommit
	dBranch
)

// action is the declared write path: a cursor-scoped executable with its
// key hint and a danger flag. DANGER actions never run directly — they
// push a confirm context (a place you can inspect, not a dialog you trust).
type action struct {
	keys   string
	name   string
	danger bool
	run    func() tea.Cmd
}

// paletteState is the ':' context: the actions available HERE, filtered by
// typed query. It grabs typing in Handle (the host's pre-routing hook —
// the same pattern as a shell input).
type paletteState struct {
	origin string
	query  string
	sel    int
	items  []action
}

// confirmState is the DANGER context: a rendered preview of the mutation,
// confirmed or cancelled with keys that route through the kit like any
// other place — never a hidden flag.
type confirmState struct {
	title   string
	preview []string
	run     func()
}

// Model is the lazylite root: the kit's wiring root plus the state the kit
// deliberately does not own — the data, the per-pane cursors, and the
// context stack.
type Model struct {
	*tuikit.App

	data *repo

	// PANE state (one per pane; each pane owns its cursor).
	files    cursorState
	commits  cursorState
	branches cursorState

	// DETAIL stack + the two single-level contexts.
	palette *paletteState
	confirm *confirmState
	origin  string // the screen to return to (palette/confirm)
	details []detail

	notice string // last action feedback, shown in the chrome
}

// New wires the whole app: kit root, global keys, chrome, six screens.
func New() *Model {
	m := &Model{App: tuikit.NewApp(), data: seedRepo()}
	m.Keys = m.globalKeys()
	m.Handle = m.handle
	m.Chrome = m.chrome
	m.newScreens() // first registered becomes current: status
	return m
}

func (m *Model) Init() tea.Cmd { return nil }

// Update delegates to the kit's routing contract; host state mutates
// through Handle and the keymap Run closures, so the model is always m.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.App.Update(msg)
	return m, cmd
}

func (m *Model) View() tea.View { return m.App.View() }

func main() {
	if _, err := tea.NewProgram(New()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazylite:", err)
		os.Exit(1)
	}
}

// ── CHROME ────────────────────────────────────────────────────────────────
// Every screen declares the same chrome shape (one status row, one hints
// row), so chrome always returns exactly two strings. The status line
// NAMES the current place — screen, pane, cursor — which is the demo of
// "the app always knows where you are".

func (m *Model) chrome(w, h int) []string {
	return []string{m.statusLine(), m.hints()}
}

func (m *Model) statusLine() string {
	line := " lazylite · screen " + m.screenName() + m.contextSuffix()
	if m.notice != "" {
		line += "  ·  " + m.notice
	}
	return line
}

// hints stays deliberately global-only: pane-level keys live in the
// context-aware help ('?'), not crammed into one line. The bottom line is
// short because help knows where you are.
func (m *Model) hints() string { return m.Keys.Hints() }

func (m *Model) screenName() string {
	if s := m.Screens.Current(); s != nil {
		return s.Name
	}
	return "?"
}

func (m *Model) focusIdx() int {
	if s := m.Screens.Current(); s != nil {
		return s.FocusRing.Index()
	}
	return 0
}

func (m *Model) contextSuffix() string {
	switch m.screenName() {
	case "status":
		if m.focusIdx() == 0 {
			if f := m.fileAt(); f != nil {
				return fmt.Sprintf(" · pane files · cursor %s (%d/%d) %s",
					f.path, m.files.cur+1, len(m.data.files), f.status)
			}
			return " · pane files · (empty)"
		}
		return m.commitSuffix("commits")
	case "log":
		return m.commitSuffix("log")
	case "branches":
		if m.focusIdx() == 0 {
			if b := m.branchAt(); b != nil {
				return fmt.Sprintf(" · pane branches · cursor %s (%d/%d)",
					b.name, m.branches.cur+1, len(m.data.branches))
			}
			return " · pane branches · (empty)"
		}
		return " · pane worktree · no cursor (static pane)"
	case "detail":
		if d := m.detailTop(); d != nil {
			return " · detail of " + m.detailName(*d)
		}
	case "confirm":
		if m.confirm != nil {
			return " · confirm: " + m.confirm.title
		}
	case "palette":
		if m.palette != nil {
			return fmt.Sprintf(" · palette: filter %q · %d/%d actions",
				m.palette.query, len(m.filteredActions()), len(m.palette.items))
		}
	}
	return ""
}

func (m *Model) commitSuffix(pane string) string {
	if c := m.commitAt(); c != nil {
		return fmt.Sprintf(" · pane %s · cursor %s %s (%d/%d)",
			pane, c.hash, c.subject, m.commits.cur+1, len(m.data.commits))
	}
	return " · pane " + pane + " · (empty)"
}

// ── GLOBAL KEYS ───────────────────────────────────────────────────────────
// The only scope allowed to claim reserved keys (esc, q, ctrl+c).

func (m *Model) globalKeys() tuikit.Keymap {
	return tuikit.Keymap{
		// FOCUS: tab is kit-owned (the ring cycles in App.Update); this
		// binding is display-only so the generated help stays truthful.
		tuikit.Bind("tab", "shift+tab").Help("cycle focus").Run(nil),
		// SCREENS switch. Stacked contexts belong to a place: jumping
		// screens abandons them (esc unwinds gracefully; numbers jump).
		tuikit.Bind("1").Help("status").Run(func() tea.Cmd { return m.goScreen("status") }),
		tuikit.Bind("2").Help("log").Run(func() tea.Cmd { return m.goScreen("log") }),
		tuikit.Bind("3").Help("branches").Run(func() tea.Cmd { return m.goScreen("branches") }),
		// PALETTE: everything you can do here, type-to-filter.
		tuikit.Bind(":").Help("actions").Run(m.openPalette),
		// MODAL: generated help that knows where you are.
		tuikit.Bind("?").Help("help").Run(m.toggleHelp),
		// The unwind rule: esc pops the innermost context, exactly one
		// level: confirm → palette → detail → leave screen → nothing.
		tuikit.Bind("esc").Help("back").Run(m.unwind),
		tuikit.Bind("q", "ctrl+c").Help("quit").Run(func() tea.Cmd { return tea.Quit }),
	}
}

// ── HOST HOOK ─────────────────────────────────────────────────────────────
// Handle sees every message before routing. The only grab is the palette's
// typing: printable keys filter the action list. Everything else (arrows,
// enter, esc, even q) falls through to the kit's routing — which is why
// quit and unwind keep working inside the palette.

func (m *Model) handle(msg tea.Msg) (tea.Cmd, bool) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok || m.palette == nil {
		return nil, false // no grab: the kit routes it
	}
	if key.String() == "backspace" {
		if n := len(m.palette.query); n > 0 {
			m.palette.query = m.palette.query[:n-1]
			m.palette.sel = 0
		}
		return nil, true
	}
	if key.Text != "" { // printable → the filter
		m.palette.query += key.Text
		m.palette.sel = 0
		return nil, true
	}
	return nil, false
}

// ── CONTEXT LIFECYCLE ─────────────────────────────────────────────────────

// goScreen is a SCREEN switch: atomic, and it abandons stacked contexts —
// they belong to a place, and you just left the place.
func (m *Model) goScreen(name string) tea.Cmd {
	m.details = nil
	if m.confirm != nil {
		m.confirm = nil
		m.setNotice("cancelled — nothing changed")
	}
	m.Screens.Switch(name)
	return nil
}

// unwind is the whole esc contract in one function.
func (m *Model) unwind() tea.Cmd {
	switch {
	case m.confirm != nil:
		m.confirm = nil
		m.setNotice("cancelled — nothing changed")
		m.Screens.Switch(m.origin)
	case m.palette != nil:
		m.closePalette()
	case len(m.details) > 0:
		d := m.details[len(m.details)-1]
		m.details = m.details[:len(m.details)-1]
		m.Screens.Switch(d.origin)
	case m.screenName() != "status":
		m.Screens.Switch("status")
	}
	return nil
}

func (m *Model) openPalette() tea.Cmd {
	if m.palette != nil {
		return nil
	}
	m.palette = &paletteState{origin: m.screenName(), items: m.actionsHere()}
	m.Screens.Switch("palette")
	return nil
}

func (m *Model) closePalette() {
	p := m.palette
	m.palette = nil
	if p != nil {
		m.Screens.Switch(p.origin)
	}
}

func (m *Model) pushConfirm(title string, preview []string, run func()) {
	m.confirm = &confirmState{title: title, preview: preview, run: run}
	m.origin = m.screenName()
	m.Screens.Switch("confirm")
}

func (m *Model) toggleHelp() tea.Cmd {
	if m.Modals.Active() {
		m.Modals.Pop()
	} else {
		m.Modals.Push(helpModal{m})
	}
	return nil
}

func (m *Model) setNotice(s string) { m.notice = s }

// ── ITEM LOOKUPS ──────────────────────────────────────────────────────────

func (m *Model) fileAt() *fileItem {
	if m.files.cur < len(m.data.files) {
		return &m.data.files[m.files.cur]
	}
	return nil
}

func (m *Model) commitAt() *commitItem {
	if m.commits.cur < len(m.data.commits) {
		return &m.data.commits[m.commits.cur]
	}
	return nil
}

func (m *Model) branchAt() *branchItem {
	if m.branches.cur < len(m.data.branches) {
		return &m.data.branches[m.branches.cur]
	}
	return nil
}

func (m *Model) detailTop() *detail {
	if len(m.details) > 0 {
		return &m.details[len(m.details)-1]
	}
	return nil
}

func (m *Model) detailName(d detail) string {
	switch d.kind {
	case dFile:
		if d.idx < len(m.data.files) {
			return m.data.files[d.idx].path
		}
	case dCommit:
		if d.idx < len(m.data.commits) {
			return m.data.commits[d.idx].hash
		}
	case dBranch:
		if d.idx < len(m.data.branches) {
			return m.data.branches[d.idx].name
		}
	}
	return "?"
}
