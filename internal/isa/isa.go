// Package isa defines the AmmaBoard boot-ten instruction set: opcodes, encodings,
// and decode helpers. Encoding is fixed 32-bit words:
//
//	[ opcode:5 | rd:3 | rs:3 | rt:3 | immediate:18 ]
//
// byte 0 = verb + destination register. Spare/reserved bits must be zero; the
// machine faults on nonzero reserved bits (it refuses nonsense).
package isa

import "fmt"

// Opcode is a 5-bit verb.
type Opcode uint8

// The boot ten (AmmaBoard-1). Names are plain-language; see PLANNING.md.
const (
	OpLoadWord       Opcode = iota + 1 // load_word  rd, imm(rs)
	OpStoreWord                        // store_word rt, imm(rs)
	OpAdd                              // add        rd, rs, rt
	OpAddNumber                        // add_number rd, rs, imm
	OpSubtract                         // subtract   rd, rs, rt
	OpJumpIfEqual                      // jump_if_equal   rs, rt, imm
	OpJumpIfNotEqual                   // jump_if_not_equal rs, rt, imm
	OpCall                             // call imm   (return address -> r7)
	OpGoto                             // goto rs    (pc = r[rs])
	OpHalt                             // halt
)

var names = map[Opcode]string{
	OpLoadWord:       "load_word",
	OpStoreWord:      "store_word",
	OpAdd:            "add",
	OpAddNumber:      "add_number",
	OpSubtract:       "subtract",
	OpJumpIfEqual:    "jump_if_equal",
	OpJumpIfNotEqual: "jump_if_not_equal",
	OpCall:           "call",
	OpGoto:           "goto",
	OpHalt:           "halt",
}

// Name returns the mnemonic for an opcode.
func (op Opcode) Name() string { return names[op] }

// Field positions and widths.
const (
	OpShift  = 27 // opcode:  bits 31..27
	RdShift  = 24 // rd:      bits 26..24
	RsShift  = 21 // rs:      bits 23..21
	RtShift  = 18 // rt:      bits 20..18
	ImmShift = 0  // imm:     bits 17..0 (18 bits, two's complement)

	ImmMask = 1<<18 - 1
	RegMask = 1<<3 - 1
	OpMask  = 1<<5 - 1
)

// Instruction is one decoded machine word.
type Instruction struct {
	Op   Opcode
	Rd   uint8
	Rs   uint8
	Rt   uint8
	Imm  int32 // sign-extended 18-bit immediate
	Word uint32
}

// Encode packs fields into a machine word. Reserved bits must be zero.
func Encode(op Opcode, rd, rs, rt uint8, imm int32) (uint32, error) {
	if imm < -(1<<17) || imm > (1<<18)-1 {
		return 0, fmt.Errorf("immediate %d out of 18-bit range", imm)
	}
	if rd > RegMask || rs > RegMask || rt > RegMask {
		return 0, fmt.Errorf("register number out of range")
	}
	w := uint32(op&OpMask)<<OpShift |
		uint32(rd&RegMask)<<RdShift |
		uint32(rs&RegMask)<<RsShift |
		uint32(rt&RegMask)<<RtShift |
		uint32(imm&ImmMask)<<ImmShift
	return w, nil
}

// MustEncode is Encode for tables and tests; it panics on invalid input.
func MustEncode(op Opcode, rd, rs, rt uint8, imm int32) uint32 {
	w, err := Encode(op, rd, rs, rt, imm)
	if err != nil {
		panic(err)
	}
	return w
}

// Decode unpacks a machine word. It returns an error if the opcode is unknown
// or any reserved (out-of-range) bits are set — the machine refuses nonsense.
func Decode(word uint32) (Instruction, error) {
	op := Opcode(word >> OpShift & OpMask)
	if _, ok := names[op]; !ok {
		return Instruction{}, fmt.Errorf("unknown opcode %d in word 0x%08x", op, word)
	}
	in := Instruction{
		Op:   op,
		Rd:   uint8(word >> RdShift & RegMask),
		Rs:   uint8(word >> RsShift & RegMask),
		Rt:   uint8(word >> RtShift & RegMask),
		Imm:  signExtend18(uint32(word) & ImmMask),
		Word: word,
	}
	// Reserved-bit check: opcode field is 5 bits inside bits 31..27; bits
	// 18..17 above the immediate belong to fields, so the only illegal
	// configuration is an opcode whose top bits overflow the 5-bit space —
	// already covered. Imm is always 18 bits; nothing is reserved.
	return in, nil
}

// signExtend18 converts an 18-bit two's-complement field to int32.
func signExtend18(v uint32) int32 {
	if v&(1<<17) != 0 { // NB: parens required — & and << share precedence in Go
		return int32(v | ^uint32(ImmMask))
	}
	return int32(v)
}
