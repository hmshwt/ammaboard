// Package isa_test holds the ISA conformance suite: the contract between the
// table (isa.Specs), the assembler, and the CPU. External test package on
// purpose — it imports ammaboard/internal/asm and .../machine/cpu, which both
// depend on isa, so it must not live inside package isa.
package isa_test

import (
	"strings"
	"testing"

	"ammaboard/internal/asm"
	"ammaboard/internal/isa"
	"ammaboard/internal/machine/cpu"
)

// TestSpecsCoverEveryOpcode: one Spec per opcode, 1..NumOpcodes, no gaps, no
// duplicates, unique mnemonics.
func TestSpecsCoverEveryOpcode(t *testing.T) {
	byOp := map[isa.Opcode]isa.Spec{}
	byName := map[string]isa.Opcode{}
	for _, s := range isa.Specs {
		if s.Mnemonic == "" {
			t.Errorf("opcode %d has an empty mnemonic", s.Op)
		}
		if prev, dup := byOp[s.Op]; dup {
			t.Errorf("opcode %d described twice (%s and %s)", s.Op, prev.Mnemonic, s.Mnemonic)
		}
		byOp[s.Op] = s
		if prev, dup := byName[s.Mnemonic]; dup {
			t.Errorf("mnemonic %q used by opcodes %d and %d", s.Mnemonic, prev, s.Op)
		}
		byName[s.Mnemonic] = s.Op
	}
	if len(isa.Specs) != isa.NumOpcodes {
		t.Errorf("Specs has %d rows, want NumOpcodes = %d", len(isa.Specs), isa.NumOpcodes)
	}
	for op := isa.Opcode(1); op <= isa.NumOpcodes; op++ {
		if _, ok := byOp[op]; !ok {
			t.Errorf("opcode %d has no Spec", op)
		}
	}
}

// canonicalLine builds the smallest valid source line for a Spec's form.
func canonicalLine(s isa.Spec) string {
	switch s.Form {
	case isa.FormZero:
		return s.Mnemonic
	case isa.FormRRR:
		return s.Mnemonic + " r1, r2, r3"
	case isa.FormRRI:
		return s.Mnemonic + " r1, r2, 5"
	case isa.FormLoad:
		return s.Mnemonic + " r1, 0(r2)"
	case isa.FormStore:
		return s.Mnemonic + " r1, 0(r2)"
	case isa.FormBranch:
		return s.Mnemonic + " r1, r2, 4"
	case isa.FormCall:
		return s.Mnemonic + " 4"
	case isa.FormGoto:
		return s.Mnemonic + " r1"
	default:
		return ""
	}
}

// TestAssemblerParity: every Spec's mnemonic assembles, and the word it
// produces decodes back to the same opcode. This is the test that fails when
// the assembler and the ISA table drift apart.
func TestAssemblerParity(t *testing.T) {
	for _, s := range isa.Specs {
		line := canonicalLine(s)
		if line == "" {
			t.Errorf("%s: no canonical line for form %d", s.Mnemonic, s.Form)
			continue
		}
		words, err := asm.Assemble(line)
		if err != nil {
			t.Errorf("%s: assembler refuses its own canonical line %q: %v", s.Mnemonic, line, err)
			continue
		}
		if len(words) != 1 {
			t.Errorf("%s: %q assembled to %d words, want 1", s.Mnemonic, line, len(words))
			continue
		}
		in, err := isa.Decode(words[0])
		if err != nil {
			t.Errorf("%s: word from %q does not decode: %v", s.Mnemonic, line, err)
			continue
		}
		if in.Op != s.Op {
			t.Errorf("%s: assembled to opcode %d, want %d", s.Mnemonic, in.Op, s.Op)
		}
	}
}

// TestCPUParity: every Spec has a CPU executor, and the CPU registers nothing
// that is not a Spec. An instruction is only real when all three layers agree.
func TestCPUParity(t *testing.T) {
	for _, s := range isa.Specs {
		if !cpu.Executable(s.Op) {
			t.Errorf("cpu has no executor for %s (opcode %d)", s.Mnemonic, s.Op)
		}
	}
	for _, op := range cpu.Executables() {
		if _, ok := isa.SpecFor(op); !ok {
			t.Errorf("cpu registers opcode %d, which has no Spec", op)
		}
	}
}

// TestUnknownMnemonicStillRefused: adds nothing, but pins the error shape a
// contributor sees when they register a CPU executor and forget the Spec.
func TestUnknownMnemonicStillRefused(t *testing.T) {
	if _, err := asm.Assemble("frobnicate r1, r2"); err == nil || !strings.Contains(err.Error(), "unknown mnemonic") {
		t.Errorf("unknown mnemonic should be refused by name, got: %v", err)
	}
}

// TestDisassembleRoundTrip: every decodable word re-assembles to itself from
// its disassembly — the TUI program pane shows exactly what runs.
func TestDisassembleRoundTrip(t *testing.T) {
	cases := []struct {
		op         isa.Opcode
		rd, rs, rt uint8
		imm        int32
	}{
		{isa.OpHalt, 0, 0, 0, 0},
		{isa.OpAllowInterrupts, 0, 0, 0, 0},
		{isa.OpAdd, 1, 2, 3, 0},
		{isa.OpShiftRightArithmetic, 7, 6, 5, 0},
		{isa.OpAddNumber, 4, 5, 0, -7},
		{isa.OpLoadWord, 2, 3, 0, 64},
		{isa.OpLoadByte, 1, 2, 0, 300},
		{isa.OpStoreWord, 0, 3, 2, -8},
		{isa.OpStoreByte, 0, 6, 4, 12},
		{isa.OpJumpIfEqual, 0, 1, 2, 16},
		{isa.OpJumpIfNotEqual, 0, 1, 2, -8},
		{isa.OpCall, 0, 0, 0, 44},
		{isa.OpGoto, 0, 1, 0, 0},
	}
	for _, c := range cases {
		w := isa.MustEncode(c.op, c.rd, c.rs, c.rt, c.imm)
		src := isa.Disassemble(w)
		words, err := asm.Assemble(src)
		if err != nil {
			t.Errorf("%s: does not re-assemble: %v", src, err)
			continue
		}
		if len(words) != 1 || words[0] != w {
			t.Errorf("%s re-assembles to %08x, want %08x", src, words[0], w)
		}
	}
	// Words that are not instructions say so honestly.
	if s := isa.Disassemble(0xffffffff); !strings.Contains(s, ".word") {
		t.Errorf("data word should disassemble as .word, got %q", s)
	}
}
