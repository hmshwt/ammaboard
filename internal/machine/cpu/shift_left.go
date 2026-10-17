// shift_left.go — shift_left rd, rs, rt: rd = rs << (rt & 31). Building
// addresses and flags. No immediate-shift encoding until sugar needs one.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpShiftLeft, execShiftLeft) }

func execShiftLeft(c *CPU, in isa.Instruction, env machine.Env) {
	was := c.Regs[in.Rd]
	n := c.Regs[in.Rt] & 31
	v := c.Regs[in.Rs] << n
	env.Logf("alu", "r%d <= %d << %d = %d (was %d)", in.Rd, c.Regs[in.Rs], n, v, was)
	c.writeReg(in.Rd, v)
}
