// Integration tests: the whole board — broker + cpu + ram + devices — driven
// through the assembler. Component-level tests live in each component's package.
package machine_test

import (
	"strings"
	"testing"

	"ammaboard/internal/asm"
	"ammaboard/internal/machine"
	"ammaboard/internal/machine/cpu"
	"ammaboard/internal/machine/echo"
	"ammaboard/internal/machine/ram"
)

const counterSrc = `
        add_number r1, r0, 0
        add_number r2, r0, 5
        add_number r3, r0, 64
loop:   store_word r1, 0(r3)
        add_number r1, r1, 1
        subtract   r4, r2, r1
        jump_if_not_equal r4, r0, loop
        load_word  r5, 0(r3)
        halt
`

func assemble(t *testing.T, src string) []uint32 {
	t.Helper()
	words, err := asm.Assemble(src)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	return words
}

func runProgram(t *testing.T, words []uint32, extra ...machine.Component) (*machine.Motherboard, *cpu.CPU, []machine.Event) {
	t.Helper()
	var events []machine.Event
	board := machine.New(func(e machine.Event) { events = append(events, e) })
	mem := ram.New(0, 16*1024)
	proc := cpu.New(0)
	board.Attach(mem)
	board.Attach(proc)
	for _, c := range extra {
		board.Attach(c)
	}
	if err := mem.Load(0, words); err != nil {
		t.Fatal(err)
	}
	board.Run(100000)
	return board, proc, events
}

// checkHalt fails unless the CPU stopped cleanly (no fault record).
func checkHalt(t *testing.T, proc *cpu.CPU) {
	t.Helper()
	if !proc.Halted || proc.Fault() != nil {
		t.Fatalf("want clean halt, halted=%v fault=%+v", proc.Halted, proc.Fault())
	}
}

// checkBoardHalt fails unless the board stopped cleanly.
func checkBoardHalt(t *testing.T, board *machine.Motherboard) {
	t.Helper()
	if got := board.StopReason(); got != machine.StopHalt {
		t.Fatalf("want clean halt, got %q (fault: %+v)", got, board.Fault())
	}
}

// faultOf returns the terminal fault record, failing the test if nothing
// faulted.
func faultOf(t *testing.T, get func() *machine.Fault) machine.Fault {
	t.Helper()
	f := get()
	if f == nil {
		t.Fatal("expected a fault record, but nothing faulted")
	}
	return *f
}

func TestCounterProgramEndToEnd(t *testing.T) {
	words := assemble(t, counterSrc)
	board, proc, _ := runProgram(t, words)

	checkHalt(t, proc)
	// r1 = 5 after the loop, r5 = last stored value (4) loaded back from 64.
	if proc.Regs[1] != 5 {
		t.Errorf("r1 = %d, want 5", proc.Regs[1])
	}
	if proc.Regs[5] != 4 {
		t.Errorf("r5 = %d, want 4 (load_word must see the latched store)", proc.Regs[5])
	}
	// 3 setup + 5 loop iterations × 4 majors + load + halt = 25.
	if board.Major != 25 {
		t.Errorf("major ticks = %d, want 25", board.Major)
	}
}

func TestR0IsHardwiredZero(t *testing.T) {
	words := assemble(t, `
        add_number r0, r0, 99
        add r2, r0, r0
        halt
`)
	_, proc, _ := runProgram(t, words)
	if proc.Regs[0] != 0 {
		t.Errorf("r0 must stay zero, got %d", proc.Regs[0])
	}
	if proc.Regs[2] != 0 {
		t.Errorf("r2 = %d, want 0", proc.Regs[2])
	}
}

func TestUnknownOpcodeHaltsMachine(t *testing.T) {
	// 0xFFFFFFFF has reserved/op bits set that decode to nothing.
	words := []uint32{0xFFFFFFFF}
	board, proc, events := runProgram(t, words)
	if faultOf(t, proc.Fault).Kind != "unknown_opcode" {
		t.Fatalf("want unknown_opcode fault, got %+v", proc.Fault())
	}
	if !board.Halted {
		t.Fatal("machine should be halted")
	}
	found := false
	for _, e := range events {
		if e.Kind == "fault" && strings.Contains(e.Detail, "unknown opcode") {
			found = true
		}
	}
	if !found {
		t.Error("fault event should be in the tick log")
	}
}

