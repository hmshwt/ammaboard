package cpu

import (
	"ammaboard/internal/machine"
)

// execute is the one call the phase loop makes into an instruction: the
// registry routes the decoded opcode to its file's exec function. Adding an
// instruction never edits this file (docs/implementation.md).
func (c *CPU) execute(env machine.Env) {
	in := c.decoded
	if in == nil {
		return
	}
	defer func() { c.decoded = nil }()
	fn, ok := lookup(in.Op)
	if !ok {
		// Decode already refused unknown opcodes; this guards a registry
		// hole (a missing file), and still refuses loudly.
		env.Logf("fault", "no implementation for %s (word 0x%08x)", in.Op.Name(), c.fetchWord)
		c.stop(env, machine.Fault{Kind: "unknown_opcode", PC: c.at(),
			Phase: machine.PhaseExecute, Word: c.fetchWord})
		return
	}
	fn(c, *in, env)
}
