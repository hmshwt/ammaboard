// jump_if_not_equal.go — jump_if_not_equal rs, rt, imm: pc += imm when
// rs != rt. The loop instruction of the boot ten.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpJumpIfNotEqual, execJumpIfNotEqual) }

func execJumpIfNotEqual(c *CPU, in isa.Instruction, env machine.Env) {
	at, next := c.at(), c.PC
	if c.Regs[in.Rs] != c.Regs[in.Rt] {
		c.PC = uint32(int32(c.PC) + in.Imm)
		env.Logf("branch", "at=0x%04x next=0x%04x -> 0x%04x (taken)", at, next, c.PC)
	} else {
		env.Logf("branch", "at=0x%04x next=0x%04x (not taken)", at, next)
	}
}
