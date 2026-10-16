package machine

import "fmt"

// Timer is the day-8 periodic interrupt source: an MMIO device that raises an
// interrupt line every N major ticks while armed.
//
// Register map (word accesses only, little-endian):
//
//	base+0  interval — write: arm with interval in major ticks (0 = disable;
//	                   any write restarts the counter); read: current interval
//	base+4  pending  — read: 1 while the line is raised; write: acknowledge
//	                   (clears the line)
//
// The counter advances once per major tick, on FETCH. Determinism by
// construction: same stores, same majors, same interrupt schedule.
type Timer struct {
	Base     uint32
	interval uint32
	counter  uint32
	pending  bool
}

func NewTimer(base uint32) *Timer { return &Timer{Base: base} }

func (t *Timer) Name() string { return "timer" }

func (t *Timer) Window() (base, size uint32) { return t.Base, 8 }

func (t *Timer) Tick(phase Phase, env Env) {
	if phase != PhaseFetch || t.interval == 0 {
		return
	}
	t.counter++
	if t.counter < t.interval {
		return
	}
	t.counter = 0
	if !t.pending {
		t.pending = true
		env.RaiseInterrupt(t.Name())
	}
}

func (t *Timer) Read(addr uint32) (uint32, BusStatus) {
	if addr%4 != 0 {
		return 0, BusNotPermitted
	}
	switch addr - t.Base {
	case 0:
		return t.interval, BusOK
	case 4:
		if t.pending {
			return 1, BusOK
		}
		return 0, BusOK
	default:
		return 0, BusNotConnected
	}
}

func (t *Timer) Write(addr, val uint32) {
	switch addr - t.Base {
	case 0:
		t.interval = val
		t.counter = 0
	case 4:
		t.pending = false // acknowledge
	}
}

// StateDump renders the timer (freeze snapshot part).
func (t *Timer) StateDump() string {
	armed := "disarmed"
	if t.interval > 0 {
		armed = fmt.Sprintf("every %d major ticks", t.interval)
	}
	line := "clear"
	if t.pending {
		line = "RAISED"
	}
	return fmt.Sprintf("timer@0x%04x: %s, line %s\n", t.Base, armed, line)
}
