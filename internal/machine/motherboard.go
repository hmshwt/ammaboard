// Package machine implements the AmmaBoard motherboard: an Elm-like, single-threaded,
// phase-stepped machine. The motherboard is the broker — components never reference
// each other, all interaction goes through ReadBus/WriteBus, and every interaction
// is logged. Same inputs + same cycle count = identical machine state, always.
package machine

import (
	"bytes"
	"fmt"
)

// Phase is a named minor tick within one major tick (one machine cycle).
type Phase string

// The AmmaBoard-1 phase schedule. CPU phases; bus latches flip between phases.
const (
	PhaseFetch   Phase = "FETCH"
	PhaseDecode  Phase = "DECODE"
	PhaseExecute Phase = "EXECUTE"
)

// Schedule is the fixed phase order per major tick. Owned by the motherboard.
var Schedule = []Phase{PhaseFetch, PhaseDecode, PhaseExecute}

// BusStatus is the uniform result of any bus interaction.
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

// pendingWrite is a value posted to the bus; it becomes visible to readers only
// at the phase latch (between phases). Everyone within a phase sees a stable bus.
type pendingWrite struct {
	addr uint32
	val  uint32
	by   string
}

// Motherboard is the root component: phase schedule, broker, log.
type Motherboard struct {
	components []Component
	readers    []Reader // parallel to components
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
}

// Run steps up to maxMajors major ticks, or until halt/fault. It returns the
// final major number reached.
func (m *Motherboard) Run(maxMajors int) int {
	for i := 0; i < maxMajors && !m.Halted; i++ {
		m.Major++
		m.sink(Event{Major: m.Major, Component: "board", Kind: "major"})
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
	// Phase latch: posted writes become visible to everyone.
	for _, w := range m.bus.pending {
		for i, r := range m.readers {
			if r == nil {
				continue
			}
			base, size := r.Window()
			if w.addr >= base && w.addr < base+size {
				if wr, ok := m.components[i].(Writer); ok {
					wr.Write(w.addr, w.val)
				}
				break
			}
		}
	}
	if len(m.bus.pending) > 0 {
		m.sink(Event{Major: m.Major, Phase: string(phase), Component: "board",
			Kind: "latch", Detail: fmt.Sprintf("%d write(s) committed", len(m.bus.pending))})
	}
	m.bus.pending = m.bus.pending[:0]
}

// ReadBus implements Env. Routes to the component whose window contains addr.
func (m *Motherboard) ReadBus(addr uint32) (uint32, BusStatus) {
	for _, r := range m.readers {
		if r == nil {
			continue
		}
		base, size := r.Window()
		if addr >= base && addr < base+size {
			v, st := r.Read(addr)
			m.event("bus_read", fmt.Sprintf("addr=0x%04x val=0x%08x status=%s", addr, v, st))
			return v, st
		}
	}
	m.event("bus_read", fmt.Sprintf("addr=0x%04x status=%s", addr, BusNotConnected))
	return 0, BusNotConnected
}

// WriteBus implements Env. Posts a write; visible after the phase latch.
func (m *Motherboard) WriteBus(addr uint32, val uint32) {
	m.event("bus_write_post", fmt.Sprintf("addr=0x%04x val=0x%08x", addr, val))
	m.bus.pending = append(m.bus.pending, pendingWrite{addr: addr, val: val})
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

// StateString dumps every component that reports state, in tick order.
func (m *Motherboard) StateString() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "AmmaBoard-1 | major tick %d | %s\n", m.Major, m.describeHalt())
	for _, c := range m.components {
		if s, ok := c.(StateReporter); ok {
			fmt.Fprintf(&b, "--- %s ---\n%s", c.Name(), s.StateDump())
		}
	}
	return b.String()
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
