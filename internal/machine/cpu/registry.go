package cpu

import (
	"sort"

	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

// execFunc executes one decoded instruction: the CPU's state, the decoded
// operand fields, the broker/log. Per-instruction files register their
// function in init() — the execute loop itself stays untouched.
type execFunc func(c *CPU, in isa.Instruction, env machine.Env)

// instructions is the opcode registry, filled at init time by the
// per-instruction files (add.go, load_word.go, …).
var instructions = map[isa.Opcode]execFunc{}

// register adds one instruction to the table. Duplicate registrations are a
// programming error and panic at init time, not at run time.
func register(op isa.Opcode, fn execFunc) {
	if _, dup := instructions[op]; dup {
		panic("cpu: instruction already registered: " + op.Name())
	}
	instructions[op] = fn
}

func lookup(op isa.Opcode) (execFunc, bool) {
	fn, ok := instructions[op]
	return fn, ok
}

// Executable reports whether the CPU has an executor for op — the CPU's side
// of the ISA parity contract (see internal/isa/parity_test.go).
func Executable(op isa.Opcode) bool {
	_, ok := instructions[op]
	return ok
}

// Executables returns every registered opcode, sorted.
func Executables() []isa.Opcode {
	ops := make([]isa.Opcode, 0, len(instructions))
	for op := range instructions {
		ops = append(ops, op)
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i] < ops[j] })
	return ops
}
