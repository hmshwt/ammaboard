// call.go — call imm: pc += imm and the return address lands in r7, the fixed
// link register (the phone always rings at the same desk). Generality becomes
// assembler sugar later; the hardware stays one register wide.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpCall, execCall) }

func execCall(c *CPU, in isa.Instruction, env machine.Env) {
	at := c.at()
	c.Regs[7] = c.PC // r7 = link register (r7 is never r0: plain store is fine)
	c.PC = uint32(int32(c.PC) + in.Imm)
	env.Logf("call", "at=0x%04x r7=0x%04x -> 0x%04x", at, c.Regs[7], c.PC)
}
