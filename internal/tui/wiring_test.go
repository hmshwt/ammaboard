package tui

// wiring_test.go — the kit-contract tests: the assembled wiring is a value
// graph, so it can be audited (AssertCleanWiring), and the routing/capture
// contract can be exercised headlessly. These are the tests the contract
// calls "the real gate": CI fails before anything ships.

import (
	"strings"
	"testing"

	"ammaboard/internal/tuikit"

	tea "charm.land/bubbletea/v2"
)

// The acceptance test's step 4: the fully wired app must audit clean.
func TestBoardWiringIsClean(t *testing.T) {
	m := newTestModel(t, "halt\n")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 145, Height: 30})
	m = next.(*Model)
	tuikit.AssertCleanWiring(t, m.App)
}

// The shell grab must swallow keys that would otherwise route: typing "q"
// into the shell types 'q' — it never reaches the global quit binding.
func TestShellGrabSwallowsRouting(t *testing.T) {
	m := newTestModel(t, "halt\n")
	m = key(m, ":")
	if !m.input.Focused() {
		t.Fatal("':' should focus the shell")
	}
	m = key(m, "q")
	if m.input.Value() != "q" {
		t.Fatalf("grab must feed the input, got %q", m.input.Value())
	}
	// esc leaves the grab.
	m = keyEsc(m)
	if m.input.Focused() {
		t.Fatal("esc should blur the shell")
	}
}

// Region keys route to the focused pane only: two j's under two different
// focuses scroll two different panes, each away from its top row.
func TestScrollRoutesToFocusedPane(t *testing.T) {
	m := newTestModel(t, longProgram(60))
	next, _ := m.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	m = next.(*Model)

	m = key(m, "j") // program pane (focus 0) scrolls one line
	m = key(m, "\t")
	m = key(m, "j") // machine pane (focus 1) scrolls one line

	v := m.View().Content
	if strings.Contains(v, "add_number r1, r0, 0") {
		t.Error("program pane should have scrolled past its first line")
	}
	if strings.Contains(v, "0000  0848") {
		t.Error("machine pane should have scrolled past its first memory row")
	}
}

// Under an open modal the kit captures: screen and region keys do not leak,
// global verbs answer, esc pops one level.
func TestHelpModalCapturesAndUnwinds(t *testing.T) {
	m := newTestModel(t, longProgram(4))

	m = key(m, "?")
	if !m.Modals.Active() {
		t.Fatal("'?' should open the help modal")
	}
	m = key(m, "s")
	if m.board.Major != 0 {
		t.Fatal("screen-scope keys must not leak under a modal")
	}
	m = key(m, "?") // global verbs answer: '?' toggles what '?' opened
	if m.Modals.Active() {
		t.Fatal("'?' under the modal should close it")
	}

	// esc pops one level.
	m = key(m, "?")
	m = keyEsc(m)
	if m.Modals.Active() {
		t.Fatal("esc should pop the modal")
	}
}

// The machine keeps stepping under the help overlay: ticks are not keys.
func TestTicksFlowUnderModal(t *testing.T) {
	m := newTestModel(t, longProgram(4))
	m.exec("run")
	m = key(m, "?")
	if !m.Modals.Active() {
		t.Fatal("precondition: help open")
	}
	next, _ := m.Update(frameMsg{})
	m = next.(*Model)
	if m.board.Major == 0 {
		t.Fatal("the machine must keep stepping while help is open")
	}
}

// keyEsc presses esc through the real Update path.
func keyEsc(m *Model) *Model {
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	return next.(*Model)
}
