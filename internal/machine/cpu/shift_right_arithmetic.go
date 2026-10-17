// shift_right_arithmetic.go — shift_right_arithmetic rd, rs, rt:
// sign-preserving right shift by (rt & 31) — division by powers of two that
// keeps negative numbers negative.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpShiftRightArithmetic, execShiftRightArithmetic) }

func execShiftRightArithmetic(c *CPU, in isa.Instruction, env machine.Env) {
	was := c.Regs[in.Rd]
	n := c.Regs[in.Rt] & 31
	v := uint32(int32(c.Regs[in.Rs]) >> n)
	env.Logf("alu", "r%d <= %d >>a %d = %d (was %d)", in.Rd, int32(c.Regs[in.Rs]), n, v, was)
	c.writeReg(in.Rd, v)
}
