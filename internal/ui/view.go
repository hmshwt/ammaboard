package ui

// view.go — rendering. Numbers and grids, never animation: every pane is a
// pure function of machine state + the event tail.
//
// Frame budget: 1 status line on top, 3 shell lines at the bottom, and the
// three panes exactly fill everything between (equal heights, enforced by
// padding/clipping every pane to the same row count). Each pane = fixed
// header + scrollable body (docs/ui.md): the machine pane keeps the CPU view
// fixed and scrolls memory; the numbers pane keeps stats fixed and scrolls
// the tick log (one blank line between major ticks). The tracked instruction
// is white on dark green — a highlighted line, not a pointer character.

import (
	"encoding/binary"
	"fmt"
	"strings"

	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
	"ammaboard/internal/machine/cpu"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	minWidth  = 141
	minHeight = 26

	leftW  = 44
	rightW = 54 // tick log gets the wider column; the machine pane gives up 10
)

var (
	cAccent  = lipgloss.Color("#4FA6FF")
	cDim     = lipgloss.Color("#6a737d")
	cRed     = lipgloss.Color("#f14c4c")
	cGreen   = lipgloss.Color("#3fb950")
	cCursor  = lipgloss.Color("#006400") // dark green
	cWhite   = lipgloss.Color("#ffffff")
	cBar     = lipgloss.Color("#1c2128")
	cBarText = lipgloss.Color("#e6edf3")

	stTop    = lipgloss.NewStyle().Background(cBar).Foreground(cBarText)
	stTitle  = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	stDim    = lipgloss.NewStyle().Foreground(cDim)
	stFault  = lipgloss.NewStyle().Foreground(cRed)
	stHalt   = lipgloss.NewStyle().Foreground(cGreen)
	stCursor = lipgloss.NewStyle().Foreground(cWhite).Background(cCursor)
	stHelp   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(cAccent).Padding(1, 2)
)

// line is one rendered row: plain text (never wider than its pane) plus an
// optional style. Padding happens at render time so backgrounds span the pane.
type line struct {
	text string
	st   *lipgloss.Style
}

func plain(s string) line { return line{text: s} }

func styled(s string, st lipgloss.Style) line { return line{text: s, st: &st} }

func (l line) render(w int) string {
	s := padRight(l.text, w)
	if l.st == nil {
		return s
	}
	return l.st.Render(s)
}

