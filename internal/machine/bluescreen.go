package machine

import (
	"fmt"
	"strings"

	"ammaboard/internal/isa"
)

// BlueScreen is the machine's full "I stopped somewhere I should not have"
// panel: what happened, what it means in plain language, the complete state
// at the moment of the fault, and the cause chain that led here. ui.md §9 —
// a fault is data, and the screen says everything about it. Rendered for any
// halt that is NOT a clean halt; a clean stop needs no blue screen.
func BlueScreen(cpu *CPU, board *Motherboard, ram *RAM, recent []Event) string {
	f := board.HaltedBy()
	var b strings.Builder

	line := strings.Repeat("=", 72)
	b.WriteString(line + "\n")
	b.WriteString(" BLUE SCREEN — AmmaBoard-2 stopped somewhere it should not have\n")
	b.WriteString(line + "\n\n")

	// ---- what happened ----
	b.WriteString(" what happened\n")
	fmt.Fprintf(&b, "   %s fault, cycle %d, phase %s\n", f.Kind, board.Major, f.Phase)
	b.WriteString("   " + faultExplanation(f) + "\n\n")

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
	fmt.Fprintf(&b, "   program counter (next fetch) ........ 0x%04x\n", cpu.PC)
	ie := "enabled"
	if cpu.Status&StatusIE == 0 {
		ie = "disabled"
	}
	fmt.Fprintf(&b, "   interrupts .......................... %s\n", ie)
	for i := 0; i < 8; i += 2 {
		fmt.Fprintf(&b, "   register_%d = %-12d register_%d = %d\n", i, cpu.Regs[i], i+1, cpu.Regs[i+1])
	}
	b.WriteString(memoryWindow(ram, f.Addr))
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
func faultExplanation(f Fault) string {
	switch f.Kind {
	case "unknown_opcode":
		return "The CPU fetched a word whose opcode is not any instruction it knows —" +
			" usually the program counter wandered into data, or memory that was never loaded."
	case "alignment":
		return "A 32-bit access touched an address that is not a multiple of 4. Words live" +
			" on 4-byte boundaries — only byte accesses may sit between them."
	case "bus_NOT_CONNECTED":
		return "The program reached an address that no component owns. The wire leads" +
			" nowhere — usually a wrong pointer or an address past the end of RAM."
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
func memoryWindow(ram *RAM, addr uint32) string {
	if addr == 0 || addr < ram.Base || addr >= ram.Base+ram.Size {
		return ""
	}
	start := (addr - ram.Base) &^ 0xF // 4-word aligned window
	var s strings.Builder
	s.WriteString("   memory around the faulting address:\n")
	for off := start; off+4 <= start+32 && off+4 <= uint32(len(ram.mem)); off += 4 {
		w := uint32(ram.mem[off]) | uint32(ram.mem[off+1])<<8 | uint32(ram.mem[off+2])<<16 | uint32(ram.mem[off+3])<<24
		mark := ""
		if addr >= ram.Base+off && addr < ram.Base+off+4 {
			mark = "  <- faulting address in this word"
		}
		fmt.Fprintf(&s, "     0x%04x: %08x%s\n", ram.Base+off, w, mark)
	}
	return s.String()
}
