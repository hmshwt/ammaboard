package machine

import (
	"fmt"

	"ammaboard/internal/isa"
)

// CPU is the AmmaBoard-1 processor: 8 registers (r0 hardwired zero), pc, and
// the boot-ten instruction set. Fault plumbing lives in the struct from day
// one — dormant until day 10, per poc.md.
type CPU struct {
	Regs   [8]uint32
	PC     uint32
	Halted bool
	Fault  *isa.Instruction // decoded word of the instruction in flight, if any

	haltedBy  *Fault
	fetchWord uint32
	decoded   *isa.Instruction
}

func NewCPU(pc uint32) *CPU { return &CPU{PC: pc} }

func (c *CPU) Name() string { return "cpu" }

func (c *CPU) writeReg(n uint8, v uint32) {
	if n == 0 {
		return // r0 is hardwired zero: the machine's one free constant
	}
	c.Regs[n] = v
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
			env.Logf("halt", "clean stop at pc=0x%04x", c.PC-4)
			halt(Fault{Kind: "halt", PC: c.PC - 4, Phase: phase})

		case isa.OpAdd:
			c.writeReg(in.Rd, c.Regs[in.Rs]+c.Regs[in.Rt])
		case isa.OpAddNumber:
			c.writeReg(in.Rd, c.Regs[in.Rs]+uint32(in.Imm))
		case isa.OpSubtract:
			c.writeReg(in.Rd, c.Regs[in.Rs]-c.Regs[in.Rt])

		case isa.OpLoadWord:
			addr := c.Regs[in.Rs] + uint32(in.Imm)
			v, st := env.ReadBus(addr)
			if st != BusOK {
				env.Logf("fault", "load_word from 0x%04x: %s", addr, st)
				halt(Fault{Kind: "alignment", Addr: addr, PC: c.PC - 4, Phase: phase})
				return
			}
			env.Logf("load", "r%d <= mem[0x%04x] = %d", in.Rd, addr, v)
			c.writeReg(in.Rd, v)

		case isa.OpStoreWord:
			addr := c.Regs[in.Rs] + uint32(in.Imm)
			if addr%4 != 0 {
				env.Logf("fault", "store_word to 0x%04x: %s", addr, BusNotPermitted)
				halt(Fault{Kind: "alignment", Addr: addr, PC: c.PC - 4, Phase: phase})
				return
			}
			env.Logf("store", "mem[0x%04x] <= r%d (%d)", addr, in.Rt, c.Regs[in.Rt])
			env.WriteBus(addr, c.Regs[in.Rt])

		case isa.OpJumpIfEqual:
			if c.Regs[in.Rs] == c.Regs[in.Rt] {
				env.Logf("branch", "taken: pc 0x%04x -> 0x%04x", c.PC, uint32(int32(c.PC)+in.Imm))
				c.PC = uint32(int32(c.PC) + in.Imm)
			}
		case isa.OpJumpIfNotEqual:
			if c.Regs[in.Rs] != c.Regs[in.Rt] {
				env.Logf("branch", "taken: pc 0x%04x -> 0x%04x", c.PC, uint32(int32(c.PC)+in.Imm))
				c.PC = uint32(int32(c.PC) + in.Imm)
			}

		case isa.OpCall:
			c.Regs[7] = c.PC // r7 = link register (the phone always rings at the same desk)
			env.Logf("call", "r7=0x%04x, pc 0x%04x -> 0x%04x", c.PC, c.PC, uint32(int32(c.PC)+in.Imm))
			c.PC = uint32(int32(c.PC) + in.Imm)

		case isa.OpGoto:
			env.Logf("goto", "pc 0x%04x -> r%d (0x%04x)", c.PC, in.Rs, c.Regs[in.Rs])
			c.PC = c.Regs[in.Rs]
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

// StateDump renders registers and pc (part of the freeze snapshot).
func (c *CPU) StateDump() string {
	s := fmt.Sprintf("pc=0x%04x\n", c.PC)
	for i, v := range c.Regs {
		s += fmt.Sprintf("r%d=%-11d 0x%08x", i, v, v)
		if i%2 == 1 {
			s += "\n"
		} else {
			s += "  "
		}
	}
	if c.haltedBy != nil {
		s += fmt.Sprintf("fault: %s at pc=0x%04x phase=%s addr=0x%04x\n",
			c.haltedBy.Kind, c.haltedBy.PC, c.haltedBy.Phase, c.haltedBy.Addr)
	}
	return s
}
