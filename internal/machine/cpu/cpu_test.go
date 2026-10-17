package cpu

import (
	"encoding/binary"
	"fmt"
	"testing"

	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

// fakeEnv is a minimal broker for component tests: 1KB of zeroed memory, an
// event recorder, and a stop latch. Writes land immediately (the latch
// semantics are the broker's contract, tested in package machine_test) —
// enough to run any single instruction in isolation.
type fakeEnv struct {
	mem    []byte
	events []string
	stop   *machine.Fault
}

func newFakeEnv() *fakeEnv { return &fakeEnv{mem: make([]byte, 1024)} }

func (f *fakeEnv) ReadBus(addr uint32) (uint32, machine.BusStatus) {
	if uint64(addr)+4 > uint64(len(f.mem)) {
		return 0, machine.BusNotConnected
	}
	return binary.LittleEndian.Uint32(f.mem[addr:]), machine.BusOK
}

func (f *fakeEnv) WriteBus(addr, val uint32, _ machine.WriteSite) {
	if uint64(addr)+4 <= uint64(len(f.mem)) {
		binary.LittleEndian.PutUint32(f.mem[addr:], val)
	}
}

func (f *fakeEnv) ReadBusByte(addr uint32) (uint8, machine.BusStatus) {
	if uint64(addr) >= uint64(len(f.mem)) {
		return 0, machine.BusNotConnected
	}
	return f.mem[addr], machine.BusOK
}

func (f *fakeEnv) WriteBusByte(addr uint32, val uint8, _ machine.WriteSite) {
	if uint64(addr) < uint64(len(f.mem)) {
		f.mem[addr] = val
	}
}

func (f *fakeEnv) RaiseInterrupt(string) {}

func (f *fakeEnv) Logf(kind, format string, args ...any) {
	f.events = append(f.events, kind+": "+fmt.Sprintf(format, args...))
}

func (f *fakeEnv) Halt(flt machine.Fault) {
	if f.stop == nil {
		c := flt
		f.stop = &c
	}
}

// CleanHalt records a clean stop (kind "halt", no fault data).
func (f *fakeEnv) CleanHalt() {
	if f.stop == nil {
		f.stop = &machine.Fault{Kind: "halt"}
	}
}

// run loads words at 0 and steps fetch/decode/execute until the CPU stops.
func run(t *testing.T, words []uint32) (*CPU, *fakeEnv) {
	t.Helper()
	c := New(0)
	env := newFakeEnv()
	for i, w := range words {
		binary.LittleEndian.PutUint32(env.mem[4*i:], w)
	}
	for i := 0; i < 1000 && !c.Halted; i++ {
		c.Tick(machine.PhaseFetch, env)
		c.Tick(machine.PhaseDecode, env)
		c.Tick(machine.PhaseExecute, env)
	}
	if !c.Halted {
		t.Fatal("program did not halt within 1000 cycles")
	}
	return c, env
}

func enc(op isa.Opcode, rd, rs, rt uint8, imm int32) uint32 {
	return isa.MustEncode(op, rd, rs, rt, imm)
}

// One table-driven harness for representative instructions: register effects,
// branch targets, and the hardwired-zero rule. Every instruction in the
// machine is additionally covered end-to-end in package machine_test.
func TestInstructionTable(t *testing.T) {
	halt := enc(isa.OpHalt, 0, 0, 0, 0)
	cases := []struct {
		name  string
		words []uint32
		want  map[uint8]uint32
	}{
		{
			name: "ALU and immediates",
			words: []uint32{
				enc(isa.OpAddNumber, 1, 0, 0, 24),
				enc(isa.OpAddNumber, 2, 0, 0, 6),
				enc(isa.OpAdd, 3, 1, 2, 0),        // 30
				enc(isa.OpSubtract, 4, 3, 2, 0),   // 24
				enc(isa.OpBitAnd, 5, 3, 2, 0),     // 30 & 6 = 6
				enc(isa.OpBitOr, 6, 1, 2, 0),      // 30
				enc(isa.OpShiftRight, 7, 1, 2, 0), // 24 >> 6 = 0
				halt,
			},
			want: map[uint8]uint32{1: 24, 2: 6, 3: 30, 4: 24, 5: 6, 6: 30, 7: 0},
		},
		{
			name: "r0 is hardwired zero",
			words: []uint32{
				enc(isa.OpAddNumber, 0, 0, 0, 99),
				enc(isa.OpAdd, 1, 0, 0, 0),
				halt,
			},
			want: map[uint8]uint32{0: 0, 1: 0},
		},
		{
			name: "branch not taken falls through",
			words: []uint32{
				enc(isa.OpAddNumber, 1, 0, 0, 1),
				enc(isa.OpJumpIfEqual, 0, 1, 0, 8), // 1 != 0: not taken, falls through
				enc(isa.OpAddNumber, 2, 0, 0, 7),
				halt,
			},
			want: map[uint8]uint32{1: 1, 2: 7},
		},
		{
			name: "call links in r7, goto returns",
			words: []uint32{
				enc(isa.OpCall, 0, 0, 0, 8), // -> word 3
				halt,                        // word 1: never reached
				enc(isa.OpHalt, 0, 0, 0, 0),
				enc(isa.OpAddNumber, 3, 0, 0, 5),
				enc(isa.OpGoto, 0, 7, 0, 0), // back to word 1 (halt)
			},
			want: map[uint8]uint32{3: 5, 7: 4},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := run(t, tc.words)
			for reg, want := range tc.want {
				if c.Regs[reg] != want {
					t.Errorf("r%d = %d, want %d", reg, c.Regs[reg], want)
				}
			}
		})
	}
}

