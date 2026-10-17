// Package asm assembles AmmaBoard assembly into machine words. Two-pass: pass
// 1 resolves labels, pass 2 encodes. Branch/call immediates are byte offsets
// relative to the instruction after the branch (pc is advanced at fetch),
// matching the CPU's semantics.
//
// Operand forms come from isa.Specs (the ISA's single source of truth); this
// package implements one encoder per form:
//
//	FormZero   no operands                halt
//	FormRRR    rd, rs, rt                 add, subtract, shifts, bitwise
//	FormRRI    rd, rs, imm|label          add_number (label = absolute address)
//	FormLoad   rd, imm(rs)                load_word, load_byte (no alignment for bytes)
//	FormStore  rt, imm(rs)                store_word, store_byte
//	FormBranch rs, rt, target|label       jump_if_equal, jump_if_not_equal
//	FormCall   target|label               call (return address -> r7)
//	FormGoto   rs                         goto
package asm

import (
	"fmt"
	"strconv"
	"strings"

	"ammaboard/internal/isa"
)

// Assemble converts source text to machine words. Comments start with ';' or
// '#'. Operand order is destination-first; the operand forms live in isa.Specs
// (summarized in the package doc) — this file never lists instructions.
func Assemble(src string) ([]uint32, error) {
	type line struct {
		no   int
		text string
	}
	var lines []line
	labels := map[string]int{} // label -> word index

	// Pass 1: strip comments, collect labels, count words.
	idx := 0
	for no, raw := range strings.Split(src, "\n") {
		t := raw
		if i := strings.IndexAny(t, ";#"); i >= 0 {
			t = t[:i]
		}
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		for strings.Contains(t, ":") && !strings.HasPrefix(t, " ") {
			name, rest, ok := cutLabel(t)
			if !ok {
				break
			}
			labels[name] = idx
			t = strings.TrimSpace(rest)
		}
		if t == "" {
			continue
		}
		lines = append(lines, line{no: no + 1, text: t})
		idx++
	}

	// Pass 2: encode.
	var out []uint32
	for _, l := range lines {
		w, err := encodeLine(l.text, labels, len(out))
		if err != nil {
			return nil, fmt.Errorf("line %d (%q): %w", l.no, l.text, err)
		}
		out = append(out, w)
	}
	return out, nil
}

func cutLabel(t string) (name, rest string, ok bool) {
	i := strings.Index(t, ":")
	if i <= 0 {
		return "", t, false
	}
	name = strings.TrimSpace(t[:i])
	if strings.ContainsAny(name, " \t,") {
		return "", t, false
	}
	return name, t[i+1:], true
}

func reg(s string) (uint8, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) != 2 || s[0] != 'r' || s[1] < '0' || s[1] > '7' {
		return 0, fmt.Errorf("bad register %q (want r0..r7)", s)
	}
	return uint8(s[1] - '0'), nil
}

func immOrLabel(s string, labels map[string]int, here int, isBranch bool) (int32, error) {
	s = strings.TrimSpace(s)
	if v, err := strconv.Atoi(s); err == nil {
		return int32(v), nil
	}
	idx, ok := labels[s]
	if !ok {
		return 0, fmt.Errorf("unknown label or number %q", s)
	}
	target := idx * 4
	if isBranch {
		return int32(target - (here*4 + 4)), nil
	}
	return int32(target), nil
}

// memOperand parses "imm(rs)" / "label(rs)" / "(rs)".
func memOperand(s string, labels map[string]int) (uint8, int32, error) {
	open := strings.Index(s, "(")
	closeP := strings.Index(s, ")")
	if open < 0 || closeP < len(s)-1 {
		return 0, 0, fmt.Errorf("bad memory operand %q (want imm(rs))", s)
	}
	base, err := reg(s[open+1 : closeP])
	if err != nil {
		return 0, 0, err
	}
	offStr := strings.TrimSpace(s[:open])
	off := int32(0)
	if offStr != "" {
		if off, err = immOrLabel(offStr, labels, 0, false); err != nil {
			return 0, 0, err
		}
	}
	return base, off, nil
}

