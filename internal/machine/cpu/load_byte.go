// load_byte.go — load_byte rd, imm(rs): zero-extended byte read. Byte access
// has no alignment constraint — that is its point (stage 2, day 7).
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpLoadByte, execLoadByte) }

func execLoadByte(c *CPU, in isa.Instruction, env machine.Env) {
	addr := c.Regs[in.Rs] + uint32(in.Imm)
	b, st := env.ReadBusByte(addr)
	if st != machine.BusOK {
		env.Logf("fault", "load_byte from 0x%04x: %s", addr, st)
		c.stop(env, machine.Fault{Kind: "bus_" + string(st), Addr: addr, PC: c.at(),
			Phase: machine.PhaseExecute, Word: c.fetchWord})
		return
	}
	was := c.Regs[in.Rd]
	env.Logf("load_byte", "r%d <= mem[0x%04x] = %d (was %d)", in.Rd, addr, b, was)
	c.writeReg(in.Rd, uint32(b)) // zero-extended
}
