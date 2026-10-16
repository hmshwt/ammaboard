// Package asm assembles AmmaBoard boot-ten assembly into machine words.
// Two-pass: pass 1 resolves labels, pass 2 encodes. Branch/call immediates are
// byte offsets relative to the instruction after the branch (pc is advanced at
// fetch), matching the CPU's semantics.
package asm

import (
	"fmt"
	"strconv"
	"strings"

	"ammaboard/internal/isa"
)

// Assemble converts source text to machine words. Comments start with ';' or '#'.
// Operand grammar (destination-first):
//
//	add        rd, rs, rt
//	add_number rd, rs, imm
//	subtract   rd, rs, rt
//	load_word  rd, imm(rs)
//	store_word rt, imm(rs)
//	jump_if_equal   rs, rt, label|imm
//	jump_if_not_equal rs, rt, label|imm
//	call       label|imm          (return address -> r7)
//	goto       rs
//	halt
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
	val := func(i int) (string, error) {
		if i >= len(args) {
			return "", fmt.Errorf("missing operand %d", i+1)
		}
		return args[i], nil
	}

	switch mnem {
	case "halt":
		return isa.MustEncode(isa.OpHalt, 0, 0, 0, 0), nil

	case "add", "subtract":
		rd, err := val(0)
		if err != nil {
			return 0, err
		}
		rs, err := val(1)
		if err != nil {
			return 0, err
		}
		rt, err := val(2)
		if err != nil {
			return 0, err
		}
		r1, e1 := reg(rd)
		r2, e2 := reg(rs)
		r3, e3 := reg(rt)
		if err := firstErr(e1, e2, e3); err != nil {
			return 0, err
		}
		op := isa.OpAdd
		if mnem == "subtract" {
			op = isa.OpSubtract
		}
		return isa.MustEncode(op, r1, r2, r3, 0), nil

	case "add_number":
		r1, err := reg(must(args, 0))
		r2, e2 := reg(must(args, 1))
		var imm int32
		if e2 == nil {
			imm, err = immOrLabel(must(args, 2), labels, here, false)
		}
		if err != nil || e2 != nil {
			return 0, firstErr(err, e2)
		}
		return isa.MustEncode(isa.OpAddNumber, r1, r2, 0, imm), nil

	case "load_word":
		rd, e1 := reg(must(args, 0))
		base, off, e2 := memOperand(must(args, 1), labels)
		if err := firstErr(e1, e2); err != nil {
			return 0, err
		}
		return isa.MustEncode(isa.OpLoadWord, rd, base, 0, off), nil

	case "store_word":
		rt, e1 := reg(must(args, 0))
		base, off, e2 := memOperand(must(args, 1), labels)
		if err := firstErr(e1, e2); err != nil {
			return 0, err
		}
		return isa.MustEncode(isa.OpStoreWord, 0, base, rt, off), nil

	case "jump_if_equal", "jump_if_not_equal":
		r1, e1 := reg(must(args, 0))
		r2, e2 := reg(must(args, 1))
		var imm int32
		var e3 error
		if e1 == nil && e2 == nil {
			imm, e3 = immOrLabel(must(args, 2), labels, here, true)
		}
		if err := firstErr(e1, e2, e3); err != nil {
			return 0, err
		}
		op := isa.OpJumpIfEqual
		if mnem == "jump_if_not_equal" {
			op = isa.OpJumpIfNotEqual
		}
		return isa.MustEncode(op, 0, r1, r2, imm), nil

	case "call":
		imm, err := immOrLabel(must(args, 0), labels, here, true)
		if err != nil {
			return 0, err
		}
		return isa.MustEncode(isa.OpCall, 0, 0, 0, imm), nil

	case "goto":
		rs, err := reg(must(args, 0))
		if err != nil {
			return 0, err
		}
		return isa.MustEncode(isa.OpGoto, 0, rs, 0, 0), nil
	}
	return 0, fmt.Errorf("unknown mnemonic %q", mnem)
}

func must(args []string, i int) string {
	if i >= len(args) {
		return ""
	}
	return args[i]
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}
