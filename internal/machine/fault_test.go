package machine_test

import (
	"strings"
	"testing"

	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
	"ammaboard/internal/machine/cpu"
	"ammaboard/internal/machine/ram"
	"ammaboard/internal/machine/timer"
)

// A store that lands on no component must stop the machine loudly, naming the
// instruction: loads already fault on a dead address — stores must not vanish
// silently (they are refused at the latch, when routing is decided).
func TestStoreToUnownedAddressFaultsAtLatch(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, 1234
        add_number r2, r0, 20480    ; 0x5000: past RAM, no device owns it
        store_word r1, 0(r2)
        halt
`)
	board, _, events := runProgram(t, words)
	f := faultOf(t, board.Fault)
	if f.Kind != "bus_NOT_CONNECTED" || f.Addr != 20480 {
		t.Fatalf("want bus_NOT_CONNECTED at 0x5000, got %+v", f)
	}
	if in, err := isa.Decode(f.Word); err != nil || in.Op != isa.OpStoreWord {
		t.Errorf("fault word must decode to the posting store: %+v (%v)", in, err)
	}
	sawDrop := false
	for _, e := range events {
		if e.Kind == "fault" && strings.Contains(e.Detail, "0x5000") && strings.Contains(e.Detail, "dropped at latch") {
			sawDrop = true
		}
	}
	if !sawDrop {
		t.Error("the latch must log the dropped write as a fault event")
	}
}

// A byte store into a word-only device window is refused the same way: the
// owner answers NOT_PERMITTED, and the latch stops on it.
func TestByteStoreIntoWordOnlyDeviceFaults(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, 5
        add_number r2, r0, 16416    ; timer base 0x4020: word-only MMIO
        store_byte r1, 0(r2)
        halt
`)
	board, _, _ := runProgram(t, words, timer.New(0x4020))
	f := faultOf(t, board.Fault)
	if f.Kind != "bus_NOT_PERMITTED" || f.Addr != 16416 {
		t.Fatalf("want bus_NOT_PERMITTED at 0x4020, got %+v", f)
	}
}

// The -max cycle budget is a pause, not a stop: an infinite program must not
// be reported as a clean halt and must not fault — the board's reason is
// StopBudget, and both board and CPU stay resumable mid-flight.
func TestBudgetExhaustedIsNotCleanHalt(t *testing.T) {
	words := assemble(t, `
loop:   goto r1                  ; r1 = 0, so pc = 0: an infinite loop
`)
	board := machine.New(func(machine.Event) {})
	mem := ram.New(0, 16*1024)
	proc := cpu.New(0)
	board.Attach(mem)
	board.Attach(proc)
	if err := mem.Load(0, words); err != nil {
		t.Fatal(err)
	}
	board.Run(10)
	if board.Halted {
		t.Fatal("budget exhaustion must not mark the machine halted")
	}
	if got := board.StopReason(); got != machine.StopBudget {
		t.Errorf("board stop reason = %q, want %q", got, machine.StopBudget)
	}
	if board.Fault() != nil {
		t.Error("a budget cut-off is not a fault: Fault() must be nil")
	}
	if proc.Halted || proc.Fault() != nil {
		t.Errorf("cpu must stay mid-flight after the budget: halted=%v fault=%v", proc.Halted, proc.Fault())
	}
	// The state banner must not claim a stop either.
	if s := board.StateString(false); !strings.Contains(s, "budget exhausted") {
		t.Errorf("state banner should report budget exhaustion:\n%s", s)
	}
	// And the run is resumable: the same machine can run to a later budget.
	if n := board.Run(5); n != 15 {
		t.Errorf("resumed run ended at major %d, want 15", n)
	}
	if got := board.StopReason(); got != machine.StopBudget {
		t.Errorf("after resuming, stop reason = %q, want %q", got, machine.StopBudget)
	}
}
