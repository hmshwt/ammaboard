package ui

// ui_test.go — the model is a pure state machine over Msgs, so the whole TUI
// is testable headlessly: drive Update with key/tick messages and assert on
// the model and the rendered View.

import (
	"fmt"
	"strings"
	"testing"

	"ammaboard/internal/asm"
	"ammaboard/internal/machine"
	"ammaboard/internal/machine/cpu"
	"ammaboard/internal/machine/ram"

	tea "charm.land/bubbletea/v2"
)

// newTestModel wires a minimal board (RAM + CPU, no devices) with a tiny
// program. The TUI's production wiring lives in cmd/ammaboard/wiring.go; this
// mirrors just enough of it to test the model.
func newTestModel(t *testing.T, src string) Model {
	t.Helper()
	words, err := asm.Assemble(src)
	if err != nil {
		t.Fatal(err)
	}
	return newTestModelWords(t, words)
}

// newTestModelWords is newTestModel for raw words (hand-rolled binaries,
// data areas, fault bait).
func newTestModelWords(t *testing.T, words []uint32) Model {
	t.Helper()
	events := NewEventLog(64)
	newMachine := func() (*machine.Motherboard, *cpu.CPU, *ram.RAM) {
		mem := ram.New(0, 16*1024)
		proc := cpu.New(0)
		board := machine.New(events.Add)
		board.Attach(mem)
		board.Attach(proc)
		if err := mem.Load(0, words); err != nil {
			t.Fatal(err)
		}
		return board, proc, mem
	}
	return New("test.amm", words, events, newMachine)
}

func TestParseShell(t *testing.T) {
	ok := []struct {
		line string
		name string
		arg  int
	}{
		{"run", "run", 1},
		{"  reset ", "reset", 1},
		{"step", "step", 1},
		{"step 5", "step", 5},
		{"speed 64", "speed", 64},
		{"quit", "quit", 1},
		{"help", "help", 1},
	}
	for _, c := range ok {
		got, err := parseShell(c.line)
		if err != nil {
			t.Errorf("%q: %v", c.line, err)
			continue
		}
		if got.name != c.name || got.arg != c.arg {
			t.Errorf("%q = %+v, want %s %d", c.line, got, c.name, c.arg)
		}
	}
	bad := []string{"", "frobnicate", "step 0", "step -3", "step x", "speed", "speed 0",
		"speed 999999", "run 5", "reset now"}
	for _, line := range bad {
		if _, err := parseShell(line); err == nil {
			t.Errorf("%q: want an error", line)
		}
	}
}

// TestShellTypingPath drives the input exactly as the terminal decoder would
// deliver it: ':' focuses the shell, printable runes (including space, which
// v2 carries as KeySpace with Text) build the line, enter executes it.
func TestShellTypingPath(t *testing.T) {
	m := newTestModel(t, `
        add_number r1, r0, 1
        add_number r1, r1, 1
        halt
`)
	next, _ := m.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	m = next.(Model)
	if !m.input.Focused() {
		t.Fatal("':' should focus the shell")
	}
	for _, r := range "step 2" {
		msg := tea.KeyPressMsg{Code: r, Text: string(r)}
		if r == ' ' {
			msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
		}
		next, _ = m.Update(msg)
		m = next.(Model)
	}
	if got := m.input.Value(); got != "step 2" {
		t.Fatalf("input value = %q, want %q (space must survive v2 key encoding)", got, "step 2")
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(Model)
	if m.board.Major != 2 {
		t.Fatalf("shell 'step 2' advanced to major %d, want 2", m.board.Major)
	}
	if m.input.Value() != "" {
		t.Fatalf("input should clear after enter, has %q", m.input.Value())
	}
}

// TestEscUnwindsHelpAndShell: esc closes the help overlay, and esc in the
// shell returns focus to the panes — one level per press.
func TestEscUnwindsHelpAndShell(t *testing.T) {
	m := newTestModel(t, "halt\n")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 145, Height: 30})
	m = next.(Model)

	esc := func(m Model) Model {
		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		return next.(Model)
	}

	m = key(m, "?")
	if !m.help {
		t.Fatal("'?' should open help")
	}
	m = esc(m)
	if m.help {
		t.Fatal("esc should close the help overlay")
	}

	next, _ = m.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	m = next.(Model)
	if !m.input.Focused() {
		t.Fatal("':' should focus the shell")
	}
	m = esc(m)
	if m.input.Focused() {
		t.Fatal("esc should return focus from the shell to the panes")
	}

	// esc with nothing open is a harmless no-op.
	m = esc(m)
	if m.help || m.input.Focused() {
		t.Fatal("esc with nothing open should change nothing")
	}
}

