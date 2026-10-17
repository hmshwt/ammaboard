package isa

import "testing"

func TestEncodeDecodeRoundTrip(t *testing.T) {
	cases := []struct {
		op         Opcode
		rd, rs, rt uint8
		imm        int32
		want       uint32
		wantImm    int32
	}{
		{OpHalt, 0, 0, 0, 0, 0x50000000, 0}, // opcode 10 in bits 31..27
		{OpAdd, 1, 2, 3, 0, 0x194C0000, 0},  // op=3 rd=1 rs=2 rt=3
		{OpAddNumber, 4, 5, 0, -1, 0, -1},
		{OpLoadWord, 2, 3, 0, 64, 0, 64},
		{OpJumpIfNotEqual, 0, 1, 2, -8, 0, -8},
		{OpCall, 0, 0, 0, 44, 0, 44},
	}
	for _, c := range cases {
		w := MustEncode(c.op, c.rd, c.rs, c.rt, c.imm)
		if c.want != 0 && w != c.want {
			t.Errorf("%s encode: got 0x%08x want 0x%08x", c.op.Name(), w, c.want)
		}
		in, err := Decode(w)
		if err != nil {
			t.Fatalf("%s decode: %v", c.op.Name(), err)
		}
		if in.Op != c.op || in.Rd != c.rd || in.Rs != c.rs || in.Rt != c.rt || in.Imm != c.wantImm {
			t.Errorf("%s roundtrip: got %+v want op=%d rd=%d rs=%d rt=%d imm=%d",
				c.op.Name(), in, c.op, c.rd, c.rs, c.rt, c.wantImm)
		}
	}
}

func TestImmediateRangeAndSign(t *testing.T) {
	max := int32(1)<<17 - 1
	min := -(int32(1) << 17)
	w := MustEncode(OpAddNumber, 1, 0, 0, max)
	if in, _ := Decode(w); in.Imm != max {
		t.Errorf("max imm: got %d want %d", in.Imm, max)
	}
	w = MustEncode(OpAddNumber, 1, 0, 0, min)
	if in, _ := Decode(w); in.Imm != min {
		t.Errorf("min imm: got %d want %d", in.Imm, min)
	}
	if _, err := Encode(OpAddNumber, 1, 0, 0, min-1); err == nil {
		t.Error("out-of-range immediate should fail")
	}
	// The positive half is what sign extension breaks: 2^17 encodes in 18
	// bits but decodes as -2^17. It must be refused, not silently wrapped.
	if _, err := Encode(OpAddNumber, 1, 0, 0, 1<<17); err == nil {
		t.Error("2^17 is not representable in 18-bit two's complement; should fail")
	}
	if _, err := Encode(OpAddNumber, 1, 0, 0, (1<<18)-1); err == nil {
		t.Error("2^18-1 is not representable; should fail")
	}
	// Every accepted immediate must round-trip exactly.
	for _, imm := range []int32{min, -1, 0, 1, max, max - 1, min + 1} {
		w, err := Encode(OpAddNumber, 1, 0, 0, imm)
		if err != nil {
			t.Errorf("imm %d should encode: %v", imm, err)
			continue
		}
		if in, err := Decode(w); err != nil || in.Imm != imm {
			t.Errorf("imm %d did not round-trip: %+v, %v", imm, in, err)
		}
	}
}

func TestDecodeRefusesNonsense(t *testing.T) {
	if _, err := Decode(0xFFFFFFFF); err == nil {
		t.Error("garbage word should not decode")
	}
	if _, err := Decode(0); err == nil {
		t.Error("opcode 0 is not a verb; should not decode")
	}
}

// Zero-operand instructions (halt + the system ops) reserve their fields:
// must be zero in the encoder AND the decoder (a hand-rolled binary must
// fault too).
func TestSystemOpsRefuseNonzeroFields(t *testing.T) {
	ops := []Opcode{OpHalt, OpAllowInterrupts, OpForbidInterrupts, OpReturnFromInterrupt}
	for _, op := range ops {
		if _, err := Encode(op, 1, 0, 0, 0); err == nil {
			t.Errorf("%s encode with rd!=0 should fail", op.Name())
		}
		if _, err := Encode(op, 0, 0, 0, 1); err == nil {
			t.Errorf("%s encode with imm!=0 should fail", op.Name())
		}
		// Bypass Encode: forge a word with a nonzero reserved field.
		forged := uint32(op)<<OpShift | 1<<RdShift
		if _, err := Decode(forged); err == nil {
			t.Errorf("%s decode with rd!=0 should refuse nonsense", op.Name())
		}
		w := MustEncode(op, 0, 0, 0, 0)
		in, err := Decode(w)
		if err != nil || in.Op != op {
			t.Errorf("%s zero-operand roundtrip: %+v, %v", op.Name(), in, err)
		}
	}
}

// Stage-2 encodings round-trip: bytes (I-format), shifts/bitwise (R-format).
func TestStage2RoundTrip(t *testing.T) {
	in, err := Decode(MustEncode(OpLoadByte, 2, 3, 0, -1))
	if err != nil || in.Op != OpLoadByte || in.Rd != 2 || in.Rs != 3 || in.Imm != -1 {
		t.Errorf("load_byte roundtrip: %+v, %v", in, err)
	}
	in, err = Decode(MustEncode(OpStoreByte, 0, 4, 5, 7))
	if err != nil || in.Op != OpStoreByte || in.Rs != 4 || in.Rt != 5 || in.Imm != 7 {
		t.Errorf("store_byte roundtrip: %+v, %v", in, err)
	}
	in, err = Decode(MustEncode(OpShiftRightArithmetic, 1, 2, 3, 0))
	if err != nil || in.Op != OpShiftRightArithmetic || in.Rd != 1 || in.Rs != 2 || in.Rt != 3 {
		t.Errorf("sra roundtrip: %+v, %v", in, err)
	}
}
