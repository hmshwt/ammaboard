package main

// The lazylite tests double as the vocabulary's proof: the wiring audit
// runs on the assembled app (the kit's pre-ship gate), and every term —
// cursor, detail stack, modal, palette, confirm context, screen switch —
// is exercised through the REAL routing ladder by driving keys headlessly.

import (
	"testing"

	"ammaboard/internal/tuikit"

	tea "charm.land/bubbletea/v2"
)

func newTest() *Model { return New() }

func press(m *Model, key string) {
	var msg tea.Msg
	switch key {
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	case "space":
		msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	case "tab":
		msg = tea.KeyPressMsg{Code: tea.KeyTab}
	default:
		msg = tea.KeyPressMsg{Code: rune(key[0]), Text: key}
	}
	m.Update(msg)
}

func current(m *Model) string { return m.screenName() }

// The audit: geometry solvable at declared minimums, key scopes disciplined
// (no shadowing, reserved keys only global), region counts honest, chrome
// shaped right. Errors fail the test; warnings are logged for judgement.
func TestAuditClean(t *testing.T) {
	m := newTest()
	tuikit.AssertCleanWiring(t, m.App)
}

// CURSOR: j/k move the cursor within the focused pane; the highlight and
// the chrome both derive from it.
func TestCursorMoves(t *testing.T) {
	m := newTest()
	press(m, "j")
	if m.files.cur != 1 {
		t.Fatalf("after j: cursor = %d, want 1", m.files.cur)
	}
	press(m, "G")
	if m.files.cur != len(m.data.files)-1 {
		t.Fatalf("after G: cursor = %d, want last", m.files.cur)
	}
	press(m, "g")
	if m.files.cur != 0 {
		t.Fatalf("after g: cursor = %d, want 0", m.files.cur)
	}
	press(m, "k") // clamps at the top
	if m.files.cur != 0 {
		t.Fatalf("after k at top: cursor = %d, want 0", m.files.cur)
	}
}

// FOCUS: tab cycles the ring; the cursor keys then apply to the pane the
// ring points at — two levels of "where am I" (pane, then item).
func TestFocusRingRoutesCursor(t *testing.T) {
	m := newTest()
	press(m, "tab")
	if m.focusIdx() != 1 {
		t.Fatalf("after tab: focus = %d, want 1", m.focusIdx())
	}
	press(m, "j")
	if m.commits.cur != 1 || m.files.cur != 0 {
		t.Fatalf("after tab+j: commits=%d files=%d, want commits moved only",
			m.commits.cur, m.files.cur)
	}
}

// SCREENS switch — flat registry, atomic.
func TestScreensSwitch(t *testing.T) {
	m := newTest()
	press(m, "2")
	if current(m) != "log" {
		t.Fatalf("after 2: screen = %s, want log", current(m))
	}
	press(m, "esc") // unwind: no contexts open → home
	if current(m) != "status" {
		t.Fatalf("after esc: screen = %s, want status", current(m))
	}
}

// DETAIL: enter (or the '+' alias) pushes the cursor's target onto the
// stack; esc pops exactly one level, back to the origin screen.
func TestDetailStack(t *testing.T) {
	m := newTest()
	path := m.data.files[0].path
	press(m, "enter")
	if current(m) != "detail" || len(m.details) != 1 {
		t.Fatalf("after enter: screen=%s details=%d, want detail/1", current(m), len(m.details))
	}
	if d := m.detailTop(); d == nil || d.kind != dFile || d.origin != "status" {
		t.Fatalf("detail top = %+v, want file from status", m.detailTop())
	}
	press(m, "esc")
	if current(m) != "status" || len(m.details) != 0 {
		t.Fatalf("after esc: screen=%s details=%d, want status/0", current(m), len(m.details))
	}
	_ = path
	// the '+' alias opens the same detail
	press(m, "+")
	if current(m) != "detail" || len(m.details) != 1 {
		t.Fatalf("after +: screen=%s details=%d, want detail/1", current(m), len(m.details))
	}
}

