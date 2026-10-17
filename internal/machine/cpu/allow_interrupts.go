// allow_interrupts.go — allow_interrupts: set IE. A line already raised may
// be delivered at the next latch edge.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpAllowInterrupts, execAllowInterrupts) }

func execAllowInterrupts(c *CPU, in isa.Instruction, env machine.Env) {
	c.Status |= StatusIE
	env.Logf("status", "ie=1")
}
