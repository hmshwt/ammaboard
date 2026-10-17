// forbid_interrupts.go — forbid_interrupts: clear IE. Raised lines stay up
// (level-triggered) and retry once interrupts are allowed again.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpForbidInterrupts, execForbidInterrupts) }

func execForbidInterrupts(c *CPU, in isa.Instruction, env machine.Env) {
	c.Status &^= StatusIE
	env.Logf("status", "ie=0")
}
