package tui

// numbers.go — the numbers pane: live stats fixed in the header (derived from
// the log, never hand-counted), the tick log scrollable in the body with one
// blank line between major ticks.

import (
	"fmt"

	"ammaboard/internal/machine"
	"ammaboard/internal/tuikit"
)

type numbersPane struct {
	scroll tuikit.Scroll
}

func newNumbersPane() *numbersPane { return &numbersPane{scroll: tuikit.NewScroll()} }

func (p *numbersPane) keys() tuikit.Keymap { return tuikit.ScrollKeys(&p.scroll) }

func (p *numbersPane) reset() { p.scroll.Reset() }

func (p *numbersPane) region(m *Model, _, h int) tuikit.Region {
	header := m.numbersHeader()
	return tuikit.Region{
		Title:       "numbers",
		Header:      header,
		Body:        p.log(m, h-3-len(header)),
		Constraints: tuikit.Constraints{MinW: 54, MinH: 12},
	}
}

// numbersHeader is the fixed stats block.
func (m *Model) numbersHeader() []tuikit.Line {
	out := []tuikit.Line{
		tuikit.Plain(fmt.Sprintf("major   %d", m.board.Major)),
		tuikit.Plain(fmt.Sprintf("events  %d", m.events.Total())),
		tuikit.Plain(fmt.Sprintf("speed   ×%d/frame", m.speed)),
	}
	switch m.board.StopReason() {
	case machine.StopHalt:
		out = append(out, tuikit.Styled("state   clean halt", stHalt))
	case machine.StopFault:
		f := m.board.Fault()
		out = append(out, tuikit.Styled(fmt.Sprintf("state   %s pc=%04x addr=%04x", f.Kind, f.PC, f.Addr), stFault))
	default:
		out = append(out, tuikit.Plain(fmt.Sprintf("state   %s", m.runState())))
	}
	return append(out, tuikit.Line{}, tuikit.Styled("tick log · f follows newest", stDim))
}

// log: the event tail windowed. Follow pins the newest line to the *bottom*
// of the window, so the tracked line passed to Window is the content end —
// clamping then lands on the last page. Manual scrolling clamps to the
// content as usual.
func (p *numbersPane) log(m *Model, bh int) []tuikit.Line {
	if bh <= 0 {
		return nil
	}
	all := m.logAll()
	top := p.scroll.Window(len(all), len(all), bh)
	end := min(len(all), top+bh)
	return all[top:end]
}

// logAll formats the recent event tail with blank separators per major tick,
// oldest to newest; faults render red.
func (m *Model) logAll() []tuikit.Line {
	var out []tuikit.Line
	last := -1
	for _, e := range m.events.Tail(256) {
		if e.Major != last {
			if len(out) > 0 {
				out = append(out, tuikit.Line{})
			}
			last = e.Major
		}
		text := fmt.Sprintf("%3d %-5s %s", e.Major, e.Component, tuikit.Truncate(e.Kind+" "+e.Detail, 42))
		if e.Kind == "fault" {
			out = append(out, tuikit.Styled(text, stFault))
		} else {
			out = append(out, tuikit.Plain(text))
		}
	}
	return out
}
