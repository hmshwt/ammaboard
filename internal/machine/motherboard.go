package machine

import (
	"bytes"
	"fmt"
)

// pendingWrite is a value posted to the bus; it becomes visible to readers
// only at the phase latch (between phases). Everyone within a phase sees a
// stable bus. Site carries the posting instruction when it could be named.
type pendingWrite struct {
	addr uint32
	val  uint32
	size uint32 // 4 = word, 1 = byte
	site WriteSite
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
	fault    *Fault // terminal fault record; nil on a clean halt or budget cut-off
	sink     func(Event)
	ticking  string // component currently in Tick
	curPhase Phase
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

// Run steps up to maxMajors major ticks (maxMajors <= 0 means no cap), or
// until the machine stops. It returns the final major number reached. A run
// that hits the cap while still executing is left mid-flight — a pause, not a
// stop — and StopReason reports StopBudget; calling Run again resumes it.
func (m *Motherboard) Run(maxMajors int) int {
	for i := 0; (maxMajors <= 0 || i < maxMajors) && !m.Halted; i++ {
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
	// A write with nowhere to land stops the machine here (deadWrite).
	for _, w := range m.bus.pending {
		if m.Halted {
			break
		}
		m.commit(w)
	}
	m.bus.pending = m.bus.pending[:0]
}

// commit routes one latched write to the component whose window contains it.
func (m *Motherboard) commit(w pendingWrite) {
	i, ok := m.route(w.addr)
	if !ok {
		m.deadWrite(w, BusNotConnected)
		return
	}
	name := m.components[i].Name()
	if w.size == 1 {
		bw, ok := m.components[i].(ByteWriter)
		if !ok {
			m.deadWrite(w, BusNotPermitted)
			return
		}
		detail := fmt.Sprintf("commit %s[0x%04x] <= byte 0x%02x", name, w.addr, uint8(w.val))
		if br, ok := m.components[i].(ByteReader); ok {
			if old, st := br.BusReadByte(w.addr); st == BusOK {
				detail += fmt.Sprintf(" (was 0x%02x)", old)
			}
		}
		bw.BusWriteByte(w.addr, uint8(w.val))
		m.sink(Event{Major: m.Major, Phase: string(m.curPhase), Component: "board",
			Kind: "latch", Detail: detail})
		return
	}
	wr, ok := m.components[i].(Writer)
	if !ok {
		m.deadWrite(w, BusNotPermitted)
		return
	}
	detail := fmt.Sprintf("commit %s[0x%04x] <= 0x%08x", name, w.addr, w.val)
	if old, st := m.readers[i].Read(w.addr); st == BusOK {
		detail += fmt.Sprintf(" (was 0x%08x)", old)
	}
	wr.Write(w.addr, w.val)
	m.sink(Event{Major: m.Major, Phase: string(m.curPhase), Component: "board",
		Kind: "latch", Detail: detail})
}

// deadWrite is a posted write with nowhere to land — no component owns the
// address, or the owner refuses it. Same class of mistake as a load from
// nowhere, and the machine refuses it just as loudly: the latch stops the
// machine with a fault naming the instruction that posted the write when it
// could (WriteSite). A store must never vanish silently.
func (m *Motherboard) deadWrite(w pendingWrite, st BusStatus) {
	sized := "word"
	if w.size == 1 {
		sized = "byte"
	}
	m.sink(Event{Major: m.Major, Phase: string(m.curPhase), Component: "board",
		Kind: "fault",
		Detail: fmt.Sprintf("%s write to 0x%04x: %s (dropped at latch)",
			sized, w.addr, st)})
	m.Halt(Fault{Kind: "bus_" + string(st), Addr: w.addr, PC: w.site.PC,
		Phase: m.curPhase, Word: w.site.Word})
}

// deliverInterrupts hands at most one raised line to the CPU per cycle. Lines
// are level-triggered: a line refused (IE clear) stays raised and retries at
// the next edge, and a line the device has already cleared (acknowledged
// before delivery) is discarded — the board re-samples the device's current
// line level rather than trusting the raise-time queue.
func (m *Motherboard) deliverInterrupts() {
	if len(m.irqs) == 0 || m.irqTarget == nil {
		return
	}
	for i := 0; i < len(m.irqs); i++ {
		line := m.irqs[i]
		if !m.lineRaised(line) {
			m.irqs = append(m.irqs[:i], m.irqs[i+1:]...)
			m.sink(Event{Major: m.Major, Phase: string(PhaseFetch), Component: "board",
				Kind:   "irq_cleared",
				Detail: fmt.Sprintf("%s: cleared before delivery (acknowledged)", line)})
			i--
			continue
		}
		taken, fromPC, fromStatus, toStatus := m.irqTarget.TakeInterrupt(TrapVector)
		if !taken {
			return
		}
		m.irqs = append(m.irqs[:i], m.irqs[i+1:]...)
		m.sink(Event{Major: m.Major, Phase: string(PhaseFetch), Component: "board",
			Kind: "interrupt",
			Detail: fmt.Sprintf("%s: pc 0x%04x -> 0x%04x, status 0x%02x -> 0x%02x",
				line, fromPC, TrapVector, fromStatus, toStatus)})
		return
	}
}

// lineRaised re-samples one raised line's owner: a device that has since
// cleared its line (ack, disarm) reports false, and the queued line is
// discarded. Lines whose owner cannot report a level are treated as raised.
func (m *Motherboard) lineRaised(line string) bool {
	for _, c := range m.components {
		if c.Name() != line {
			continue
		}
		if l, ok := c.(InterruptLine); ok {
			return l.InterruptRaised()
		}
		return true
	}
	return false // owner gone: nothing drives the wire
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
func (m *Motherboard) WriteBus(addr uint32, val uint32, site WriteSite) {
	m.event("bus_write_post", fmt.Sprintf("addr=0x%04x val=0x%08x", addr, val))
	m.bus.pending = append(m.bus.pending, pendingWrite{addr: addr, val: val, size: 4, site: site})
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
func (m *Motherboard) WriteBusByte(addr uint32, val uint8, site WriteSite) {
	m.event("bus_write_byte_post", fmt.Sprintf("addr=0x%04x val=0x%02x", addr, val))
	m.bus.pending = append(m.bus.pending, pendingWrite{addr: addr, val: uint32(val), size: 1, site: site})
}

// RaiseInterrupt implements Env: a component raises its line. Level-triggered —
// the line stays up until the device itself clears it (usually on MMIO ack),
// and the board re-samples the device at every delivery edge.
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

// Halt implements Env: stop the machine with a fault record. The first fault
// wins.
func (m *Motherboard) Halt(f Fault) {
	if m.fault == nil {
		fcopy := f
		m.fault = &fcopy
	}
	m.Halted = true
}

// CleanHalt implements Env: the program ran halt. A clean stop is not a fault
// — no record, no Blue Screen — the board just marks the machine stopped.
func (m *Motherboard) CleanHalt() {
	m.Halted = true
}

// Fault returns the terminal fault record, or nil if the machine halted
// cleanly, ran out of budget mid-flight, or is still running.
func (m *Motherboard) Fault() *Fault { return m.fault }

// StopReason reports how (or whether) the machine stopped:
//
//	StopHalt    the program ran halt
//	StopFault   a component faulted (see Fault)
//	StopBudget  the run's cycle cap was reached mid-flight: a pause,
//	            resumable by calling Run again
//	StopNone    nothing has run
//
// Never conflate StopBudget with StopHalt: "no fault" is not "halted".
func (m *Motherboard) StopReason() StopReason {
	switch {
	case m.fault != nil:
		return StopFault
	case m.Halted:
		return StopHalt
	case m.Major > 0:
		return StopBudget
	default:
		return StopNone
	}
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
	switch m.StopReason() {
	case StopHalt:
		return "clean halt"
	case StopFault:
		f := m.fault
		return fmt.Sprintf("stopped: %s (pc=0x%04x addr=0x%04x phase=%s)",
			f.Kind, f.PC, f.Addr, f.Phase)
	case StopBudget:
		return "paused: budget exhausted (still running)"
	default:
		return "not started"
	}
}
