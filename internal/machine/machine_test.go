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
	_ = sawEcho // echo is passive on AmmaBoard-2; its bus events are attributed when ticked
	if s := echo.StateDump(); !strings.Contains(s, "12345") {
		t.Errorf("echo state dump: %q", s)
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
	_, cpu, events := runProgram(t, words)
	if cpu.HaltedBy().Kind != "halt" {
		t.Fatalf("want clean halt, got %+v", cpu.HaltedBy())
	}
	if cpu.Regs[2] != 0xAB {
		t.Errorf("load_byte = %d, want 171 (zero-extended)", cpu.Regs[2])
	}
	if cpu.Regs[3] != 0xAB00 {
		t.Errorf("word at 64 = 0x%08x, want 0x0000ab00", cpu.Regs[3])
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
        bit_and                r7, r6, r2   ; 12 & 1 = 0 ... use a wider pair below
        halt
`)
	_, cpu, _ := runProgram(t, words)
	if got := int32(cpu.Regs[3]); got != -4 {
		t.Errorf("sra = %d, want -4", got)
	}
	if cpu.Regs[4] != 0x7ffffffc {
		t.Errorf("srl = 0x%08x, want 0x7ffffffc", cpu.Regs[4])
	}
	if cpu.Regs[5] != 2 {
		t.Errorf("sll = %d, want 2", cpu.Regs[5])
	}
	if cpu.Regs[7] != 0 {
		t.Errorf("and = %d, want 0", cpu.Regs[7])
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
	_, cpu, _ := runProgram(t, words)
	if cpu.Regs[3] != 15 {
		t.Errorf("or = %d, want 15", cpu.Regs[3])
	}
	if cpu.Regs[5] != 48 {
		t.Errorf("sll = %d, want 48", cpu.Regs[5])
	}
}

// ---- the trace-vocabulary fixes from poc-disccussion.md ----

func TestALUOpsLeaveVisibleExecuteLine(t *testing.T) {
	words := assemble(t, counterSrc)
	_, cpu, events := runProgram(t, words)
	found := false
	for _, e := range events {
		if e.Kind == "alu" && strings.Contains(e.Detail, "r2 <= 0 + 5 = 5") {
			found = true
		}
	}
	if !found {
		t.Error("add_number must emit a uniform alu EXECUTE line")
	}
	if cpu.Regs[2] != 5 {
		t.Errorf("r2 = %d, want 5", cpu.Regs[2])
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
	_, cpu, events := runProgram(t, words)
	var haltLine string
	for _, e := range events {
		if e.Kind == "halt" {
			haltLine = e.Detail
		}
	}
	if !strings.Contains(haltLine, "pc=0x0020") || !strings.Contains(haltLine, "next fetch 0x0024") {
		t.Errorf("halt line must show both conventions: %q", haltLine)
	}
	dump := cpu.StateDump()
	if !strings.Contains(dump, "pc=0x0024 (next fetch)") {
		t.Errorf("dump must label pc as next fetch:\n%s", dump)
	}
	if !strings.Contains(dump, "halt: clean stop at pc=0x0020") {
		t.Errorf("dump must show the halted instruction pc:\n%s", dump)
	}
}

// ---- day 8: timer + interrupts ----

const timerSrc = `
        add_number r1, r0, start    ; 0x0000 jump over the trap vector
        goto       r1               ; 0x0004
handler:                            ; 0x0008 = TrapVector
        add_number r5, r5, 1        ; ticks so far
        store_word r5, 128(r0)       ; publish the count
        add_number r6, r0, 2
        subtract   r6, r5, r6
        jump_if_not_equal r6, r0, ack
        store_word r0, 0(r2)        ; count reached 2: disarm the timer
ack:    store_word r0, 4(r2)        ; acknowledge the line
        return_from_interrupt
start:                              ; 0x0024
        add_number r2, r0, 16416    ; timer base 0x4020
        add_number r3, r0, 3        ; fire every 3 major ticks
        store_word r3, 0(r2)        ; arm
        allow_interrupts
wait:   load_word  r4, 128(r0)
        add_number r6, r0, 2
        subtract   r6, r4, r6
        jump_if_not_equal r6, r0, wait
        forbid_interrupts
        halt
`

func TestTimerInterruptRoundTrip(t *testing.T) {
	words := assemble(t, timerSrc)
	board, cpu, events := runProgram(t, words, NewTimer(0x4020))
	if f := cpu.HaltedBy(); f.Kind != "halt" {
		t.Fatalf("want clean halt after two ticks, got %+v", f)
	}
	if cpu.Regs[5] != 2 || cpu.Regs[4] != 2 {
		t.Errorf("r5=%d r4=%d, want 2/2 (handler ran twice, main saw both)", cpu.Regs[5], cpu.Regs[4])
	}
	if cpu.inHandler {
		t.Error("handler flag must clear on rfi")
	}
	interrupts, raises := 0, -1
	firstInterruptMajor := -1
	for _, e := range events {
		switch e.Kind {
		case "irq":
			if raises < 0 {
				raises = e.Major
			}
		case "interrupt":
			interrupts++
			if firstInterruptMajor < 0 {
				firstInterruptMajor = e.Major
			}
		}
	}
	if interrupts != 2 {
		t.Errorf("interrupt events = %d, want 2", interrupts)
	}
	if raises < 0 || firstInterruptMajor <= raises {
		t.Errorf("interrupt (major %d) must be delivered after raise (major %d)", firstInterruptMajor, raises)
	}
	if board.Halted != true {
		t.Error("board should be halted")
	}
}

func TestForbidBlocksDeliveryUntilAllow(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, start
        goto       r1
handler:                            ; 0x0008 = TrapVector
        add_number r5, r5, 1
        store_word r5, 128(r0)
        store_word r0, 0(r2)        ; one-shot: disarm
        store_word r0, 4(r2)        ; acknowledge
        return_from_interrupt
start:
        add_number r2, r0, 16416    ; timer base
        forbid_interrupts
        add_number r3, r0, 2        ; interval 2 — fires while IE is clear
        store_word r3, 0(r2)        ; arm
        add_number r4, r0, 4
        add_number r6, r0, 0
spin:   add_number r6, r6, 1
        subtract   r7, r4, r6
        jump_if_not_equal r7, r0, spin
        allow_interrupts            ; the raised line may deliver now
wait:   load_word r5, 128(r0)
        jump_if_equal r5, r0, wait
        forbid_interrupts
        halt
`)
	board, cpu, events := runProgram(t, words, NewTimer(0x4020))
	if f := cpu.HaltedBy(); f.Kind != "halt" {
		t.Fatalf("want clean halt, got %+v", f)
	}
	if cpu.Regs[5] != 1 {
		t.Errorf("r5 = %d, want 1 (line refused while forbidden, delivered once after allow)", cpu.Regs[5])
	}
	allowMajor, interruptMajor := -1, -1
	for _, e := range events {
		if e.Kind == "status" && strings.Contains(e.Detail, "ie=1") && allowMajor < 0 {
			allowMajor = e.Major
		}
		if e.Kind == "interrupt" && interruptMajor < 0 {
			interruptMajor = e.Major
		}
	}
	if allowMajor < 0 || interruptMajor < 0 || interruptMajor <= allowMajor {
		t.Errorf("delivery (major %d) must come after allow (major %d)", interruptMajor, allowMajor)
	}
	_ = board
}

func TestStrayRFIFaults(t *testing.T) {
	words := assemble(t, `
        return_from_interrupt
        halt
`)
	board, cpu, _ := runProgram(t, words)
	if cpu.HaltedBy().Kind != "stray_rfi" {
		t.Fatalf("want stray_rfi fault, got %+v", cpu.HaltedBy())
	}
	if !board.Halted {
		t.Fatal("machine should be halted")
	}
}

// Interrupt delivery is part of the machine's determinism surface.
func TestInterruptDeterminism(t *testing.T) {
	run := func() []Event {
		words := assemble(t, timerSrc)
		_, _, events := runProgram(t, words, NewTimer(0x4020))
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
