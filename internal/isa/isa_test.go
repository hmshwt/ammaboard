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
}

func TestDecodeRefusesNonsense(t *testing.T) {
	if _, err := Decode(0xFFFFFFFF); err == nil {
		t.Error("garbage word should not decode")
	}
	if _, err := Decode(0); err == nil {
		t.Error("opcode 0 is not a verb; should not decode")
	}
}
