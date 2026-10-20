package tui

// machine.go — the machine pane: the CPU view fixed in the header, the whole
// RAM scrollable in the body. The row holding the tracked address is
// highlighted; follow pins the pc row a third of the window from the top.

import (
	"encoding/binary"
	"fmt"
	"strings"

	"ammaboard/internal/machine/cpu"
	"ammaboard/internal/tuikit"
)

type machinePane struct {
	scroll tuikit.Scroll
}

func newMachinePane() *machinePane { return &machinePane{scroll: tuikit.NewScroll()} }

func (p *machinePane) keys() tuikit.Keymap { return tuikit.ScrollKeys(&p.scroll) }

func (p *machinePane) reset() { p.scroll.Reset() }

func (p *machinePane) region(m *Model, _, h int) tuikit.Region {
	header := m.machineHeader()
	return tuikit.Region{
		Title:       "machine",
		Header:      header,
		Body:        p.memory(m, h-3-len(header)),
		Constraints: tuikit.Constraints{MinW: 43, MinH: 15},
	}
}

// machineHeader is the fixed CPU view: registers, pc, status, memory label.
func (m *Model) machineHeader() []tuikit.Line {
	var out []tuikit.Line
	for i := 0; i < 8; i += 2 {
		out = append(out, tuikit.Plain(fmt.Sprintf("r%d %08x   r%d %08x",
			i, m.proc.Regs[i], i+1, m.proc.Regs[i+1])))
	}
	out = append(out, tuikit.Plain(fmt.Sprintf("pc %04x (next fetch)", m.proc.PC)))
	ie := 0
	if m.proc.Status&cpu.StatusIE != 0 {
		ie = 1
	}
	out = append(out, tuikit.Plain(fmt.Sprintf("ie %d · status %08x", ie, m.proc.Status)))
	return append(out, tuikit.Line{}, tuikit.Styled("memory · f follows the pc", stDim))
}

// memory: the whole RAM as scrollable rows of 4 words; the row holding the
// cursor address is highlighted.
func (p *machinePane) memory(m *Model, bh int) []tuikit.Line {
	if bh <= 0 {
		return nil
	}
	dump := m.mem.Dump()
	rows := len(dump) / 16
	cur := m.cursorAddr()
	top := p.scroll.Window(int(cur/16), rows, bh)
	out := make([]tuikit.Line, 0, bh)
	for r := top; r < rows && len(out) < bh; r++ {
		addr := uint32(16 * r)
		cells := make([]string, 4)
		for c := 0; c < 4; c++ {
			cells[c] = fmt.Sprintf("%08x", binary.LittleEndian.Uint32(dump[addr+uint32(4*c):]))
		}
		text := fmt.Sprintf("%04x  %s", addr, strings.Join(cells, " "))
		if cur >= addr && cur < addr+16 {
			out = append(out, tuikit.Styled(text, stCursor))
		} else {
			out = append(out, tuikit.Plain(text))
		}
	}
	return out
}