func TestStepAdvancesAndHalts(t *testing.T) {
	m := newTestModel(t, `
        add_number r1, r0, 1
        halt
`)
	// A paused machine does nothing on frames.
	next, _ := m.Update(frameMsg{})
	m = next.(Model)
	if m.board.Major != 0 {
		t.Fatalf("paused frame advanced to major %d", m.board.Major)
	}
	next, _ = m.exec("step 1")
	m = next.(Model)
	if m.board.Major != 1 {
		t.Fatalf("step: major = %d, want 1", m.board.Major)
	}
	next, _ = m.exec("step 100")
	m = next.(Model)
	if !m.board.Halted || m.board.StopReason() != machine.StopHalt {
		t.Fatalf("want clean halt, got %v (fault %v)", m.board.StopReason(), m.board.Fault())
	}
	if m.proc.Regs[1] != 1 {
		t.Fatalf("r1 = %d, want 1", m.proc.Regs[1])
	}
}

func TestRunLoopStopsAtHalt(t *testing.T) {
	m := newTestModel(t, `
        halt
`)
	next, _ := m.exec("run")
	m = next.(Model)
	if !m.running {
		t.Fatal("run should set running")
	}
	for i := 0; i < 5 && m.running; i++ {
		next, _ = m.Update(frameMsg{})
		m = next.(Model)
	}
	if m.running {
		t.Fatal("frame loop must stop itself when the machine halts")
	}
	if m.board.StopReason() != machine.StopHalt {
		t.Fatalf("stop reason = %v, want halt", m.board.StopReason())
	}
	if len(m.shellOut) == 0 || !strings.Contains(m.shellOut[len(m.shellOut)-1], "clean halt") {
		t.Fatalf("shell should report the stop, got %v", m.shellOut)
	}
}

func TestResetRebuildsMachineAndLog(t *testing.T) {
	m := newTestModel(t, `
        halt
`)
	next, _ := m.exec("step 10")
	m = next.(Model)
	if m.board.Major == 0 || m.events.Total() == 0 {
		t.Fatal("precondition: machine ran")
	}
	next, _ = m.exec("reset")
	m = next.(Model)
	if m.board.Major != 0 || m.board.Halted || m.events.Total() != 0 {
		t.Fatalf("reset: major=%d halted=%v events=%d", m.board.Major, m.board.Halted, m.events.Total())
	}
}

func TestViewRendersPanes(t *testing.T) {
	m := newTestModel(t, `
        add_number r1, r0, 7
        add_number r1, r1, 1
        add_number r1, r1, 1
        halt
`)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 145, Height: 30})
	m = next.(Model)
	v := m.View()
	for _, want := range []string{"program", "machine", "numbers", "add_number r1, r0, 7",
		"r0 00000000", "memory · f follows the pc", "tick log · f follows newest", "not started"} {
		if !strings.Contains(v.Content, want) {
			t.Errorf("view missing %q", want)
		}
	}

	next, _ = m.exec("step 3")
	m = next.(Model)
	if v := m.View(); !strings.Contains(v.Content, "major 3") {
		t.Error("view should show major 3")
	}

	// Too small: a clear refusal, not a broken layout.
	next, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = next.(Model)
	if v := m.View(); !strings.Contains(v.Content, "terminal too small") {
		t.Error("small terminal should show the resize notice")
	}
}

func TestViewShowsFaultState(t *testing.T) {
	m := newTestModelWords(t, []uint32{0xffffffff})
	next, _ := m.exec("step 1")
	m = next.(Model)
	if m.board.StopReason() != machine.StopFault {
		t.Fatalf("stop reason = %v, want fault", m.board.StopReason())
	}
	next, _ = m.Update(tea.WindowSizeMsg{Width: 145, Height: 30})
	m = next.(Model)
	if v := m.View(); !strings.Contains(v.Content, "unknown_opcode") {
		t.Error("view should surface the fault kind")
	}
	// Commands against a stopped machine refuse politely.
	next, _ = m.exec("run")
	m = next.(Model)
	if m.running {
		t.Error("run against a halted machine must not start")
	}
}

