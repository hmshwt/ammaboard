// add.go — add rd, rs, rt: three-register integer addition.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpAdd, execAdd) }

func execAdd(c *CPU, in isa.Instruction, env machine.Env) {
	was := c.Regs[in.Rd]
	v := c.Regs[in.Rs] + c.Regs[in.Rt]
	env.Logf("alu", "r%d <= %d + %d = %d (was %d)", in.Rd, c.Regs[in.Rs], c.Regs[in.Rt], v, was)
	c.writeReg(in.Rd, v)
}
