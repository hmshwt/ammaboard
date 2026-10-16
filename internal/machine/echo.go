package machine

import "fmt"

// EchoDevice is a minimal pluggable component — the PoC's proof that new
// hardware can be wired onto the board with a type and one Attach call, and
// zero edits anywhere else. It owns a small address window: reads return the
// last value written (a one-word mailbox), and every interaction is logged.
type EchoDevice struct {
	Base uint32
	val  uint32
	set  bool
}

func NewEchoDevice(base uint32) *EchoDevice { return &EchoDevice{Base: base} }

func (e *EchoDevice) Name() string { return "echo" }

func (e *EchoDevice) Tick(phase Phase, env Env) {}

func (e *EchoDevice) Window() (base, size uint32) { return e.Base, 4 }

func (e *EchoDevice) Read(addr uint32) (uint32, BusStatus) {
	if !e.set {
		return 0, BusIdle
	}
	return e.val, BusOK
}

func (e *EchoDevice) Write(addr, val uint32) {
	e.val = val
	e.set = true
}

// StateDump renders the mailbox (freeze snapshot part).
func (e *EchoDevice) StateDump() string {
	if !e.set {
		return fmt.Sprintf("echo@0x%04x: empty\n", e.Base)
	}
	return fmt.Sprintf("echo@0x%04x: %d (0x%08x)\n", e.Base, e.val, e.val)
}
