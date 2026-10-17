// Package machine implements the AmmaBoard motherboard: an Elm-like, single-threaded,
// phase-stepped machine. The motherboard is the broker — components never reference
// each other, all interaction goes through ReadBus/WriteBus, and every interaction
// is logged. Same inputs + same cycle count = identical machine state, always.
package machine

import (
	"bytes"
	"fmt"
)

// MachineVersion names the machine generation: ISA stage, not app release
// (motherboard.md §2). AmmaBoard-2 = boot ten + bytes/shifts/interrupts.
const MachineVersion = "AmmaBoard-2"

// Phase is a named minor tick within one major tick (one machine cycle).
type Phase string

// The AmmaBoard-2 phase schedule. CPU phases; bus latches flip between phases.
const (
	PhaseFetch   Phase = "FETCH"
	PhaseDecode  Phase = "DECODE"
	PhaseExecute Phase = "EXECUTE"
)

// Schedule is the fixed minor-tick order within one major tick.
var Schedule = []Phase{PhaseFetch, PhaseDecode, PhaseExecute}

// BusStatus is what the broker answers with. Failures are data, not Go errors.
type BusStatus string

const (
	BusOK           BusStatus = "OK"
	BusNotConnected BusStatus = "NOT_CONNECTED"
	BusNotPermitted BusStatus = "NOT_PERMITTED" // e.g. unaligned word access
	BusIdle         BusStatus = "IDLE"
)

// Fault is first-class data: the machine's honest "I stopped, here is why".
type Fault struct {
	Kind  string // "unknown_opcode", "alignment", "halt"
	Addr  uint32
	PC    uint32
	Phase Phase
	Word  uint32
}

// Env is what a component gets each tick: the broker + the log + the power to
// stop the machine. Components never hold references to each other — only to
// this interface.
type Env interface {
	ReadBus(addr uint32) (uint32, BusStatus)
	WriteBus(addr uint32, val uint32)
	ReadBusByte(addr uint32) (uint8, BusStatus)
	WriteBusByte(addr uint32, val uint8)
	RaiseInterrupt(line string)
	Logf(kind, format string, args ...any)
	Halt(f Fault)
}

// Component is one device on the board. Tick is called once per phase, in the
// motherboard's fixed order. A tick must be synchronous and deterministic.
type Component interface {
	Name() string
	Tick(phase Phase, env Env)
}

// Reader is implemented by components that own an address window (e.g. RAM).
// The broker routes ReadBus calls for addresses inside Window to this component.
type Reader interface {
	Window() (base, size uint32)
	Read(addr uint32) (uint32, BusStatus)
}

// Writer is implemented by readable components that also accept latched writes.
type Writer interface {
	Write(addr uint32, val uint32)
}

// ByteReader is implemented by window owners that also serve byte-granular
// reads. A word-only device in whose window a byte access lands answers
// NOT_PERMITTED — the machine refuses nonsense, symmetrically.
type ByteReader interface {
	BusReadByte(addr uint32) (uint8, BusStatus)
}

// ByteWriter accepts latched byte-granular writes.
type ByteWriter interface {
	BusWriteByte(addr uint32, val uint8)
}

// interruptible is implemented by components that can take an interrupt at
// the board's latch edge (the CPU). The broker knows no details beyond this.
type interruptible interface {
	TakeInterrupt(vector uint32) (taken bool, fromPC, fromStatus uint32)
}

// pendingWrite is a value posted to the bus; it becomes visible to readers only
// at the phase latch (between phases). Everyone within a phase sees a stable bus.
type pendingWrite struct {
	addr uint32
	val  uint32
	size uint32 // 4 = word, 1 = byte
}

// Motherboard is the root component: phase schedule, broker, log.
type Motherboard struct {
	components []Component
	readers    []Reader // parallel to components
	irqTarget  interruptible
	irqs       []string // raised interrupt lines, in raise order (one wire per source)
	bus        struct {
		pending []pendingWrite
	}
	Major    int
	Halted   bool
	Fault    *Fault
	sink     func(Event)
	ticking  string // component currently in Tick
	curPhase Phase
}

// Event is one logged interaction.
type Event struct {
	Major     int
	Phase     string
	Component string
	Kind      string
	Detail    string
}

// New creates a motherboard. sink receives every event, in order. Components
// tick in the order given.
func New(sink func(Event)) *Motherboard {
	if sink == nil {
		sink = func(Event) {}
	}
	return &Motherboard{sink: sink}
}

// Attach wires a component onto the board. Order = tick order.
func (m *Motherboard) Attach(c Component) {
	m.components = append(m.components, c)
	if r, ok := c.(Reader); ok {
		m.readers = append(m.readers, r)
	} else {
		m.readers = append(m.readers, nil)
	}
	if t, ok := c.(interruptible); ok && m.irqTarget == nil {
		m.irqTarget = t // first interrupt-capable component services lines (one CPU for now)
	}
}

