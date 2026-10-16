package asm

import (
	"testing"

	"ammaboard/internal/isa"
)

const counter = `
; classic loop: store 0..4 at 64..80, load the last one back
        add_number r1, r0, 0        ; i = 0
        add_number r2, r0, 5        ; n = 5
        add_number r3, r0, 64       ; base
loop:   store_word r1, 0(r3)
        add_number r1, r1, 1
        subtract   r4, r2, r1       ; r4 = n - i
        jump_if_not_equal r4, r0, loop
        load_word  r5, 16(r3)
        halt
`

func TestAssembleCounter(t *testing.T) {
	words, err := Assemble(counter)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 9 {
		t.Fatalf("got %d words, want 9", len(words))
	}
	// Word 6 is jump_if_not_equal r4, r0, loop.
	// loop = word 3; here = 6; offset = 12 - (24+4) = -16.
	in, err := isa.Decode(words[6])
	if err != nil {
		t.Fatal(err)
	}
	if in.Op != isa.OpJumpIfNotEqual || in.Rs != 4 || in.Rt != 0 || in.Imm != -16 {
		t.Errorf("branch word: got %+v, want rs=4 rt=0 imm=-16", in)
	}
	// Word 8 is halt.
	if in, _ := isa.Decode(words[8]); in.Op != isa.OpHalt {
		t.Errorf("last word should be halt, got %s", in.Op.Name())
	}
}

func TestCallAndGoto(t *testing.T) {
	src := `
        call sub1
        halt
sub1:   add_number r1, r0, 7
        goto r7
`
	words, err := Assemble(src)
	if err != nil {
		t.Fatal(err)
	}
	call, _ := isa.Decode(words[0])
	if call.Op != isa.OpCall || call.Imm != 4 { // sub1 at word 2: 2*4 - (0*4+4) = 4
		t.Errorf("call: got %+v (want imm=4)", call)
	}
	g, _ := isa.Decode(words[3])
	if g.Op != isa.OpGoto || g.Rs != 7 {
		t.Errorf("goto: got %+v (want rs=7)", g)
	}
}

func TestErrors(t *testing.T) {
	for _, bad := range []string{
		"frobnicate r1, r2",             // unknown mnemonic
		"add r9, r1, r2",                // register out of range
		"load_word r1, 2(r1)",           // unaligned offset is fine to ENCODE (faults at runtime)
		"jump_if_equal r1, r2, nowhere", // unknown label
		"add r1, r2",                    // missing operand
	} {
		if _, err := Assemble(bad); err == nil {
			// Note: load_word 2(r1) assembles fine; alignment is a runtime fault.
			if !contains(bad, "load_word") {
				t.Errorf("expected error for %q", bad)
			}
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
