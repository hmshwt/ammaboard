package tui

// program.go — the program pane: disassembly, one word per line, the tracked
// instruction white on dark green — a highlighted line, not a pointer. A pure
// view over the words plus the pc; the scroll state is the pane's own.

import (
	"fmt"

	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
	"ammaboard/internal/tuikit"
)

type programPane struct {
	scroll tuikit.Scroll
}

func newProgramPane() *programPane { return &programPane{scroll: tuikit.NewScroll()} }

func (p *programPane) keys() tuikit.Keymap { return tuikit.ScrollKeys(&p.scroll) }

func (p *programPane) reset() { p.scroll.Reset() }

func (p *programPane) region(m *Model, _, h int) tuikit.Region {
	return tuikit.Region{
		Title:       "program",
		Body:        p.body(m, h-3), // borders + title row
		Constraints: tuikit.Constraints{MinW: 42, MinH: 10},
	}
}

// body: the scroll window over the disassembly, the cursor line highlighted.
func (p *programPane) body(m *Model, bh int) []tuikit.Line {
	if bh <= 0 {
		return nil
	}
	cur := m.cursorAddr()
	top := p.scroll.Window(int(cur/4), len(m.words), bh)
	out := make([]tuikit.Line, 0, bh)
	for i := top; i < len(m.words) && len(out) < bh; i++ {
		addr := uint32(4 * i)
		text := fmt.Sprintf("%04x  %s", addr, tuikit.Truncate(isa.Disassemble(m.words[i]), 34))
		if addr == cur {
			out = append(out, tuikit.Styled(text, stCursor))
		} else {
			out = append(out, tuikit.Plain(text))
		}
	}
	if m.board.StopReason() == machine.StopHalt {
		out = append(out, tuikit.Styled("— clean halt —", stHalt))
	}
	return out
}