// Run steps up to maxMajors major ticks, or until halt/fault. It returns the
// final major number reached.
func (m *Motherboard) Run(maxMajors int) int {
	for i := 0; i < maxMajors && !m.Halted; i++ {
		m.Major++
		m.sink(Event{Major: m.Major, Component: "board", Kind: "major"})
		m.deliverInterrupts() // the seam: latch edge of the previous cycle, before the next fetch
		for _, phase := range Schedule {
			m.runPhase(phase)
			if m.Halted {
				break
			}
		}
		if !m.Halted {
			m.sink(Event{Major: m.Major, Component: "board", Kind: "latch", Detail: "end of major tick"})
		}
	}
	return m.Major
}

func (m *Motherboard) runPhase(phase Phase) {
	m.curPhase = phase
	for _, c := range m.components {
		m.ticking = c.Name()
		c.Tick(phase, m)
		if m.Halted {
			return
		}
	}
	m.ticking = "board"
	// Phase latch: posted writes become visible to everyone, one commit
	// event each — the trace shows not just THAT state changed but WHERE.
	for _, w := range m.bus.pending {
		m.commit(w)
	}
	m.bus.pending = m.bus.pending[:0]
}

func (m *Motherboard) commit(w pendingWrite) {
	for i, r := range m.readers {
		if r == nil {
			continue
		}
		base, size := r.Window()
		if w.addr < base || w.addr >= base+size {
			continue
		}
		name := m.components[i].Name()
		switch {
		case w.size == 1:
			if bw, ok := m.components[i].(ByteWriter); ok {
				detail := fmt.Sprintf("commit %s[0x%04x] <= byte 0x%02x", name, w.addr, uint8(w.val))
				if br, ok := m.components[i].(ByteReader); ok {
					if old, st := br.BusReadByte(w.addr); st == BusOK {
						detail += fmt.Sprintf(" (was 0x%02x)", old)
					}
				}
				bw.BusWriteByte(w.addr, uint8(w.val))
				m.sink(Event{Major: m.Major, Phase: string(m.curPhase), Component: "board",
					Kind: "latch", Detail: detail})
			}
		default:
			if wr, ok := m.components[i].(Writer); ok {
				detail := fmt.Sprintf("commit %s[0x%04x] <= 0x%08x", name, w.addr, w.val)
				if old, st := r.Read(w.addr); st == BusOK {
					detail += fmt.Sprintf(" (was 0x%08x)", old)
				}
				wr.Write(w.addr, w.val)
				m.sink(Event{Major: m.Major, Phase: string(m.curPhase), Component: "board",
					Kind: "latch", Detail: detail})
			}
		}
		return
	}
}

// deliverInterrupts hands at most one raised line to the CPU per cycle. A
// line refused (IE clear) stays raised and retries at the next edge.
func (m *Motherboard) deliverInterrupts() {
	if len(m.irqs) == 0 || m.irqTarget == nil {
		return
	}
	taken, fromPC, fromStatus := m.irqTarget.TakeInterrupt(TrapVector)
	if !taken {
		return
	}
	line := m.irqs[0]
	m.irqs = m.irqs[1:]
	m.sink(Event{Major: m.Major, Phase: string(PhaseFetch), Component: "board",
		Kind: "interrupt",
		Detail: fmt.Sprintf("%s: pc 0x%04x -> 0x%04x, status 0x%02x -> 0x%02x",
			line, fromPC, TrapVector, fromStatus, fromStatus&^StatusIE)})
}

// route finds the window owner for addr: (index, ok).
func (m *Motherboard) route(addr uint32) (int, bool) {
	for i, r := range m.readers {
		if r == nil {
			continue
		}
		base, size := r.Window()
		if addr >= base && addr < base+size {
			return i, true
		}
	}
	return 0, false
}

// ReadBus implements Env. Routes to the component whose window contains addr.
func (m *Motherboard) ReadBus(addr uint32) (uint32, BusStatus) {
	i, ok := m.route(addr)
	if !ok {
		m.event("bus_read", fmt.Sprintf("addr=0x%04x status=%s", addr, BusNotConnected))
		return 0, BusNotConnected
	}
	v, st := m.readers[i].Read(addr)
	m.event("bus_read", fmt.Sprintf("addr=0x%04x val=0x%08x status=%s", addr, v, st))
	return v, st
}

