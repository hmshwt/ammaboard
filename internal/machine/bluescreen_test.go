package machine

import (
	"strings"
	"testing"
)

func renderFaultScreen(t *testing.T, words []uint32) (string, *CPU) {
	t.Helper()
	var events []Event
	board := New(func(e Event) { events = append(events, e) })
	ram := NewRAM(0, 16*1024)
	cpu := NewCPU(0)
	board.Attach(ram)
	board.Attach(cpu)
	board.Attach(NewTimer(0x4020)) // the standard wiring always carries the timer
	if err := ram.Load(0, words); err != nil {
		t.Fatal(err)
	}
	board.Run(1000)
	if !board.Halted {
		t.Fatal("fault program should halt the machine")
	}
	return BlueScreen(cpu, board, ram, events), cpu
}

func TestBlueScreenOnUnalignedStore(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, 1234
        add_number r2, r0, 66
        store_word r1, 0(r2)
        halt
`)
	bs, cpu := renderFaultScreen(t, words)
	if cpu.HaltedBy().Kind != "alignment" {
		t.Fatalf("want alignment fault, got %+v", cpu.HaltedBy())
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
	bs, cpu := renderFaultScreen(t, []uint32{0xFFFFFFFF})
	if cpu.HaltedBy().Kind != "unknown_opcode" {
		t.Fatalf("want unknown_opcode fault, got %+v", cpu.HaltedBy())
	}
	for _, want := range []string{
		"unknown_opcode fault, cycle 1, phase DECODE",
		"wandered into data",
		"not a known instruction",
		"opcode field (bits 31..27) .......... 31 of 1..20",
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
	bs, cpu := renderFaultScreen(t, words)
	if cpu.HaltedBy().Kind != "bus_NOT_CONNECTED" {
		t.Fatalf("want bus_NOT_CONNECTED fault, got %+v", cpu.HaltedBy())
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