func TestUnalignedLoadHaltsWithAlignmentFault(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, 66
        load_word  r2, 0(r1)
        halt
`)
	_, proc, _ := runProgram(t, words)
	f := faultOf(t, proc.Fault)
	if f.Kind != "alignment" || f.Addr != 66 {
		t.Fatalf("want alignment fault at 66, got %+v", f)
	}
}

// Stores and reads must refuse nonsense symmetrically: an unaligned store
// halts the machine exactly like an unaligned load — it must never vanish.
func TestUnalignedStoreHaltsWithAlignmentFault(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, 66
        add_number r2, r0, 1234
        store_word r2, 0(r1)
        halt
`)
	board := machine.New(func(machine.Event) {})
	mem := ram.New(0, 16*1024)
	board.Attach(mem)
	board.Attach(cpu.New(0))
	if err := mem.Load(0, words); err != nil {
		t.Fatal(err)
	}
	before := mem.Dump()
	board.Run(1000)
	after := mem.Dump()

	f := faultOf(t, board.Fault)
	if f.Kind != "alignment" || f.Addr != 66 {
		t.Fatalf("want alignment fault at 66, got %+v", f)
	}
	if string(before) != string(after) {
		t.Fatal("unaligned store must not modify memory")
	}
}

func TestWritesVisibleOnlyAfterLatch(t *testing.T) {
	// Post a store and read the same address in later phases of the SAME tick:
	// the read must still see the old value (writes land at the latch).
	words := assemble(t, `
        add_number r1, r0, 42
        store_word r1, 0(r0)
        load_word  r2, 0(r0)
        halt
`)
	_, proc, _ := runProgram(t, words)
	if proc.Regs[2] != 42 {
		t.Errorf("r2 = %d, want 42 (store committed at latch, load is a later tick)", proc.Regs[2])
	}
}

// The PoC's determinism criterion: same program, two runs, byte-identical event
// streams (major, phase, component, kind, detail).
func TestDeterminism(t *testing.T) {
	run := func() []machine.Event {
		words := assemble(t, counterSrc)
		_, _, events := runProgram(t, words)
		return events
	}
	a, b := run(), run()
	if len(a) != len(b) {
		t.Fatalf("event count differs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("event %d differs:\n%+v\n%+v", i, a[i], b[i])
		}
	}
}

// The PoC's pluggability criterion: a brand-new component, wired on with one
// Attach call and zero edits anywhere else, participates in the bus — stores
// land in it at the latch, loads read back through the broker, and the commit
// names the device.
func TestEchoDevicePluggability(t *testing.T) {
	dev := echo.New(0x4000)
	words := assemble(t, `
        add_number r1, r0, 777
        add_number r2, r0, 16384    ; echo base 0x4000
        store_word r1, 0(r2)        ; posted, then latched into the mailbox
        load_word  r3, 0(r2)        ; ...and read back through the broker
        halt
`)
	board, proc, events := runProgram(t, words, dev)
	checkBoardHalt(t, board)
	checkHalt(t, proc)
	if proc.Regs[3] != 777 {
		t.Errorf("read back %d, want 777 (echo is just addresses)", proc.Regs[3])
	}
	if s := dev.StateDump(); !strings.Contains(s, "777") {
		t.Errorf("echo state dump: %q", s)
	}
	sawCommit := false
	for _, e := range events {
		if e.Kind == "latch" && strings.Contains(e.Detail, "commit echo[0x4000] <= 0x00000309") {
			sawCommit = true
		}
	}
	if !sawCommit {
		t.Error("latch commit must name the device that owns the address")
	}
}

// ---- stage 2: bytes, shifts, bitwise ----

