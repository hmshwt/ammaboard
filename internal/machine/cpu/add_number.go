// add_number.go — add_number rd, rs, imm: register plus an 18-bit immediate
// (also how absolute byte addresses are loaded: label or number).
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpAddNumber, execAddNumber) }

func execAddNumber(c *CPU, in isa.Instruction, env machine.Env) {
	was := c.Regs[in.Rd]
	v := c.Regs[in.Rs] + uint32(in.Imm)
	env.Logf("alu", "r%d <= %d + %d = %d (was %d)", in.Rd, c.Regs[in.Rs], in.Imm, v, was)
	c.writeReg(in.Rd, v)
}
