// shift_right.go — shift_right rd, rs, rt: logical right shift by (rt & 31),
// zero-filling from the top — the tool for field extraction.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpShiftRight, execShiftRight) }

func execShiftRight(c *CPU, in isa.Instruction, env machine.Env) {
	was := c.Regs[in.Rd]
	n := c.Regs[in.Rt] & 31
	v := c.Regs[in.Rs] >> n
	env.Logf("alu", "r%d <= %d >> %d = %d (was %d)", in.Rd, c.Regs[in.Rs], n, v, was)
	c.writeReg(in.Rd, v)
}
