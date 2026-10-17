// return_from_interrupt.go — return_from_interrupt: restore the pc and status
// parked by a taken interrupt. Running it with no interrupt in flight is
// nonsense — the machine refuses it loudly (stray_rfi).
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpReturnFromInterrupt, execReturnFromInterrupt) }

func execReturnFromInterrupt(c *CPU, in isa.Instruction, env machine.Env) {
	if !c.inHandler {
		env.Logf("fault", "return_from_interrupt with no interrupt in flight")
		c.stop(env, machine.Fault{Kind: "stray_rfi", PC: c.at(),
			Phase: machine.PhaseExecute, Word: c.fetchWord})
		return
	}
	at := c.at()
	c.PC, c.Status = c.intPC, c.intStatus
	c.inHandler = false
	env.Logf("rfi", "at=0x%04x pc -> 0x%04x (status 0x%02x)", at, c.PC, c.Status)
}
