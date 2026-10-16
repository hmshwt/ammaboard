// Package isa defines the AmmaBoard ISA: opcodes, encodings, and decode
// helpers. Encoding is fixed 32-bit words:
//
//	[ opcode:5 | rd:3 | rs:3 | rt:3 | immediate:18 ]
//
// byte 0 = verb + destination register. Spare/reserved bits must be zero; the
// machine faults on nonzero reserved bits (it refuses nonsense).
package isa

import "fmt"

// Opcode is a 5-bit verb.
type Opcode uint8

// Stage 1 "boot ten" (AmmaBoard-1) and stage 2 (AmmaBoard-2). Names are
// plain-language; opcode order follows poc.md §1. See PLANNING.md.
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
	// Stage 2 (days 7–8): bytes, shifts, bitwise, interrupts.
	OpLoadByte             // load_byte  rd, imm(rs)   (zero-extended)
	OpStoreByte            // store_byte rt, imm(rs)   (low byte)
	OpShiftLeft            // shift_left            rd, rs, rt (amount = rt&31)
	OpShiftRight           // shift_right           rd, rs, rt (logical)
	OpShiftRightArithmetic // shift_right_arithmetic rd, rs, rt
	OpBitAnd               // bit_and rd, rs, rt
	OpBitOr                // bit_or  rd, rs, rt
	OpAllowInterrupts      // allow_interrupts        (zero-operand)
	OpForbidInterrupts     // forbid_interrupts       (zero-operand)
	OpReturnFromInterrupt  // return_from_interrupt   (zero-operand)
)

var names = map[Opcode]string{
	OpLoadWord:             "load_word",
	OpStoreWord:            "store_word",
	OpAdd:                  "add",
	OpAddNumber:            "add_number",
	OpSubtract:             "subtract",
	OpJumpIfEqual:          "jump_if_equal",
	OpJumpIfNotEqual:       "jump_if_not_equal",
	OpCall:                 "call",
	OpGoto:                 "goto",
	OpHalt:                 "halt",
	OpLoadByte:             "load_byte",
	OpStoreByte:            "store_byte",
	OpShiftLeft:            "shift_left",
	OpShiftRight:           "shift_right",
	OpShiftRightArithmetic: "shift_right_arithmetic",
	OpBitAnd:               "bit_and",
	OpBitOr:                "bit_or",
	OpAllowInterrupts:      "allow_interrupts",
	OpForbidInterrupts:     "forbid_interrupts",
	OpReturnFromInterrupt:  "return_from_interrupt",
}

// zeroOperandOps take no fields: rd/rs/rt/imm are reserved and must be zero —
// the machine refuses nonsense (the documented teachable fork).
var zeroOperandOps = map[Opcode]bool{
	OpAllowInterrupts:     true,
	OpForbidInterrupts:    true,
	OpReturnFromInterrupt: true,
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
	if zeroOperandOps[op] && (rd != 0 || rs != 0 || rt != 0 || imm != 0) {
		return 0, fmt.Errorf("%s takes no operands: reserved fields must be zero", op.Name())
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
	// Zero-operand (system) ops are the exception: their fields ARE reserved
	// and must be zero.
	if zeroOperandOps[op] && (in.Rd != 0 || in.Rs != 0 || in.Rt != 0 || uint32(in.Imm)&ImmMask != 0) {
		return Instruction{}, fmt.Errorf("%s: reserved fields must be zero in word 0x%08x", op.Name(), word)
	}
	return in, nil
}

// signExtend18 converts an 18-bit two's-complement field to int32.
func signExtend18(v uint32) int32 {
	if v&(1<<17) != 0 { // NB: parens required — & and << share precedence in Go
		return int32(v | ^uint32(ImmMask))
	}
	return int32(v)
}
