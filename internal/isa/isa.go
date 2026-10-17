// Package isa defines the AmmaBoard ISA: opcodes, encodings, and decode
// helpers, plus Specs — the one table that describes each instruction
// (opcode, mnemonic, operand form) and keeps the assembler and the CPU in
// parity. Encoding is fixed 32-bit words:
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
// plain-language; opcode order and growth stages follow docs/isa.md.
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

// Form is an instruction's operand grammar — the shape people write and the
// shape the fields take. The assembler parses each mnemonic by its form; Encode
// and Decode treat FormZero's fields as reserved. Nothing else repeats the
// operand lists.
type Form uint8

const (
	FormZero   Form = iota // no operands: halt, allow_interrupts, …
	FormRRR                // rd, rs, rt         add, subtract, shifts, bitwise
	FormRRI                // rd, rs, imm        add_number
	FormLoad               // rd, imm(rs)        load_word, load_byte
	FormStore              // rt, imm(rs)        store_word, store_byte
	FormBranch             // rs, rt, target     jump_if_equal, jump_if_not_equal
	FormCall               // target             call
	FormGoto               // rs                 goto
)

// Spec is one instruction's full description: opcode, mnemonic, operand form.
// This table is the single source of truth that keeps assembler and CPU in
// parity — the assembler encodes by Form, the CPU registers its executor by
// Opcode, and a conformance test (parity_test.go) fails the build if any layer
// drifts. Adding an instruction = a row here + a file in package cpu + tests.
type Spec struct {
	Op       Opcode
	Mnemonic string
	Form     Form
}

// NumOpcodes is how many verbs the ISA defines; opcodes run 1..NumOpcodes.
const NumOpcodes = 20

// Specs lists every instruction, in opcode order.
var Specs = []Spec{
	{OpLoadWord, "load_word", FormLoad},
	{OpStoreWord, "store_word", FormStore},
	{OpAdd, "add", FormRRR},
	{OpAddNumber, "add_number", FormRRI},
	{OpSubtract, "subtract", FormRRR},
	{OpJumpIfEqual, "jump_if_equal", FormBranch},
	{OpJumpIfNotEqual, "jump_if_not_equal", FormBranch},
	{OpCall, "call", FormCall},
	{OpGoto, "goto", FormGoto},
	{OpHalt, "halt", FormZero},
	{OpLoadByte, "load_byte", FormLoad},
	{OpStoreByte, "store_byte", FormStore},
	{OpShiftLeft, "shift_left", FormRRR},
	{OpShiftRight, "shift_right", FormRRR},
	{OpShiftRightArithmetic, "shift_right_arithmetic", FormRRR},
	{OpBitAnd, "bit_and", FormRRR},
	{OpBitOr, "bit_or", FormRRR},
	{OpAllowInterrupts, "allow_interrupts", FormZero},
	{OpForbidInterrupts, "forbid_interrupts", FormZero},
	{OpReturnFromInterrupt, "return_from_interrupt", FormZero},
}

var (
	specByOp   = map[Opcode]Spec{}
	specByName = map[string]Spec{}
)

func init() {
	for _, s := range Specs {
		specByOp[s.Op] = s
		specByName[s.Mnemonic] = s
	}
}

// Lookup finds a Spec by mnemonic (lowercase; the assembler lowercases first).
func Lookup(mnemonic string) (Spec, bool) {
	s, ok := specByName[mnemonic]
	return s, ok
}

// SpecFor finds a Spec by opcode.
func SpecFor(op Opcode) (Spec, bool) {
	s, ok := specByOp[op]
	return s, ok
}

// Name returns the mnemonic for an opcode ("" if unknown).
func (op Opcode) Name() string { return specByOp[op].Mnemonic }

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
	// The immediate field is 18-bit two's complement: [-2^17, 2^17-1].
	// Accepting anything wider would not round-trip — Decode sign-extends
	// bit 17, so e.g. 2^17 would silently decode as -2^17.
	if imm < -(1<<17) || imm > (1<<17)-1 {
		return 0, fmt.Errorf("immediate %d out of 18-bit signed range", imm)
	}
	if rd > RegMask || rs > RegMask || rt > RegMask {
		return 0, fmt.Errorf("register number out of range")
	}
	// Zero-operand instructions reserve their fields: the machine refuses
	// nonsense. (halt joined this rule with the Specs table — previously its
	// fields were quietly ignored.)
	if s, ok := specByOp[op]; ok && s.Form == FormZero && (rd != 0 || rs != 0 || rt != 0 || imm != 0) {
		return 0, fmt.Errorf("%s takes no operands: reserved fields must be zero", s.Mnemonic)
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
	spec, ok := specByOp[op]
	if !ok {
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
	// Zero-operand (system + halt) ops are the exception: their fields ARE
	// reserved and must be zero.
	if spec.Form == FormZero && (in.Rd != 0 || in.Rs != 0 || in.Rt != 0 || uint32(in.Imm)&ImmMask != 0) {
		return Instruction{}, fmt.Errorf("%s: reserved fields must be zero in word 0x%08x", spec.Mnemonic, word)
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
