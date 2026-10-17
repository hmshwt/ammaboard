// halt.go — halt: the program is done, the machine stops cleanly. A clean
// stop is not a fault: no fault record, and no Blue Screen — the board just
// marks the machine stopped. Both pc conventions are shown: the halt's own
// address and the next fetch.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpHalt, execHalt) }

func execHalt(c *CPU, in isa.Instruction, env machine.Env) {
	at := c.at()
	env.Logf("halt", "clean stop at pc=0x%04x (next fetch 0x%04x)", at, c.PC)
	c.Halted = true
	c.stopPC = at
	env.CleanHalt()
}
