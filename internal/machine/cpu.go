package machine

import (
	"fmt"

	"ammaboard/internal/isa"
)

// Status register bits. PRIV is stage-3 hardware: present in the encoding,
// dormant until base+limit day (poc.md §1).
const (
	StatusIE   uint32 = 1 << 0 // interrupts enabled
	StatusPriv uint32 = 1 << 1 // kernel mode (dormant)
)

// TrapVector is where the board aims the pc when a pending interrupt line is
// delivered. Fixed for AmmaBoard-2; vectored controllers arrive with a second
// interrupt source.
const TrapVector uint32 = 0x0008

// CPU is the AmmaBoard-2 processor: 8 registers (r0 hardwired zero), pc, a
// status register (IE, dormant PRIV), and the stage-1 + stage-2 instruction
// sets. Fault plumbing lives in the struct from day one — dormant until day
// 10, per poc.md.
type CPU struct {
	Regs   [8]uint32
	PC     uint32 // invariant: always the address of the NEXT fetch
	Halted bool
	Status uint32

	haltedBy  *Fault
	fetchWord uint32
	decoded   *isa.Instruction

	// Interrupt shadow state: where a taken interrupt parks the machine
	// until return_from_interrupt. No stack, no memory traffic — the
	// interrupted context is two words of hardware state.
	intPC     uint32
	intStatus uint32
	inHandler bool
}

func NewCPU(pc uint32) *CPU { return &CPU{PC: pc} }

func (c *CPU) Name() string { return "cpu" }

func (c *CPU) writeReg(n uint8, v uint32) {
	if n == 0 {
		return // r0 is hardwired zero: the machine's one free constant
	}
	c.Regs[n] = v
}

// TakeInterrupt delivers a raised line at the board's latch edge: the
// interrupted pc and status park in shadow registers, IE clears (interrupts
// are not nested — the handler runs with them forbidden), and the pc aims at
// the trap vector. Refused while IE is clear; the line stays raised and the
// board retries at the next edge.
func (c *CPU) TakeInterrupt(vector uint32) (taken bool, fromPC, fromStatus uint32) {
	if c.Halted || c.Status&StatusIE == 0 {
		return false, 0, 0
	}
	c.intPC, c.intStatus = c.PC, c.Status
	c.Status &^= StatusIE
	c.PC = vector
	c.inHandler = true
	return true, c.intPC, c.intStatus
}

