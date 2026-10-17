// load_word.go — load_word rd, imm(rs): rd = memory[rs + imm], word-aligned.
// Unaligned addresses fault: the machine refuses nonsense.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpLoadWord, execLoadWord) }

func execLoadWord(c *CPU, in isa.Instruction, env machine.Env) {
	addr := c.Regs[in.Rs] + uint32(in.Imm)
	if addr%4 != 0 {
		env.Logf("fault", "load_word from 0x%04x: %s", addr, machine.BusNotPermitted)
		c.stop(env, machine.Fault{Kind: "alignment", Addr: addr, PC: c.at(),
			Phase: machine.PhaseExecute, Word: c.fetchWord})
		return
	}
	v, st := env.ReadBus(addr)
	if st != machine.BusOK {
		env.Logf("fault", "load_word from 0x%04x: %s", addr, st)
		c.stop(env, machine.Fault{Kind: "bus_" + string(st), Addr: addr, PC: c.at(),
			Phase: machine.PhaseExecute, Word: c.fetchWord})
		return
	}
	was := c.Regs[in.Rd]
	env.Logf("load", "r%d <= mem[0x%04x] = %d (was %d)", in.Rd, addr, v, was)
	c.writeReg(in.Rd, v)
}
