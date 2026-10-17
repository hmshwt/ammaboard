// store_word.go — store_word rt, imm(rs): memory[rs + imm] = rt, word-aligned,
// posted to the bus (visible at the phase latch). Unaligned addresses fault;
// a write that lands on no component stops the machine at the latch.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpStoreWord, execStoreWord) }

func execStoreWord(c *CPU, in isa.Instruction, env machine.Env) {
	addr := c.Regs[in.Rs] + uint32(in.Imm)
	if addr%4 != 0 {
		env.Logf("fault", "store_word to 0x%04x: %s", addr, machine.BusNotPermitted)
		c.stop(env, machine.Fault{Kind: "alignment", Addr: addr, PC: c.at(),
			Phase: machine.PhaseExecute, Word: c.fetchWord})
		return
	}
	env.Logf("store", "mem[0x%04x] <= r%d (%d)", addr, in.Rt, c.Regs[in.Rt])
	env.WriteBus(addr, c.Regs[in.Rt], machine.WriteSite{PC: c.at(), Word: c.fetchWord})
}