func (c *CPU) Tick(phase Phase, env Env) {
	if c.Halted {
		return
	}
	halt := func(f Fault) {
		c.Halted = true
		c.haltedBy = &f
		env.Halt(f)
	}
	switch phase {
	case PhaseFetch:
		env.Logf("fetch", "pc=0x%04x", c.PC)
		w, st := env.ReadBus(c.PC)
		if st != BusOK {
			env.Logf("fault", "fetch from 0x%04x: %s", c.PC, st)
			halt(Fault{Kind: "bus_" + string(st), Addr: c.PC, PC: c.PC, Phase: phase})
			return
		}
		c.fetchWord = w
		c.PC += 4 // fetch advances pc; branches/calls are relative to the next word

	case PhaseDecode:
		in, err := isa.Decode(c.fetchWord)
		if err != nil {
			// Reserved-bit / unknown-opcode: the machine refuses nonsense.
			env.Logf("fault", "%v (word 0x%08x)", err, c.fetchWord)
			halt(Fault{Kind: "unknown_opcode", PC: c.PC - 4, Phase: phase, Word: c.fetchWord})
			return
		}
		c.decoded = &in
		env.Logf("decode", "%s rd=r%d rs=r%d rt=r%d imm=%d", in.Op.Name(), in.Rd, in.Rs, in.Rt, in.Imm)

	case PhaseExecute:
		in := c.decoded
		if in == nil {
			return
		}
		switch in.Op {
		case isa.OpHalt:
			env.Logf("halt", "clean stop at pc=0x%04x (next fetch 0x%04x)", c.PC-4, c.PC)
			halt(Fault{Kind: "halt", PC: c.PC - 4, Phase: phase})

		// ---- ALU: every state change gets a visible EXECUTE line ----
		case isa.OpAdd:
			was := c.Regs[in.Rd]
			v := c.Regs[in.Rs] + c.Regs[in.Rt]
			env.Logf("alu", "r%d <= %d + %d = %d (was %d)", in.Rd, c.Regs[in.Rs], c.Regs[in.Rt], v, was)
			c.writeReg(in.Rd, v)
		case isa.OpAddNumber:
			was := c.Regs[in.Rd]
			v := c.Regs[in.Rs] + uint32(in.Imm)
			env.Logf("alu", "r%d <= %d + %d = %d (was %d)", in.Rd, c.Regs[in.Rs], in.Imm, v, was)
			c.writeReg(in.Rd, v)
		case isa.OpSubtract:
			was := c.Regs[in.Rd]
			v := c.Regs[in.Rs] - c.Regs[in.Rt]
			env.Logf("alu", "r%d <= %d - %d = %d (was %d)", in.Rd, c.Regs[in.Rs], c.Regs[in.Rt], v, was)
			c.writeReg(in.Rd, v)
		case isa.OpBitAnd:
			was := c.Regs[in.Rd]
			v := c.Regs[in.Rs] & c.Regs[in.Rt]
			env.Logf("alu", "r%d <= %d & %d = %d (was %d)", in.Rd, c.Regs[in.Rs], c.Regs[in.Rt], v, was)
			c.writeReg(in.Rd, v)
		case isa.OpBitOr:
			was := c.Regs[in.Rd]
			v := c.Regs[in.Rs] | c.Regs[in.Rt]
			env.Logf("alu", "r%d <= %d | %d = %d (was %d)", in.Rd, c.Regs[in.Rs], c.Regs[in.Rt], v, was)
			c.writeReg(in.Rd, v)
		case isa.OpShiftLeft:
			was := c.Regs[in.Rd]
			n := c.Regs[in.Rt] & 31
			v := c.Regs[in.Rs] << n
			env.Logf("alu", "r%d <= %d << %d = %d (was %d)", in.Rd, c.Regs[in.Rs], n, v, was)
			c.writeReg(in.Rd, v)
		case isa.OpShiftRight:
			was := c.Regs[in.Rd]
			n := c.Regs[in.Rt] & 31
			v := c.Regs[in.Rs] >> n
			env.Logf("alu", "r%d <= %d >> %d = %d (was %d)", in.Rd, c.Regs[in.Rs], n, v, was)
			c.writeReg(in.Rd, v)
		case isa.OpShiftRightArithmetic:
			was := c.Regs[in.Rd]
			n := c.Regs[in.Rt] & 31
			v := uint32(int32(c.Regs[in.Rs]) >> n)
			env.Logf("alu", "r%d <= %d >>a %d = %d (was %d)", in.Rd, int32(c.Regs[in.Rs]), n, v, was)
			c.writeReg(in.Rd, v)

		// ---- memory: words are aligned and posted; bytes need no alignment ----
		case isa.OpLoadWord:
			addr := c.Regs[in.Rs] + uint32(in.Imm)
			if addr%4 != 0 {
				env.Logf("fault", "load_word from 0x%04x: %s", addr, BusNotPermitted)
				halt(Fault{Kind: "alignment", Addr: addr, PC: c.PC - 4, Phase: phase, Word: c.fetchWord})
				return
			}
			v, st := env.ReadBus(addr)
			if st != BusOK {
				env.Logf("fault", "load_word from 0x%04x: %s", addr, st)
				halt(Fault{Kind: "bus_" + string(st), Addr: addr, PC: c.PC - 4, Phase: phase, Word: c.fetchWord})
				return
			}
			was := c.Regs[in.Rd]
			env.Logf("load", "r%d <= mem[0x%04x] = %d (was %d)", in.Rd, addr, v, was)
			c.writeReg(in.Rd, v)
		case isa.OpStoreWord:
			addr := c.Regs[in.Rs] + uint32(in.Imm)
			if addr%4 != 0 {
				env.Logf("fault", "store_word to 0x%04x: %s", addr, BusNotPermitted)
				halt(Fault{Kind: "alignment", Addr: addr, PC: c.PC - 4, Phase: phase, Word: c.fetchWord})
				return
			}
			env.Logf("store", "mem[0x%04x] <= r%d (%d)", addr, in.Rt, c.Regs[in.Rt])
			env.WriteBus(addr, c.Regs[in.Rt])
		case isa.OpLoadByte:
			addr := c.Regs[in.Rs] + uint32(in.Imm)
			b, st := env.ReadBusByte(addr)
			if st != BusOK {
				env.Logf("fault", "load_byte from 0x%04x: %s", addr, st)
				halt(Fault{Kind: "bus_" + string(st), Addr: addr, PC: c.PC - 4, Phase: phase, Word: c.fetchWord})
				return
			}
			was := c.Regs[in.Rd]
			env.Logf("load_byte", "r%d <= mem[0x%04x] = %d (was %d)", in.Rd, addr, b, was)
			c.writeReg(in.Rd, uint32(b)) // zero-extended
		case isa.OpStoreByte:
			addr := c.Regs[in.Rs] + uint32(in.Imm)
			env.Logf("store_byte", "mem[0x%04x] <= r%d (byte %d)", addr, in.Rt, c.Regs[in.Rt]&0xff)
			env.WriteBusByte(addr, uint8(c.Regs[in.Rt]))

		// ---- control flow: pc = next fetch; "at" is this instruction ----
		case isa.OpJumpIfEqual:
			at, next := c.PC-4, c.PC
			if c.Regs[in.Rs] == c.Regs[in.Rt] {
				c.PC = uint32(int32(c.PC) + in.Imm)
				env.Logf("branch", "at=0x%04x next=0x%04x -> 0x%04x (taken)", at, next, c.PC)
			} else {
				env.Logf("branch", "at=0x%04x next=0x%04x (not taken)", at, next)
			}
		case isa.OpJumpIfNotEqual:
			at, next := c.PC-4, c.PC
			if c.Regs[in.Rs] != c.Regs[in.Rt] {
				c.PC = uint32(int32(c.PC) + in.Imm)
				env.Logf("branch", "at=0x%04x next=0x%04x -> 0x%04x (taken)", at, next, c.PC)
			} else {
				env.Logf("branch", "at=0x%04x next=0x%04x (not taken)", at, next)
			}
		case isa.OpCall:
			at := c.PC - 4
			c.Regs[7] = c.PC // r7 = link register (the phone always rings at the same desk)
			c.PC = uint32(int32(c.PC) + in.Imm)
			env.Logf("call", "at=0x%04x r7=0x%04x -> 0x%04x", at, c.Regs[7], c.PC)
		case isa.OpGoto:
			at := c.PC - 4
			c.PC = c.Regs[in.Rs]
			env.Logf("goto", "at=0x%04x -> r%d (0x%04x)", at, in.Rs, c.PC)

		// ---- system ----
		case isa.OpAllowInterrupts:
			c.Status |= StatusIE
			env.Logf("status", "ie=1")
		case isa.OpForbidInterrupts:
			c.Status &^= StatusIE
			env.Logf("status", "ie=0")
		case isa.OpReturnFromInterrupt:
			if !c.inHandler {
				env.Logf("fault", "return_from_interrupt with no interrupt in flight")
				halt(Fault{Kind: "stray_rfi", PC: c.PC - 4, Phase: phase, Word: c.fetchWord})
				return
			}
			at := c.PC - 4
			c.PC, c.Status = c.intPC, c.intStatus
			c.inHandler = false
			env.Logf("rfi", "at=0x%04x pc -> 0x%04x (status 0x%02x)", at, c.PC, c.Status)
		}
		c.decoded = nil
	}
}

// HaltedBy reports the terminal fault record.
func (c *CPU) HaltedBy() Fault {
	if c.haltedBy != nil {
		return *c.haltedBy
	}
	return Fault{Kind: "halt"}
}

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
	if c.haltedBy != nil {
		if c.haltedBy.Kind == "halt" {
			s += fmt.Sprintf("halt: clean stop at pc=0x%04x phase=%s\n", c.haltedBy.PC, c.haltedBy.Phase)
		} else {
			s += fmt.Sprintf("fault: %s at pc=0x%04x phase=%s addr=0x%04x\n",
				c.haltedBy.Kind, c.haltedBy.PC, c.haltedBy.Phase, c.haltedBy.Addr)
		}
	}
	return s
}

func privName(status uint32) string {
	if status&StatusPriv != 0 {
		return "kernel"
	}
	return "user"
}
