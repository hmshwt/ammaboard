// Package machine implements the AmmaBoard motherboard: an Elm-like,
// single-threaded, phase-stepped machine. The motherboard is the broker —
// components never reference each other, all interaction goes through
// ReadBus/WriteBus, and every interaction is logged. Same inputs + same
// cycle count = identical machine state, always.
//
// This package holds the contracts only. Components live in their own
// packages — internal/machine/cpu, .../ram, .../timer, .../echo — one package
// per component, so the package boundary enforces the bus-only rule
// (motherboard.md §8).
package machine

// MachineVersion names the machine generation: ISA stage, not app release
// (motherboard.md §2). AmmaBoard-2 = boot ten + bytes/shifts/interrupts.
const MachineVersion = "AmmaBoard-2"

// TrapVector is where the board aims the pc when a pending interrupt line is
// delivered. Fixed for AmmaBoard-2; vectored controllers arrive with a second
// interrupt source.
const TrapVector uint32 = 0x0008

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

// Fault is the machine's honest fault record: what went wrong, where, and
// which instruction did it. It is a fault taxonomy only — a clean halt and a
// budget cut-off are not here, because they are not things that went wrong.
// See StopReason.
type Fault struct {
	Kind  string // "unknown_opcode", "alignment", "stray_rfi", "bus_NOT_CONNECTED", …
	Addr  uint32
	PC    uint32
	Phase Phase
	Word  uint32
}

// StopReason says how (or whether) the machine stopped. It is deliberately not
// a Fault: neither a clean halt nor a budget cut-off has an address or an
// instruction to blame, and neither should ever look like one.
type StopReason string

const (
	StopNone   StopReason = ""                 // nothing has stopped it (Run not called)
	StopHalt   StopReason = "halt"             // the program ran halt
	StopFault  StopReason = "fault"            // a component faulted; see Motherboard.Fault
	StopBudget StopReason = "budget_exhausted" // the run's cycle cap was reached mid-flight
)

// WriteSite names the instruction that posted a bus write, so a latch-time
// failure can point at the offender. The CPU fills both fields; components
// that cannot attribute a write pass the zero value.
type WriteSite struct {
	PC   uint32
	Word uint32
}

// Env is what a component gets each tick: the broker + the log + the power to
// stop the machine. Components never hold references to each other — only to
// this interface.
type Env interface {
	ReadBus(addr uint32) (uint32, BusStatus)
	WriteBus(addr uint32, val uint32, site WriteSite)
	ReadBusByte(addr uint32) (uint8, BusStatus)
	WriteBusByte(addr uint32, val uint8, site WriteSite)
	RaiseInterrupt(line string)
	Logf(kind, format string, args ...any)
	// Halt stops the machine with a fault record.
	Halt(f Fault)
	// CleanHalt stops the machine because the program ran halt: a stop, not a
	// fault, with no record.
	CleanHalt()
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

// InterruptLine is implemented by components that raise interrupt lines and
// can report the line's current level. The board re-samples it at every
// delivery edge — lines are level-triggered, so a line the device has already
// cleared (acknowledged) is discarded, never delivered stale.
type InterruptLine interface {
	InterruptRaised() bool
}

// interruptible is implemented by components that can take an interrupt at
// the board's latch edge (the CPU). The broker knows no details beyond this.
type interruptible interface {
	TakeInterrupt(vector uint32) (taken bool, fromPC, fromStatus, toStatus uint32)
}

// Event is one logged interaction.
type Event struct {
	Major     int
	Phase     string
	Component string
	Kind      string
	Detail    string
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