// WriteBus implements Env. Posts a write; visible after the phase latch.
func (m *Motherboard) WriteBus(addr uint32, val uint32) {
	m.event("bus_write_post", fmt.Sprintf("addr=0x%04x val=0x%08x", addr, val))
	m.bus.pending = append(m.bus.pending, pendingWrite{addr: addr, val: val, size: 4})
}

// ReadBusByte implements Env for byte-granular loads. A window owner without
// byte support answers NOT_PERMITTED rather than pretending.
func (m *Motherboard) ReadBusByte(addr uint32) (uint8, BusStatus) {
	i, ok := m.route(addr)
	if !ok {
		m.event("bus_read_byte", fmt.Sprintf("addr=0x%04x status=%s", addr, BusNotConnected))
		return 0, BusNotConnected
	}
	br, ok := m.components[i].(ByteReader)
	if !ok {
		m.event("bus_read_byte", fmt.Sprintf("addr=0x%04x status=%s", addr, BusNotPermitted))
		return 0, BusNotPermitted
	}
	b, st := br.BusReadByte(addr)
	m.event("bus_read_byte", fmt.Sprintf("addr=0x%04x val=0x%02x status=%s", addr, b, st))
	return b, st
}

// WriteBusByte implements Env: posts a byte write; visible after the latch.
func (m *Motherboard) WriteBusByte(addr uint32, val uint8) {
	m.event("bus_write_byte_post", fmt.Sprintf("addr=0x%04x val=0x%02x", addr, val))
	m.bus.pending = append(m.bus.pending, pendingWrite{addr: addr, val: uint32(val), size: 1})
}

// RaiseInterrupt implements Env: a component raises its line. Level-triggered —
// the line stays up until the device itself clears it (usually on MMIO ack).
func (m *Motherboard) RaiseInterrupt(line string) {
	for _, l := range m.irqs {
		if l == line {
			return
		}
	}
	m.irqs = append(m.irqs, line)
	m.event("irq", fmt.Sprintf("%s raised", line))
}

// Logf implements Env: route component events into the tick log.
func (m *Motherboard) Logf(kind, format string, args ...any) {
	m.event(kind, fmt.Sprintf(format, args...))
}

// event writes to the sink, attributing to the component currently in Tick.
func (m *Motherboard) event(kind, detail string) {
	m.sink(Event{Major: m.Major, Phase: string(m.curPhase), Component: m.ticking, Kind: kind, Detail: detail})
}

// Halt stops the machine at the end of the current phase. The first fault wins.
func (m *Motherboard) Halt(f Fault) {
	if m.Fault == nil {
		fcopy := f
		m.Fault = &fcopy
	}
	m.Halted = true
}

// HaltedBy reports the terminal fault, or a clean halt if none.
func (m *Motherboard) HaltedBy() Fault {
	if m.Fault != nil {
		return *m.Fault
	}
	return Fault{Kind: "halt"}
}

// pendingWritesForTest exposes posted-but-unlatched writes (tests only).
func (m *Motherboard) pendingWritesForTest() []pendingWrite {
	return m.bus.pending
}

// StateReporter is optionally implemented by components that can dump state
// (used by Blue-Screen-style snapshots and the CLI).
type StateReporter interface {
	StateDump() string
}

// VerboseStateReporter is implemented by components with a fuller dump for
// the curious learner (CLI -dump-full).
type VerboseStateReporter interface {
	StateDumpVerbose() string
}

// StateString dumps every component that reports state, in tick order.
// verbose switches in the fuller dumps where a component offers one.
func (m *Motherboard) StateString(verbose bool) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s | major tick %d | %s\n", MachineVersion, m.Major, m.describeHalt())
	for _, c := range m.components {
		var s string
		if verbose {
			if v, ok := c.(VerboseStateReporter); ok {
				s = v.StateDumpVerbose()
			}
		}
		if s == "" {
			if r, ok := c.(StateReporter); ok {
				s = r.StateDump()
			}
		}
		if s != "" {
			fmt.Fprintf(&b, "--- %s ---\n%s", c.Name(), s)
		}
	}
	return b.String()
}

// DescribeWindows lists every address window on the board — the map a player
// needs when a faulting address leads nowhere.
func (m *Motherboard) DescribeWindows() string {
	s := ""
	for i, c := range m.components {
		if r := m.readers[i]; r != nil {
			base, size := r.Window()
			s += fmt.Sprintf("  %-6s 0x%04x..0x%04x (%d bytes)\n", c.Name(), base, base+size, size)
		}
	}
	return s
}

func (m *Motherboard) describeHalt() string {
	f := m.HaltedBy()
	switch f.Kind {
	case "halt":
		return "clean halt"
	default:
		return fmt.Sprintf("stopped: %s (pc=0x%04x addr=0x%04x phase=%s)",
			f.Kind, f.PC, f.Addr, f.Phase)
	}
}
