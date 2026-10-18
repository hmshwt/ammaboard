package bluescreen_test

import (
	"strings"
	"testing"

	"ammaboard/internal/asm"
	"ammaboard/internal/machine"
	"ammaboard/internal/machine/bluescreen"
	"ammaboard/internal/machine/cpu"
	"ammaboard/internal/machine/ram"
	"ammaboard/internal/machine/timer"
)

func renderFaultScreen(t *testing.T, words []uint32) (string, *cpu.CPU, *machine.Motherboard) {
	t.Helper()
	var events []machine.Event
	board := machine.New(func(e machine.Event) { events = append(events, e) })
	mem := ram.New(0, 16*1024)
	proc := cpu.New(0)
	board.Attach(mem)
	board.Attach(proc)
	board.Attach(timer.New(0x4020)) // the standard wiring always carries the timer
	if err := mem.Load(0, words); err != nil {
		t.Fatal(err)
	}
	board.Run(1000)
	if !board.Halted {
		t.Fatal("fault program should halt the machine")
	}
	return bluescreen.Render(proc, board, mem, events), proc, board
}

func assemble(t *testing.T, src string) []uint32 {
	t.Helper()
	words, err := asm.Assemble(src)
	if err != nil {
		t.Fatal(err)
	}
	return words
}

// faultOf returns a terminal fault record, failing if nothing faulted.
func faultOf(t *testing.T, get func() *machine.Fault) machine.Fault {
	t.Helper()
	f := get()
	if f == nil {
		t.Fatal("expected a fault record, but nothing faulted")
	}
	return *f
}

func TestBlueScreenOnUnalignedStore(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, 1234
        add_number r2, r0, 66
        store_word r1, 0(r2)
        halt
`)
	bs, proc, _ := renderFaultScreen(t, words)
	if faultOf(t, proc.Fault).Kind != "alignment" {
		t.Fatalf("want alignment fault, got %+v", proc.Fault())
	}
	for _, want := range []string{
		"BLUE SCREEN",
		"alignment fault, cycle 3, phase EXECUTE",
		"not a multiple of 4",
		"store_word rd=r0 rs=r2 rt=r1 imm=0",
		"faulting address .................... 0x0042",
		"register_1 = 1234",
		"<- faulting address in this word",
		"ram    0x0000..0x4000",
		">>",
		"the full trail is in trace.txt",
	} {
		if !strings.Contains(bs, want) {
			t.Errorf("blue screen missing %q\n%s", want, bs)
		}
	}
}

// A faulting word that decodes gets disassembled; one that does not gets its
// opcode field named — the player sees the nonsense itself.
func TestBlueScreenOnUnknownOpcode(t *testing.T) {
	bs, proc, _ := renderFaultScreen(t, []uint32{0xFFFFFFFF})
	if faultOf(t, proc.Fault).Kind != "unknown_opcode" {
		t.Fatalf("want unknown_opcode fault, got %+v", proc.Fault())
	}
	for _, want := range []string{
		"unknown_opcode fault, cycle 1, phase DECODE",
		"wandered into data",
		"not a known instruction",
		"opcode field (bits 31..25) .......... 127 of 1..20",
	} {
		if !strings.Contains(bs, want) {
			t.Errorf("blue screen missing %q\n%s", want, bs)
		}
	}
}

// A jump past the end of RAM: bus fault, no memory window to show, but the
// window map tells the player where the wire does lead.
func TestBlueScreenOnBusFault(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, 4096
        add_number r2, r0, 20480
        goto       r2
`)
	bs, proc, _ := renderFaultScreen(t, words)
	if faultOf(t, proc.Fault).Kind != "bus_NOT_CONNECTED" {
		t.Fatalf("want bus_NOT_CONNECTED fault, got %+v", proc.Fault())
	}
	if !strings.Contains(bs, "no component owns") {
		t.Errorf("blue screen missing bus explanation:\n%s", bs)
	}
	if strings.Contains(bs, "memory around the faulting address") {
		t.Error("faulting address outside RAM must not show a RAM window")
	}
	if !strings.Contains(bs, "timer  0x4020..0x4028") {
		t.Errorf("blue screen missing window map:\n%s", bs)
	}
}

// A dropped store (latch-time fault) renders the full panel, disassembling
// the store that posted it — the receipts survive even for latch faults.
func TestBlueScreenOnDroppedStore(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, 1234
        add_number r2, r0, 20480
        store_word r1, 0(r2)
        halt
`)
	bs, _, board := renderFaultScreen(t, words)
	if f := faultOf(t, board.Fault); f.Kind != "bus_NOT_CONNECTED" {
		t.Fatalf("want latch-time bus_NOT_CONNECTED on the board, got %+v", f)
	}
	for _, want := range []string{
		"bus_NOT_CONNECTED fault",
		"store_word rd=r0 rs=r2 rt=r1 imm=0",
		"dropped at latch",
	} {
		if !strings.Contains(bs, want) {
			t.Errorf("blue screen missing %q\n%s", want, bs)
		}
	}
}
