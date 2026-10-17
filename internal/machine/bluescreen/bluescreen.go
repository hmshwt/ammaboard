// Package bluescreen renders the machine's full "I stopped somewhere I should
// not have" panel (ui.md §9): what happened, what it means in plain language,
// the complete state at the moment of the fault, and the cause chain that led
// here. A fault is data, and the screen says everything about it. Rendered for
// any halt that is NOT a clean halt; a clean stop needs no blue screen.
package bluescreen

import (
	"encoding/binary"
	"fmt"
	"strings"

	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
	"ammaboard/internal/machine/cpu"
	"ammaboard/internal/machine/ram"
)

// Render builds the panel for the board's terminal fault. recent is the event
// stream so far, used for the cause chain.
func Render(c *cpu.CPU, board *machine.Motherboard, r *ram.RAM, recent []machine.Event) string {
	f := board.Fault()
	if f == nil {
		return "" // nothing faulted: a blue screen would be a lie
	}
	var b strings.Builder

	line := strings.Repeat("=", 72)
	b.WriteString(line + "\n")
	fmt.Fprintf(&b, " BLUE SCREEN — %s stopped somewhere it should not have\n", machine.MachineVersion)
	b.WriteString(line + "\n\n")

	// ---- what happened ----
	b.WriteString(" what happened\n")
	fmt.Fprintf(&b, "   %s fault, cycle %d, phase %s\n", f.Kind, board.Major, f.Phase)
	b.WriteString("   " + faultExplanation(*f) + "\n\n")

	// ---- the instruction that caused it ----
	b.WriteString(" the instruction that caused it\n")
	fmt.Fprintf(&b, "   program counter (this instruction) ... 0x%04x\n", f.PC)
	if in, err := isa.Decode(f.Word); err == nil {
		fmt.Fprintf(&b, "   instruction ......................... %s rd=r%d rs=r%d rt=r%d imm=%d\n",
			in.Op.Name(), in.Rd, in.Rs, in.Rt, in.Imm)
		fmt.Fprintf(&b, "   machine word ........................ 0x%08x\n", f.Word)
	} else if f.Word != 0 {
		fmt.Fprintf(&b, "   machine word ........................ 0x%08x — not a known instruction\n", f.Word)
		fmt.Fprintf(&b, "   opcode field (bits 31..27) .......... %d of 1..20\n", f.Word>>27)
	}
	if f.Addr != 0 {
		fmt.Fprintf(&b, "   faulting address .................... 0x%04x\n", f.Addr)
	}
	b.WriteString("\n")

	// ---- the machine at the moment of the fault ----
	b.WriteString(" the machine at the moment of the fault\n")
	fmt.Fprintf(&b, "   program counter (next fetch) ........ 0x%04x\n", c.PC)
	ie := "enabled"
	if c.Status&cpu.StatusIE == 0 {
		ie = "disabled"
	}
	fmt.Fprintf(&b, "   interrupts .......................... %s\n", ie)
	for i := 0; i < 8; i += 2 {
		fmt.Fprintf(&b, "   register_%d = %-12d register_%d = %d\n", i, c.Regs[i], i+1, c.Regs[i+1])
	}
	b.WriteString(memoryWindow(r, f.Addr))
	b.WriteString("   address windows on this board:\n")
	b.WriteString(board.DescribeWindows())
	b.WriteString("\n")

	// ---- how it got here ----
	b.WriteString(" how it got here (last events before the stop)\n")
	n := len(recent)
	if n > 8 {
		n = 8
	}
	for _, e := range recent[len(recent)-n:] {
		marker := "   "
		if e.Kind == "fault" {
			marker = " >>"
		}
		fmt.Fprintf(&b, " %s%6d %-8s %-6s %-8s %s\n", marker, e.Major, e.Phase, e.Component, e.Kind, e.Detail)
	}
	b.WriteString("\n   the full trail is in trace.txt — every cycle, one explained line each;\n")
	b.WriteString("   find the cycle number above and read what led up to it.\n")
	b.WriteString(line + "\n")
	return b.String()
}

// faultExplanation is the kernel's plain-language "what this usually means".
func faultExplanation(f machine.Fault) string {
	switch f.Kind {
	case "unknown_opcode":
		return "The CPU fetched a word whose opcode is not any instruction it knows —" +
			" usually the program counter wandered into data, or memory that was never loaded."
	case "alignment":
		return "A 32-bit access touched an address that is not a multiple of 4. Words live" +
			" on 4-byte boundaries — only byte accesses may sit between them."
	case "bus_NOT_CONNECTED":
		return "The program reached an address that no component owns. The wire leads" +
			" nowhere — usually a wrong pointer, an address past the end of RAM, or a store" +
			" posted to a gap between devices."
	case "bus_NOT_PERMITTED":
		return "The access reached a component that refuses it: an unaligned word on RAM," +
			" or a byte access into a word-only device."
	case "stray_rfi":
		return "return_from_interrupt ran with no interrupt in flight — the handler was" +
			" entered without a taken interrupt, or returned twice."
	default:
		return fmt.Sprintf("The machine refused to continue (%s).", f.Kind)
	}
}

// memoryWindow renders the words around the faulting address, if it falls in
// RAM. The faulting word is marked — visibility down to the byte.
func memoryWindow(r *ram.RAM, addr uint32) string {
	if addr == 0 || addr < r.Base || addr >= r.Base+r.Size {
		return ""
	}
	mem := r.Dump()
	start := (addr - r.Base) &^ 0xF // 4-word aligned window
	var s strings.Builder
	s.WriteString("   memory around the faulting address:\n")
	for off := start; off+4 <= start+32 && off+4 <= uint32(len(mem)); off += 4 {
		w := binary.LittleEndian.Uint32(mem[off:])
		mark := ""
		if addr >= r.Base+off && addr < r.Base+off+4 {
			mark = "  <- faulting address in this word"
		}
		fmt.Fprintf(&s, "     0x%04x: %08x%s\n", r.Base+off, w, mark)
	}
	return s.String()
}
