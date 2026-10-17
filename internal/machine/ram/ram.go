// Package ram implements the AmmaBoard RAM: byte-addressable little-endian
// memory covering one address window. It owns its state; the only way in is
// through the bus — the broker routes reads here and latches writes here.
package ram

import (
	"encoding/binary"
	"fmt"

	"ammaboard/internal/machine"
)

// RAM is byte-addressable little-endian memory covering [Base, Base+Size).
type RAM struct {
	Base uint32
	Size uint32
	mem  []byte
}

// New creates zeroed memory covering [base, base+size).
func New(base, size uint32) *RAM {
	return &RAM{Base: base, Size: size, mem: make([]byte, size)}
}

func (r *RAM) Name() string { return "ram" }

// Tick: RAM is passive — it acts only when the bus touches it.
func (r *RAM) Tick(machine.Phase, machine.Env) {}

func (r *RAM) Window() (base, size uint32) { return r.Base, r.Size }

// Read returns one aligned word. Unaligned access is refused (the machine
// refuses nonsense — real ISAs vary here; ours is strict by choice).
func (r *RAM) Read(addr uint32) (uint32, machine.BusStatus) {
	if addr%4 != 0 {
		return 0, machine.BusNotPermitted
	}
	off := addr - r.Base
	if uint64(off)+4 > uint64(len(r.mem)) {
		return 0, machine.BusNotConnected
	}
	return binary.LittleEndian.Uint32(r.mem[off:]), machine.BusOK
}

// Write stores one word. Unaligned writes are refused loudly by the caller's
// side: the broker faults them before the latch, same as reads — the machine
// refuses nonsense, symmetrically.
func (r *RAM) Write(addr, val uint32) {
	if addr%4 != 0 || uint64(addr-r.Base)+4 > uint64(len(r.mem)) {
		return // never reached in a well-formed latch; broker checks first
	}
	binary.LittleEndian.PutUint32(r.mem[addr-r.Base:], val)
}

// BusReadByte returns one byte. Byte access has no alignment constraint —
// that is the point of byte-granular ops (stage 2).
func (r *RAM) BusReadByte(addr uint32) (uint8, machine.BusStatus) {
	off := addr - r.Base
	if uint64(off) >= uint64(len(r.mem)) {
		return 0, machine.BusNotConnected
	}
	return r.mem[off], machine.BusOK
}

// BusWriteByte stores one byte (latched by the broker like word writes).
func (r *RAM) BusWriteByte(addr uint32, val uint8) {
	off := addr - r.Base
	if uint64(off) >= uint64(len(r.mem)) {
		return // never reached in a well-formed latch; broker routes first
	}
	r.mem[off] = val
}

// Load copies a program into memory at a word-aligned byte offset.
func (r *RAM) Load(byteOffset uint32, words []uint32) error {
	if byteOffset%4 != 0 {
		return fmt.Errorf("load offset must be word-aligned")
	}
	if uint64(byteOffset)+4*uint64(len(words)) > uint64(len(r.mem)) {
		return fmt.Errorf("program does not fit in RAM")
	}
	for i, w := range words {
		r.Write(byteOffset+uint32(4*i), w)
	}
	return nil
}

// Dump returns a copy of the full memory image (hexdump-friendly; ≤32KB by
// the sizing rule).
func (r *RAM) Dump() []byte {
	out := make([]byte, len(r.mem))
	copy(out, r.mem)
	return out
}

// StateDump renders a compact hexdump of nonzero words (freeze snapshot part).
func (r *RAM) StateDump() string {
	return r.renderDump(false)
}

// StateDumpVerbose renders every word — the -dump-full view for the curious
// learner. Same format, no nonzero filter, no truncation.
func (r *RAM) StateDumpVerbose() string {
	return r.renderDump(true)
}

func (r *RAM) renderDump(full bool) string {
	s := fmt.Sprintf("ram: 0x%04x..0x%04x (%d bytes)\n", r.Base, r.Base+r.Size, r.Size)
	nonzero := 0
	for off := 0; off+4 <= len(r.mem); off += 4 {
		w := binary.LittleEndian.Uint32(r.mem[off:])
		if w == 0 && !full {
			continue // compact: only nonzero words
		}
		s += fmt.Sprintf("0x%04x: %08x\n", r.Base+uint32(off), w)
		if w != 0 {
			nonzero++
		}
		if !full && nonzero >= 64 {
			s += "... (truncated)\n"
			break
		}
	}
	return s
}