// longProgram builds n distinguishable instructions plus halt.
func longProgram(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, "add_number r1, r0, %d\n", i)
	}
	sb.WriteString("halt\n")
	return sb.String()
}

func key(m Model, s string) Model {
	next, _ := m.Update(tea.KeyPressMsg{Code: rune(s[0]), Text: s})
	if k := next.(Model); true {
		return k
	}
	return m
}

func TestScrollKeysAndFollow(t *testing.T) {
	m := newTestModel(t, longProgram(60))
	next, _ := m.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	m = next.(Model)

	// Follow (default): the window sits at the top around pc = 0.
	if v := m.View(); !strings.Contains(v.Content, "add_number r1, r0, 0") {
		t.Error("follow should show the first instructions")
	}

	// G: bottom — the halt is visible, the first instruction is not.
	m = key(m, "G")
	v := m.View()
	if !strings.Contains(v.Content, "halt") || strings.Contains(v.Content, "add_number r1, r0, 0") {
		t.Errorf("G should jump to the bottom:\n%s", v.Content)
	}

	// g: top again.
	m = key(m, "g")
	if v := m.View(); !strings.Contains(v.Content, "add_number r1, r0, 0") {
		t.Error("g should jump back to the top")
	}

	// Manual scroll detaches: the window moves one line, then stays put
	// while the pc advances.
	m = key(m, "j")
	v = m.View()
	if !strings.Contains(v.Content, "add_number r1, r0, 1") || strings.Contains(v.Content, "add_number r1, r0, 0") {
		t.Errorf("j should scroll one line:\n%s", v.Content)
	}
	next, _ = m.exec("step 25")
	m = next.(Model)
	v = m.View()
	if !strings.Contains(v.Content, "add_number r1, r0, 1") || strings.Contains(v.Content, "add_number r1, r0, 25") {
		t.Errorf("after manual scroll the window must not follow the pc:\n%s", v.Content)
	}

	// f: follow again — the window re-centres on the pc (word 25).
	m = key(m, "f")
	if v := m.View(); !strings.Contains(v.Content, "add_number r1, r0, 25") {
		t.Errorf("f should re-follow the pc:\n%s", v.Content)
	}
}

// TestPaneHeightsEqualAndFillFrame locks the layout contract: exactly 1 line
// on top, a 6-line bottom block (boxed shell + keyhints), and three panes of
// identical height between — every pane's border corners share the same rows.
func TestPaneHeightsEqualAndFillFrame(t *testing.T) {
	m := newTestModel(t, longProgram(40))
	next, _ := m.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	m = next.(Model)
	rows := strings.Split(m.View().Content, "\n")
	if len(rows) != 30 {
		t.Fatalf("frame is %d rows, want exactly 30 (1 status + 23 panes + 6 bottom)", len(rows))
	}
	for _, c := range []struct {
		row   int
		glyph string
	}{
		{1, "┌"}, {23, "└"},
	} {
		if n := strings.Count(rows[c.row], c.glyph); n != 3 {
			t.Errorf("row %d has %d %q corners, want 3 (panes must share top and bottom rows)", c.row, n, c.glyph)
		}
	}
	// The shell box sits right under the panes; the keyhints close the frame.
	if n := strings.Count(rows[24], "┌"); n != 1 {
		t.Errorf("row 24 should open the shell box, found %d corners", n)
	}
	if !strings.Contains(rows[29], "q quit") {
		t.Errorf("bottom line should carry the keyhints:\n%s", rows[29])
	}
	if strings.Contains(rows[0], "tab focus") {
		t.Error("top bar must not carry keyhints anymore")
	}
}

func TestLogGroupsByMajorTick(t *testing.T) {
	m := newTestModel(t, longProgram(4))
	next, _ := m.exec("step 3")
	m = next.(Model)
	all := m.logAll()
	var blanks int
	for _, l := range all {
		if l.text == "" {
			blanks++
		}
	}
	// 3 majors: at least 2 separators between groups.
	if blanks < 2 {
		t.Errorf("want ≥2 blank separators between major ticks, got %d in %d lines", blanks, len(all))
	}
}