func TestByteLoadStoreRoundTrip(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, 171      ; 0xAB
        store_byte r1, 65(r0)       ; unaligned on purpose: bytes need no alignment
        load_byte  r2, 65(r0)
        load_word  r3, 64(r0)       ; the byte must sit inside the word, little-endian
        halt
`)
	_, proc, events := runProgram(t, words)
	checkHalt(t, proc)
	if proc.Regs[2] != 0xAB {
		t.Errorf("load_byte = %d, want 171 (zero-extended)", proc.Regs[2])
	}
	if proc.Regs[3] != 0xAB00 {
		t.Errorf("word at 64 = 0x%08x, want 0x0000ab00", proc.Regs[3])
	}
	sawPost, sawRead, sawCommit := false, false, false
	for _, e := range events {
		switch {
		case e.Kind == "bus_write_byte_post":
			sawPost = true
		case e.Kind == "bus_read_byte":
			sawRead = true
		case e.Kind == "latch" && strings.Contains(e.Detail, "commit ram[0x0041] <= byte 0xab"):
			sawCommit = true
		}
	}
	if !sawPost || !sawRead || !sawCommit {
		t.Errorf("byte bus events missing: post=%v read=%v commit=%v", sawPost, sawRead, sawCommit)
	}
}

func TestShiftsAndBitwise(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, -8       ; 0xfffffff8
        add_number r2, r0, 1
        shift_right_arithmetic r3, r1, r2   ; sign-preserving: -4
        shift_right            r4, r1, r2   ; logical:      0x7ffffffc
        shift_left             r5, r2, r2   ; 1 << 1 = 2
        add_number r6, r0, 12
        bit_and                r7, r6, r2   ; 12 & 1 = 0
        halt
`)
	_, proc, _ := runProgram(t, words)
	if got := int32(proc.Regs[3]); got != -4 {
		t.Errorf("sra = %d, want -4", got)
	}
	if proc.Regs[4] != 0x7ffffffc {
		t.Errorf("srl = 0x%08x, want 0x7ffffffc", proc.Regs[4])
	}
	if proc.Regs[5] != 2 {
		t.Errorf("sll = %d, want 2", proc.Regs[5])
	}
	if proc.Regs[7] != 0 {
		t.Errorf("and = %d, want 0", proc.Regs[7])
	}
}

func TestBitOrAndShiftLeftValues(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, 12
        add_number r2, r0, 3
        bit_or     r3, r1, r2       ; 15
        add_number r4, r0, 4
        shift_left r5, r2, r4       ; 48
        halt
`)
	_, proc, _ := runProgram(t, words)
	if proc.Regs[3] != 15 {
		t.Errorf("or = %d, want 15", proc.Regs[3])
	}
	if proc.Regs[5] != 48 {
		t.Errorf("sll = %d, want 48", proc.Regs[5])
	}
}

// ---- trace vocabulary (frozen by decisions.md D13–D17, D26) ----

func TestALUOpsLeaveVisibleExecuteLine(t *testing.T) {
	words := assemble(t, counterSrc)
	_, proc, events := runProgram(t, words)
	found := false
	for _, e := range events {
		if e.Kind == "alu" && strings.Contains(e.Detail, "r2 <= 0 + 5 = 5") {
			found = true
		}
	}
	if !found {
		t.Error("add_number must emit a uniform alu EXECUTE line")
	}
	if proc.Regs[2] != 5 {
		t.Errorf("r2 = %d, want 5", proc.Regs[2])
	}
}

func TestLatchCommitsNameTheirTarget(t *testing.T) {
	words := assemble(t, counterSrc)
	_, _, events := runProgram(t, words)
	found := false
	for _, e := range events {
		if e.Kind == "latch" && strings.Contains(e.Detail, "commit ram[0x0040] <= 0x00000004") {
			found = true
		}
	}
	if !found {
		t.Error("latch commit must name the target component and value")
	}
}

func TestHaltLineAndDumpAgreeOnPC(t *testing.T) {
	words := assemble(t, counterSrc)
	_, proc, events := runProgram(t, words)
	var haltLine string
	for _, e := range events {
		if e.Kind == "halt" {
			haltLine = e.Detail
		}
	}
	if !strings.Contains(haltLine, "pc=0x0020") || !strings.Contains(haltLine, "next fetch 0x0024") {
		t.Errorf("halt line must show both conventions: %q", haltLine)
	}
	dump := proc.StateDump()
	if !strings.Contains(dump, "pc=0x0024 (next fetch)") {
		t.Errorf("dump must label pc as next fetch:\n%s", dump)
	}
	if !strings.Contains(dump, "halt: clean stop at pc=0x0020") {
		t.Errorf("dump must show the halted instruction pc:\n%s", dump)
	}
}