// View renders the frame: the too-small notice, the help modal, or the body.
func (m Model) View() tea.View {
	var body string
	switch {
	case m.width < minWidth || m.height < minHeight:
		body = fmt.Sprintf("terminal too small: need ≥ %d×%d, have %d×%d — resize me",
			minWidth, minHeight, m.width, m.height)
	case m.help:
		body = m.renderHelp()
	default:
		body = m.renderBody()
	}
	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

// renderBody: status bar / three equal panes / 3-line shell.
func (m Model) renderBody() string {
	inner := m.height - 6 // minus top bar (1), shell (3), pane borders (2)

	left := m.renderPane(0, leftW, inner, plain("program"),
		nil, m.programVisible(inner-1, m.scroll[0]))
	center := m.renderPane(1, m.width-leftW-rightW, inner, plain("machine"),
		m.machineHeader(), m.memoryVisible(inner-9, m.scroll[1])) // header is 8 lines
	right := m.renderPane(2, rightW, inner, plain("numbers"),
		m.numbersHeader(), m.logVisible(inner-7, m.scroll[2])) // header is 6 lines

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
	return lipgloss.JoinVertical(lipgloss.Left, m.topBar(), panes, m.shellBlock())
}

// renderPane boxes one pane: title + fixed header + scroll window, padded or
// clipped to exactly inner rows — the equal-height guarantee is structural.
func (m Model) renderPane(focusIdx, width, inner int, title line, header, body []line) string {
	st := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(cDim)
	if m.focus == focusIdx && !m.help {
		st = st.BorderForeground(cAccent)
	}
	cw := width - 2
	rows := make([]line, 0, inner+2)
	rows = append(rows, title)
	rows = append(rows, header...)
	rows = append(rows, body...)
	for len(rows) < inner {
		rows = append(rows, line{})
	}
	if len(rows) > inner {
		rows = rows[:inner]
	}
	out := make([]string, len(rows))
	for i, l := range rows {
		out[i] = l.render(cw)
	}
	return st.Render(strings.Join(out, "\n"))
}

// cursorAddr is the address the panes track: the next fetch while running
// (the pc convention), backing up to the halted instruction once stopped so
// the highlight sits on the word that stopped the machine.
func (m Model) cursorAddr() uint32 {
	cur := m.proc.PC
	if m.board.Halted && cur >= 4 {
		cur -= 4
	}
	return cur
}

// windowTop resolves a scroll window: following pins the tracked line a third
// from the top; manual scroll clamps to the content.
func windowTop(sc scroller, track, max, bh int) int {
	if max <= bh {
		return 0
	}
	if sc.follow {
		return clampInt(track-bh/3, 0, max-bh)
	}
	return clampInt(sc.top, 0, max-bh)
}

// programVisible: disassembly, one word per line, cursor line highlighted.
func (m Model) programVisible(bh int, sc scroller) []line {
	if bh <= 0 {
		return nil
	}
	cur := m.cursorAddr()
	top := windowTop(sc, int(cur/4), len(m.words), bh)
	out := make([]line, 0, bh)
	for i := top; i < len(m.words) && len(out) < bh; i++ {
		addr := uint32(4 * i)
		text := fmt.Sprintf("%04x  %s", addr, truncate(isa.Disassemble(m.words[i]), 34))
		if addr == cur {
			out = append(out, styled(text, stCursor))
		} else {
			out = append(out, plain(text))
		}
	}
	if m.board.StopReason() == machine.StopHalt {
		out = append(out, styled("— clean halt —", stHalt))
	}
	return out
}

// machineHeader is the fixed CPU view: registers, pc, status, memory label.
func (m Model) machineHeader() []line {
	var out []line
	for i := 0; i < 8; i += 2 {
		out = append(out, plain(fmt.Sprintf("r%d %08x   r%d %08x",
			i, m.proc.Regs[i], i+1, m.proc.Regs[i+1])))
	}
	out = append(out, plain(fmt.Sprintf("pc %04x (next fetch)", m.proc.PC)))
	ie := 0
	if m.proc.Status&cpu.StatusIE != 0 {
		ie = 1
	}
	out = append(out, plain(fmt.Sprintf("ie %d · status %08x", ie, m.proc.Status)))
	return append(out, line{}, styled("memory · f follows the pc", stDim))
}

// memoryVisible: the whole RAM as scrollable rows of 4 words; the row holding
// the cursor address is highlighted.
func (m Model) memoryVisible(bh int, sc scroller) []line {
	if bh <= 0 {
		return nil
	}
	dump := m.mem.Dump()
	rows := len(dump) / 16
	cur := m.cursorAddr()
	top := windowTop(sc, int(cur/16), rows, bh)
	out := make([]line, 0, bh)
	for r := top; r < rows && len(out) < bh; r++ {
		addr := uint32(16 * r)
		cells := make([]string, 4)
		for c := 0; c < 4; c++ {
			cells[c] = fmt.Sprintf("%08x", binary.LittleEndian.Uint32(dump[addr+uint32(4*c):]))
		}
		text := fmt.Sprintf("%04x  %s", addr, strings.Join(cells, " "))
		if cur >= addr && cur < addr+16 {
			out = append(out, styled(text, stCursor))
		} else {
			out = append(out, plain(text))
		}
	}
	return out
}

// numbersHeader is the fixed stats block — derived from the log, never
// hand-counted.
func (m Model) numbersHeader() []line {
	out := []line{
		plain(fmt.Sprintf("major   %d", m.board.Major)),
		plain(fmt.Sprintf("events  %d", m.events.Total())),
		plain(fmt.Sprintf("speed   ×%d/frame", m.speed)),
	}
	switch m.board.StopReason() {
	case machine.StopHalt:
		out = append(out, styled("state   clean halt", stHalt))
	case machine.StopFault:
		f := m.board.Fault()
		out = append(out, styled(fmt.Sprintf("state   %s pc=%04x addr=%04x", f.Kind, f.PC, f.Addr), stFault))
	default:
		out = append(out, plain(fmt.Sprintf("state   %s", m.runState())))
	}
	return append(out, line{}, styled("tick log · f follows newest", stDim))
}

// logVisible: the event tail grouped by major tick — one blank line between
// ticks — oldest to newest. follow pins the bottom.
func (m Model) logVisible(bh int, sc scroller) []line {
	if bh <= 0 {
		return nil
	}
	all := m.logAll()
	top := max(0, len(all)-bh)
	if !sc.follow {
		top = clampInt(sc.top, 0, top)
	}
	end := min(len(all), top+bh)
	return all[top:end]
}

// logAll formats the recent event tail with blank separators per major tick.
func (m Model) logAll() []line {
	var out []line
	last := -1
	for _, e := range m.events.Tail(256) {
		if e.Major != last {
			if len(out) > 0 {
				out = append(out, line{})
			}
			last = e.Major
		}
		text := fmt.Sprintf("%3d %-5s %s", e.Major, e.Component, truncate(e.Kind+" "+e.Detail, 42))
		if e.Kind == "fault" {
			out = append(out, styled(text, stFault))
		} else {
			out = append(out, plain(text))
		}
	}
	return out
}

// topBar carries status on the left and keyhints on the right — there is no
// separate keyhint row (the shell owns the bottom 3 lines).
func (m Model) topBar() string {
	left := fmt.Sprintf(" AmmaBoard-2 · %s · major %d · %s · ×%d ",
		m.name, m.board.Major, m.runState(), m.speed)
	keys := " tab focus · space run/pause · s step · j/k scroll · f follow · : shell · ? help · q quit "
	avail := m.width - len(left) - len(keys)
	if avail < 0 {
		keys = truncate(keys, max(0, m.width-len(left)))
		avail = 0
	}
	return stTop.Width(m.width).Render(left + strings.Repeat(" ", avail) + keys)
}

// shellBlock is exactly 3 lines: the prompt plus the two most recent replies.
func (m Model) shellBlock() string {
	lines := []string{m.input.View()}
	for _, o := range m.shellOut[max(0, len(m.shellOut)-2):] {
		lines = append(lines, stDim.Render("  "+o))
	}
	for len(lines) < 3 {
		lines = append(lines, "")
	}
	return strings.Join(lines[:3], "\n")
}

// renderHelp replaces the panes with the help modal (same frame, other body).
func (m Model) renderHelp() string {
	help := `AmmaBoard TUI

  tab / shift+tab   cycle pane focus
  space             run / pause
  s                 step one major tick
  + / -             speed ×2 / ÷2
  j/k or up/down    scroll the focused pane's body
  g / G             jump to top / bottom
  f                 re-follow (pc / newest events)
  :                 focus the shell (esc leaves)
  ?                 toggle this help (esc closes)
  q                 quit

  shell: run · step [n] · reset · speed n · quit · help`
	box := lipgloss.NewStyle().Width(m.width-4).Height(m.height-6).
		AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Center).
		Padding(0, 2).Render(stHelp.Render(help))

	return lipgloss.JoinVertical(lipgloss.Left, m.topBar(), box, m.shellBlock())
}

// padRight pads a plain string to exactly w columns so pane borders align and
// cursor backgrounds span the full width.
func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// truncate caps a plain string to n characters with an ellipsis.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n > 1 {
		return s[:n-1] + "…"
	}
	return s[:n]
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// runState names the run loop: running, paused, or the stop verdict.
func (m Model) runState() string {
	if m.board.Halted {
		return m.stateLine()
	}
	if m.running {
		return "running"
	}
	if m.board.Major == 0 {
		return "not started"
	}
	return "paused"
}