// A taken interrupt parks pc + status, forbids nesting, and aims at the
// vector; return_from_interrupt restores both and clears the handler flag.
func TestTakeInterruptAndReturn(t *testing.T) {
	c := New(0)
	env := newFakeEnv()
	// Trap vector 0x8 holds: add_number r5, r5, 1 ; return_from_interrupt
	binary.LittleEndian.PutUint32(env.mem[0x8:], enc(isa.OpAddNumber, 5, 5, 0, 1))
	binary.LittleEndian.PutUint32(env.mem[0xc:], enc(isa.OpReturnFromInterrupt, 0, 0, 0, 0))
	c.Status |= StatusIE

	taken, fromPC, fromStatus, toStatus := c.TakeInterrupt(0x8)
	if !taken || fromPC != 0 || fromStatus&StatusIE == 0 || toStatus&StatusIE != 0 {
		t.Fatalf("take: taken=%v fromPC=%x from=%x to=%x", taken, fromPC, fromStatus, toStatus)
	}
	if c.PC != 0x8 || !c.inHandler {
		t.Fatalf("after take: pc=%x inHandler=%v", c.PC, c.inHandler)
	}
	for _, phase := range []machine.Phase{machine.PhaseFetch, machine.PhaseDecode, machine.PhaseExecute} {
		c.Tick(phase, env)
	}
	for _, phase := range []machine.Phase{machine.PhaseFetch, machine.PhaseDecode, machine.PhaseExecute} {
		c.Tick(phase, env)
	}
	if c.Regs[5] != 1 {
		t.Errorf("handler body ran %d time(s), want 1", c.Regs[5])
	}
	if c.PC != 0 || c.Status&StatusIE == 0 || c.inHandler {
		t.Errorf("after rfi: pc=%x status=%x inHandler=%v", c.PC, c.Status, c.inHandler)
	}
}

// faultOf returns the CPU's terminal fault record, failing if none exists.
func faultOf(t *testing.T, get func() *machine.Fault) machine.Fault {
	t.Helper()
	f := get()
	if f == nil {
		t.Fatal("expected a fault record, but nothing faulted")
	}
	return *f
}

// The fault plumbing is first-class from day one: an unaligned load halts the
// CPU with an alignment record, not a panic.
func TestAlignmentFault(t *testing.T) {
	c, _ := run(t, []uint32{
		enc(isa.OpAddNumber, 1, 0, 0, 66),
		enc(isa.OpLoadWord, 2, 1, 0, 0),
		enc(isa.OpHalt, 0, 0, 0, 0),
	})
	f := faultOf(t, c.Fault)
	if f.Kind != "alignment" || f.Addr != 66 || f.Phase != machine.PhaseExecute {
		t.Fatalf("want alignment fault at 66, got %+v", f)
	}
}

// A fresh CPU is not halted and has no fault record. Budget exhaustion is the
// board's concept (where the cycle cap lives), not the CPU's.
func TestFreshCPUIsNotHalted(t *testing.T) {
	c := New(0)
	if c.Halted || c.Fault() != nil {
		t.Errorf("fresh CPU: halted=%v fault=%v", c.Halted, c.Fault())
	}
}
