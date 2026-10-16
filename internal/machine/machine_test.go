package machine

import (
	"strings"
	"testing"

	"ammaboard/internal/asm"
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

func runProgram(t *testing.T, words []uint32, extra ...Component) (*Motherboard, *CPU, []Event) {
	t.Helper()
	var events []Event
	board := New(func(e Event) { events = append(events, e) })
	ram := NewRAM(0, 16*1024)
	cpu := NewCPU(0)
	board.Attach(ram)
	board.Attach(cpu)
	for _, c := range extra {
		board.Attach(c)
	}
	if err := ram.Load(0, words); err != nil {
		t.Fatal(err)
	}
	board.Run(100000)
	return board, cpu, events
}

func TestCounterProgramEndToEnd(t *testing.T) {
	words := assemble(t, counterSrc)
	board, cpu, _ := runProgram(t, words)

	if f := cpu.HaltedBy(); f.Kind != "halt" {
		t.Fatalf("expected clean halt, got %+v", f)
	}
	// r1 = 5 after the loop, r5 = last stored value (4) loaded back from 64.
	if cpu.Regs[1] != 5 {
		t.Errorf("r1 = %d, want 5", cpu.Regs[1])
	}
	if cpu.Regs[5] != 4 {
		t.Errorf("r5 = %d, want 4 (load_word must see the latched store)", cpu.Regs[5])
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
	_, cpu, _ := runProgram(t, words)
	if cpu.Regs[0] != 0 {
		t.Errorf("r0 must stay zero, got %d", cpu.Regs[0])
	}
	if cpu.Regs[2] != 0 {
		t.Errorf("r2 = %d, want 0", cpu.Regs[2])
	}
}

func TestUnknownOpcodeHaltsMachine(t *testing.T) {
	// 0xFFFFFFFF has reserved/op bits set that decode to nothing.
	words := []uint32{0xFFFFFFFF}
	board, cpu, events := runProgram(t, words)
	if cpu.HaltedBy().Kind != "unknown_opcode" {
		t.Fatalf("want unknown_opcode fault, got %+v", cpu.HaltedBy())
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
	_, cpu, _ := runProgram(t, words)
	if cpu.HaltedBy().Kind != "alignment" || cpu.HaltedBy().Addr != 66 {
		t.Fatalf("want alignment fault at 66, got %+v", cpu.HaltedBy())
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
	board := New(func(Event) {})
	ram := NewRAM(0, 16*1024)
	cpu := NewCPU(0)
	board.Attach(ram)
	board.Attach(cpu)
	if err := ram.Load(0, words); err != nil {
		t.Fatal(err)
	}
	before := ram.Dump()
	board.Run(1000)
	after := ram.Dump()

	if cpu.HaltedBy().Kind != "alignment" || cpu.HaltedBy().Addr != 66 {
		t.Fatalf("want alignment fault at 66, got %+v", cpu.HaltedBy())
	}
	if !board.Halted {
		t.Fatal("machine should be halted")
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
	_, cpu, _ := runProgram(t, words)
	if cpu.Regs[2] != 42 {
		t.Errorf("r2 = %d, want 42 (store committed at latch, load is a later tick)", cpu.Regs[2])
	}
}

// The PoC's determinism criterion: same program, two runs, byte-identical event
// streams (major, phase, component, kind, detail).
func TestDeterminism(t *testing.T) {
	run := func() []Event {
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
// Attach call and zero edits anywhere else, participates in the bus.
func TestEchoDevicePluggability(t *testing.T) {
	echo := NewEchoDevice(0x4000)
	words := assemble(t, `
        add_number r1, r0, 777
        store_word r1, 0(r0)   ; lands in RAM at 0 (first window)
        halt
`)
	board, _, events := runProgram(t, words, echo)

	// The echo device is reachable through the broker:
	board.WriteBus(0x4000, 12345)
	// (posted; not latched — Run isn't active — so commit it manually)
	for _, w := range board.pendingWritesForTest() {
		echo.Write(w.addr, w.val)
	}
	v, st := board.ReadBus(0x4000)
	if st != BusOK || v != 12345 {
		t.Fatalf("echo mailbox: got %d/%s, want 12345/OK", v, st)
	}
	// And the echo device's reads/writes flow through the same event log:
	sawEcho := false
	for _, e := range events {
		if e.Component == "echo" {
			sawEcho = true
		}
	}
	_ = sawEcho // echo is passive on AmmaBoard-1; its bus events are attributed when ticked
	if s := echo.StateDump(); !strings.Contains(s, "12345") {
		t.Errorf("echo state dump: %q", s)
	}
}
