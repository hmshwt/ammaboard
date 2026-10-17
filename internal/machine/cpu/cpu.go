// Package cpu implements the AmmaBoard-2 processor: 8 registers (r0
// hardwired zero), a pc, a status register (IE, dormant PRIV), and the
// stage-1 + stage-2 instruction sets.
//
// One file per instruction, registered by opcode in init() — adding an
// instruction means adding a file + its tests and never touching the execute
// loop (motherboard.md §8). Fault plumbing lives in the struct from day one,
// dormant until day 10.
package cpu

import (
	"fmt"

	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

// Status register bits. PRIV is stage-3 hardware: present in the encoding,
// dormant until the base+limit day (poc.md §1).
const (
	StatusIE   uint32 = 1 << 0 // interrupts enabled
	StatusPriv uint32 = 1 << 1 // kernel mode (dormant)
)

// CPU is the AmmaBoard-2 processor.
type CPU struct {
	Regs   [8]uint32
	PC     uint32 // invariant: always the address of the NEXT fetch
	Halted bool
	Status uint32

	faulted   *machine.Fault // terminal fault record; nil on a clean halt
	stopPC    uint32         // address of the halt instruction that stopped us
	fetchWord uint32
	decoded   *isa.Instruction

	// Interrupt shadow state: where a taken interrupt parks the machine
	// until return_from_interrupt. No stack, no memory traffic — the
	// interrupted context is two words of hardware state.
	intPC     uint32
	intStatus uint32
	inHandler bool
}

// New creates a CPU whose first fetch is at pc.
func New(pc uint32) *CPU { return &CPU{PC: pc} }

func (c *CPU) Name() string { return "cpu" }

func (c *CPU) Tick(phase machine.Phase, env machine.Env) {
	if c.Halted {
		return
	}
	switch phase {
	case machine.PhaseFetch:
		c.fetch(env)
	case machine.PhaseDecode:
		c.decode(env)
	case machine.PhaseExecute:
		c.execute(env)
	}
}

func (c *CPU) fetch(env machine.Env) {
	env.Logf("fetch", "pc=0x%04x", c.PC)
	w, st := env.ReadBus(c.PC)
	if st != machine.BusOK {
		env.Logf("fault", "fetch from 0x%04x: %s", c.PC, st)
		c.stop(env, machine.Fault{Kind: "bus_" + string(st), Addr: c.PC, PC: c.PC, Phase: machine.PhaseFetch})
		return
	}
	c.fetchWord = w
	c.PC += 4 // fetch advances pc; branches/calls are relative to the next word
}

func (c *CPU) decode(env machine.Env) {
	in, err := isa.Decode(c.fetchWord)
	if err != nil {
		// Reserved-bit / unknown-opcode: the machine refuses nonsense.
		env.Logf("fault", "%v (word 0x%08x)", err, c.fetchWord)
		c.stop(env, machine.Fault{Kind: "unknown_opcode", PC: c.PC - 4, Phase: machine.PhaseDecode, Word: c.fetchWord})
		return
	}
	c.decoded = &in
	env.Logf("decode", "%s rd=r%d rs=r%d rt=r%d imm=%d", in.Op.Name(), in.Rd, in.Rs, in.Rt, in.Imm)
}

// TakeInterrupt delivers a raised line at the board's latch edge: the
// interrupted pc and status park in shadow registers, IE clears (interrupts
// are not nested — the handler runs with them forbidden), and the pc aims at
// the trap vector. Refused while IE is clear; the line stays raised and the
// board retries at the next edge. toStatus is the post-take status, so the
// broker can log the transition without knowing the bit's meaning.
func (c *CPU) TakeInterrupt(vector uint32) (taken bool, fromPC, fromStatus, toStatus uint32) {
	if c.Halted || c.Status&StatusIE == 0 {
		return false, 0, 0, 0
	}
	c.intPC, c.intStatus = c.PC, c.Status
	c.Status &^= StatusIE
	c.PC = vector
	c.inHandler = true
	return true, c.intPC, c.intStatus, c.Status
}

// stop records a terminal fault and halts the CPU. Clean halts do not come
// through here — they call env.CleanHalt (see halt.go). The board keeps the
// first fault, so both paths stay honest under any ordering.
func (c *CPU) stop(env machine.Env, f machine.Fault) {
	c.Halted = true
	c.faulted = &f
	env.Halt(f)
}

// at returns the address of the instruction being executed (pc is one word
// ahead — the next-fetch invariant).
func (c *CPU) at() uint32 { return c.PC - 4 }

func (c *CPU) writeReg(n uint8, v uint32) {
	if n == 0 {
		return // r0 is hardwired zero: the machine's one free constant
	}
	c.Regs[n] = v
}

// Fault returns the terminal fault record, or nil if the CPU halted cleanly
// (or is still running). Budget exhaustion is the board's concept — a CPU that
// is not Halted simply has no record.
func (c *CPU) Fault() *machine.Fault { return c.faulted }

// StateDump renders registers, pc and status (part of the freeze snapshot).
func (c *CPU) StateDump() string {
	s := fmt.Sprintf("pc=0x%04x (next fetch)\n", c.PC)
	ie := "off"
	if c.Status&StatusIE != 0 {
		ie = "on"
	}
	s += fmt.Sprintf("status ie=%s priv=%s", ie, privName(c.Status))
	if c.inHandler {
		s += " in_handler=yes"
	}
	s += "\n"
	for i, v := range c.Regs {
		s += fmt.Sprintf("r%d=%-11d 0x%08x", i, v, v)
		if i%2 == 1 {
			s += "\n"
		} else {
			s += "  "
		}
	}
	if c.faulted != nil {
		s += fmt.Sprintf("fault: %s at pc=0x%04x phase=%s addr=0x%04x\n",
			c.faulted.Kind, c.faulted.PC, c.faulted.Phase, c.faulted.Addr)
	} else if c.Halted {
		s += fmt.Sprintf("halt: clean stop at pc=0x%04x phase=%s\n", c.stopPC, machine.PhaseExecute)
	}
	return s
}

func privName(status uint32) string {
	if status&StatusPriv != 0 {
		return "kernel"
	}
	return "user"
}
