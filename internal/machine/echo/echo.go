// Package echo implements the AmmaBoard pluggability proof: a minimal device
// that owns a small address window (a one-word mailbox). It is wired onto the
// board with one Attach call and zero edits anywhere else (docs/implementation.md,
// "Adding a device").
package echo

import (
	"fmt"

	"ammaboard/internal/machine"
)

// Device is a one-word mailbox: reads return the last value written, and
// every interaction is logged by the broker. It is passive — no Tick work.
type Device struct {
	Base uint32
	val  uint32
	set  bool
}

// New creates an empty echo mailbox at base (4 bytes of MMIO).
func New(base uint32) *Device { return &Device{Base: base} }

func (d *Device) Name() string { return "echo" }

func (d *Device) Tick(machine.Phase, machine.Env) {}

func (d *Device) Window() (base, size uint32) { return d.Base, 4 }

func (d *Device) Read(addr uint32) (uint32, machine.BusStatus) {
	if !d.set {
		return 0, machine.BusIdle
	}
	return d.val, machine.BusOK
}

func (d *Device) Write(addr, val uint32) {
	d.val = val
	d.set = true
}

// StateDump renders the mailbox (freeze snapshot part).
func (d *Device) StateDump() string {
	if !d.set {
		return fmt.Sprintf("echo@0x%04x: empty\n", d.Base)
	}
	return fmt.Sprintf("echo@0x%04x: %d (0x%08x)\n", d.Base, d.val, d.val)
}
