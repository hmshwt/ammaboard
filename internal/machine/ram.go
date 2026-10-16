package machine

import "fmt"

// RAM is byte-addressable little-endian memory. It owns its state; the only way
// in is through the bus (ReadBus routes here, WriteBus latches into here).
type RAM struct {
	Base uint32
	Size uint32
	mem  []byte
}

// NewRAM creates zeroed memory covering [Base, Base+Size).
func NewRAM(base, size uint32) *RAM {
	return &RAM{Base: base, Size: size, mem: make([]byte, size)}
}

func (r *RAM) Name() string { return "ram" }

// Tick: RAM is passive on AmmaBoard-2 — it acts only when the bus touches it.
func (r *RAM) Tick(phase Phase, env Env) {}

func (r *RAM) Window() (base, size uint32) { return r.Base, r.Size }

// Read returns one aligned word. Unaligned access is refused (the machine
// refuses nonsense — real ISAs vary here; ours is strict by choice).
func (r *RAM) Read(addr uint32) (uint32, BusStatus) {
	if addr%4 != 0 {
		return 0, BusNotPermitted
	}
	off := addr - r.Base
	if int(off)+4 > len(r.mem) {
		return 0, BusNotConnected
	}
	v := uint32(r.mem[off]) | uint32(r.mem[off+1])<<8 | uint32(r.mem[off+2])<<16 | uint32(r.mem[off+3])<<24
	return v, BusOK
}

// Write stores one word. Unaligned writes are refused loudly by the caller's
// side: the broker marks them BusNotPermitted and the CPU faults on them,
// same as reads — the machine refuses nonsense, symmetrically.
func (r *RAM) Write(addr, val uint32) {
	if addr%4 != 0 || int(addr-r.Base)+4 > len(r.mem) {
		return // never reached in a well-formed latch; broker checks first
	}
	off := addr - r.Base
	r.mem[off] = byte(val)
	r.mem[off+1] = byte(val >> 8)
	r.mem[off+2] = byte(val >> 16)
	r.mem[off+3] = byte(val >> 24)
}

// BusReadByte returns one byte. Byte access has no alignment constraint —
// that is the point of byte-granular ops (stage 2).
func (r *RAM) BusReadByte(addr uint32) (uint8, BusStatus) {
	off := addr - r.Base
	if off >= uint32(len(r.mem)) {
		return 0, BusNotConnected
	}
	return r.mem[off], BusOK
}

// BusWriteByte stores one byte (latched by the broker like word writes).
func (r *RAM) BusWriteByte(addr uint32, val uint8) {
	off := addr - r.Base
	if off >= uint32(len(r.mem)) {
		return // never reached in a well-formed latch; broker routes first
	}
	r.mem[off] = val
}

// Load copies a program into memory at word-aligned byte offset.
func (r *RAM) Load(byteOffset uint32, words []uint32) error {
	if byteOffset%4 != 0 {
		return fmt.Errorf("load offset must be word-aligned")
	}
	if int(byteOffset)+4*len(words) > len(r.mem) {
		return fmt.Errorf("program does not fit in RAM")
	}
	for i, w := range words {
		r.Write(byteOffset+uint32(4*i), w)
	}
	return nil
}

// Dump returns the full memory image (hexdump-friendly; ≤32KB by the sizing rule).
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
		w := uint32(r.mem[off]) | uint32(r.mem[off+1])<<8 | uint32(r.mem[off+2])<<16 | uint32(r.mem[off+3])<<24
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
