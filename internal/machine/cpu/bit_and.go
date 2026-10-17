// bit_and.go — bit_and rd, rs, rt: bitwise AND (masking, field extraction).
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpBitAnd, execBitAnd) }

func execBitAnd(c *CPU, in isa.Instruction, env machine.Env) {
	was := c.Regs[in.Rd]
	v := c.Regs[in.Rs] & c.Regs[in.Rt]
	env.Logf("alu", "r%d <= %d & %d = %d (was %d)", in.Rd, c.Regs[in.Rs], c.Regs[in.Rt], v, was)
	c.writeReg(in.Rd, v)
}