// DANGER: the confirm context is a place with keys, not a hidden flag.
// esc cancels, y confirms, and the mutation applies only after confirm.
func TestDangerConfirmFlow(t *testing.T) {
	m := newTest()
	before := len(m.data.files)
	victim := m.data.files[m.files.cur].path

	press(m, "d")
	if current(m) != "confirm" || m.confirm == nil {
		t.Fatalf("after d: screen=%s, want confirm", current(m))
	}
	press(m, "esc") // unwind cancels
	if current(m) != "status" || len(m.data.files) != before {
		t.Fatalf("after esc on confirm: screen=%s files=%d, want status/%d",
			current(m), len(m.data.files), before)
	}

	press(m, "d")
	press(m, "y")
	if len(m.data.files) != before-1 {
		t.Fatalf("after y: files=%d, want %d", len(m.data.files), before-1)
	}
	for _, f := range m.data.files {
		if f.path == victim {
			t.Fatalf("discarded file %q still present", victim)
		}
	}
}

// PALETTE: ':' derives the actions available at the cursor; typing filters
// (the host grab); enter runs the selected action — danger actions still
// pass through the confirm context.
func TestPaletteFiltersAndRuns(t *testing.T) {
	m := newTest()
	before := len(m.data.files)

	press(m, ":")
	if current(m) != "palette" || m.palette == nil {
		t.Fatalf("after ':': screen=%s, want palette", current(m))
	}
	if len(m.palette.items) == 0 {
		t.Fatal("palette derived no actions")
	}

	// typing filters (printable keys are grabbed by Handle)
	press(m, "d")
	press(m, "i")
	press(m, "s")
	press(m, "c")
	if m.palette.query != "disc" {
		t.Fatalf("query = %q, want %q", m.palette.query, "disc")
	}
	filtered := m.filteredActions()
	if len(filtered) == 0 || !contains(filtered[0].name, "discard") {
		t.Fatalf("filtered = %v, want a discard action first", actionNames(filtered))
	}

	// enter runs it → danger → confirm context
	press(m, "enter")
	if current(m) != "confirm" {
		t.Fatalf("after enter on palette: screen=%s, want confirm", current(m))
	}
	press(m, "y")
	if len(m.data.files) != before-1 {
		t.Fatalf("after confirm: files=%d, want %d", len(m.data.files), before-1)
	}
}

// The cursorless pane degrades the verbs: the palette there lists the
// universal set only — no cursor-scoped actions.
func TestCursorlessPanePalette(t *testing.T) {
	m := newTest()
	press(m, "3") // branches screen
	press(m, "tab")
	if m.focusIdx() != 1 {
		t.Fatalf("focus = %d, want the worktree pane", m.focusIdx())
	}
	press(m, ":")
	if current(m) != "palette" {
		t.Fatalf("screen = %s, want palette", current(m))
	}
	for _, a := range m.palette.items {
		if contains(a.name, "checkout") || contains(a.name, "delete") {
			t.Fatalf("cursorless pane offered cursor-scoped action %q", a.name)
		}
	}
}

// MODAL: '?' pushes generated help; under capture, '?' (global verb)
// toggles it closed and esc pops it.
func TestHelpModalCapture(t *testing.T) {
	m := newTest()
	press(m, "?")
	if !m.Modals.Active() {
		t.Fatal("after '?': no modal active")
	}
	press(m, "?")
	if m.Modals.Active() {
		t.Fatal("'?' under capture should toggle the modal closed")
	}
	press(m, "?")
	press(m, "esc")
	if m.Modals.Active() {
		t.Fatal("esc should pop the modal")
	}
}

// ACTION: staging mutates the fixture and the view reflects it immediately
// (immediate mode: the next frame re-derives from the new state).
func TestStageAndCommit(t *testing.T) {
	m := newTest()
	staged := 0
	for i := range m.data.files {
		if m.data.files[i].staged {
			staged++
		}
	}
	press(m, "space")
	if m.data.stagedCount() != staged+1 {
		t.Fatalf("staged = %d, want %d", m.data.stagedCount(), staged+1)
	}
	press(m, "c")
	if m.data.stagedCount() != 0 {
		t.Fatalf("commit left %d staged files", m.data.stagedCount())
	}
	if m.data.commits[0].author != "you" {
		t.Fatalf("new commit missing at top: %+v", m.data.commits[0])
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func actionNames(as []action) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.name
	}
	return out
}
