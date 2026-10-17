// jump_if_equal.go — jump_if_equal rs, rt, imm: pc += imm when rs == rt.
// The immediate is a byte offset relative to the next instruction (pc is
// advanced at fetch).
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpJumpIfEqual, execJumpIfEqual) }

func execJumpIfEqual(c *CPU, in isa.Instruction, env machine.Env) {
	at, next := c.at(), c.PC
	if c.Regs[in.Rs] == c.Regs[in.Rt] {
		c.PC = uint32(int32(c.PC) + in.Imm)
		env.Logf("branch", "at=0x%04x next=0x%04x -> 0x%04x (taken)", at, next, c.PC)
	} else {
		env.Logf("branch", "at=0x%04x next=0x%04x (not taken)", at, next)
	}
}
