// goto.go — goto rs: pc = rs. The return half of call (goto r7) and the one
// indirect jump on the machine.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpGoto, execGoto) }

func execGoto(c *CPU, in isa.Instruction, env machine.Env) {
	at := c.at()
	c.PC = c.Regs[in.Rs]
	env.Logf("goto", "at=0x%04x -> r%d (0x%04x)", at, in.Rs, c.PC)
}