func encodeLine(t string, labels map[string]int, here int) (uint32, error) {
	fields := strings.FieldsFunc(t, func(r rune) bool { return r == ' ' || r == ',' })
	if len(fields) == 0 {
		return 0, fmt.Errorf("empty instruction")
	}
	mnem := strings.ToLower(fields[0])
	args := fields[1:]
	// No per-instruction knowledge lives here: isa.Specs says each mnemonic's
	// operand form, and this file implements one encoder per form. Adding an
	// instruction whose form already exists needs no change in this file.
	spec, ok := isa.Lookup(mnem)
	if !ok {
		return 0, fmt.Errorf("unknown mnemonic %q", mnem)
	}
	switch spec.Form {
	case isa.FormZero:
		if len(args) != 0 {
			return 0, fmt.Errorf("%s takes no operands", mnem)
		}
		return enc(spec.Op, 0, 0, 0, 0)

	case isa.FormRRR: // rd, rs, rt
		if len(args) != 3 {
			return 0, fmt.Errorf("%s wants rd, rs, rt", mnem)
		}
		rd, e1 := reg(args[0])
		rs, e2 := reg(args[1])
		rt, e3 := reg(args[2])
		if err := firstErr(e1, e2, e3); err != nil {
			return 0, err
		}
		return enc(spec.Op, rd, rs, rt, 0)

	case isa.FormRRI: // rd, rs, imm|label
		if len(args) != 3 {
			return 0, fmt.Errorf("%s wants rd, rs, imm", mnem)
		}
		rd, e1 := reg(args[0])
		rs, e2 := reg(args[1])
		var imm int32
		var e3 error
		if e1 == nil && e2 == nil {
			imm, e3 = immOrLabel(args[2], labels, here, false)
		}
		if err := firstErr(e1, e2, e3); err != nil {
			return 0, err
		}
		return enc(spec.Op, rd, rs, 0, imm)

	case isa.FormLoad: // rd, imm(rs)
		if len(args) != 2 {
			return 0, fmt.Errorf("%s wants rd, imm(rs)", mnem)
		}
		rd, e1 := reg(args[0])
		base, off, e2 := memOperand(args[1], labels)
		if err := firstErr(e1, e2); err != nil {
			return 0, err
		}
		return enc(spec.Op, rd, base, 0, off)

	case isa.FormStore: // rt, imm(rs)
		if len(args) != 2 {
			return 0, fmt.Errorf("%s wants rt, imm(rs)", mnem)
		}
		rt, e1 := reg(args[0])
		base, off, e2 := memOperand(args[1], labels)
		if err := firstErr(e1, e2); err != nil {
			return 0, err
		}
		return enc(spec.Op, 0, base, rt, off)

	case isa.FormBranch: // rs, rt, target
		if len(args) != 3 {
			return 0, fmt.Errorf("%s wants rs, rt, target", mnem)
		}
		rs, e1 := reg(args[0])
		rt, e2 := reg(args[1])
		var imm int32
		var e3 error
		if e1 == nil && e2 == nil {
			imm, e3 = immOrLabel(args[2], labels, here, true)
		}
		if err := firstErr(e1, e2, e3); err != nil {
			return 0, err
		}
		return enc(spec.Op, 0, rs, rt, imm)

	case isa.FormCall: // target
		if len(args) != 1 {
			return 0, fmt.Errorf("%s wants target", mnem)
		}
		imm, err := immOrLabel(args[0], labels, here, true)
		if err != nil {
			return 0, err
		}
		return enc(spec.Op, 0, 0, 0, imm)

	case isa.FormGoto: // rs
		if len(args) != 1 {
			return 0, fmt.Errorf("%s wants rs", mnem)
		}
		rs, err := reg(args[0])
		if err != nil {
			return 0, err
		}
		return enc(spec.Op, 0, rs, 0, 0)
	}
	return 0, fmt.Errorf("%s: unsupported operand form %d", mnem, spec.Form)
}

// enc packs one instruction, returning Encode's error instead of panicking —
// bad source must be reported as a line error, never crash the tool.
func enc(op isa.Opcode, rd, rs, rt uint8, imm int32) (uint32, error) {
	return isa.Encode(op, rd, rs, rt, imm)
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}
