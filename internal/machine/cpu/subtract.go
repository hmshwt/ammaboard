// subtract.go — subtract rd, rs, rt: rd = rs - rt (wraps, two's complement).
// There are no condition flags: comparisons are "subtract then branch on
// zero", by design (a day-4/5 primer note).
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpSubtract, execSubtract) }

func execSubtract(c *CPU, in isa.Instruction, env machine.Env) {
	was := c.Regs[in.Rd]
	v := c.Regs[in.Rs] - c.Regs[in.Rt]
	env.Logf("alu", "r%d <= %d - %d = %d (was %d)", in.Rd, c.Regs[in.Rs], c.Regs[in.Rt], v, was)
	c.writeReg(in.Rd, v)
}
