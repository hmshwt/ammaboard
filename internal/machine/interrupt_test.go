package machine_test

import (
	"strings"
	"testing"

	"ammaboard/internal/machine"
	"ammaboard/internal/machine/timer"
)

const timerSrc = `
        add_number r1, r0, start    ; 0x0000 jump over the trap vector
        goto       r1               ; 0x0004
handler:                            ; 0x0008 = TrapVector
        add_number r5, r5, 1        ; ticks so far
        store_word r5, 128(r0)       ; publish the count
        add_number r6, r0, 2
        subtract   r6, r5, r6
        jump_if_not_equal r6, r0, ack
        store_word r0, 0(r2)        ; count reached 2: disarm the timer
ack:    store_word r0, 4(r2)        ; acknowledge the line
        return_from_interrupt
start:                              ; 0x0024
        add_number r2, r0, 16416    ; timer base 0x4020
        add_number r3, r0, 3        ; fire every 3 major ticks
        store_word r3, 0(r2)        ; arm
        allow_interrupts
wait:   load_word  r4, 128(r0)
        add_number r6, r0, 2
        subtract   r6, r4, r6
        jump_if_not_equal r6, r0, wait
        forbid_interrupts
        halt
`

func TestTimerInterruptRoundTrip(t *testing.T) {
	words := assemble(t, timerSrc)
	board, proc, events := runProgram(t, words, timer.New(0x4020))
	checkHalt(t, proc)
	if proc.Regs[5] != 2 || proc.Regs[4] != 2 {
		t.Errorf("r5=%d r4=%d, want 2/2 (handler ran twice, main saw both)", proc.Regs[5], proc.Regs[4])
	}
	interrupts, raises := 0, -1
	firstInterruptMajor := -1
	for _, e := range events {
		switch e.Kind {
		case "irq":
			if raises < 0 {
				raises = e.Major
			}
		case "interrupt":
			interrupts++
			if firstInterruptMajor < 0 {
				firstInterruptMajor = e.Major
			}
		}
	}
	if interrupts != 2 {
		t.Errorf("interrupt events = %d, want 2", interrupts)
	}
	if raises < 0 || firstInterruptMajor <= raises {
		t.Errorf("interrupt (major %d) must be delivered after raise (major %d)", firstInterruptMajor, raises)
	}
	if !board.Halted {
		t.Error("board should be halted")
	}
}

func TestForbidBlocksDeliveryUntilAllow(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, start
        goto       r1
handler:                            ; 0x0008 = TrapVector
        add_number r5, r5, 1
        store_word r5, 128(r0)
        store_word r0, 0(r2)        ; one-shot: disarm
        store_word r0, 4(r2)        ; acknowledge
        return_from_interrupt
start:
        add_number r2, r0, 16416    ; timer base
        forbid_interrupts
        add_number r3, r0, 2        ; interval 2 — fires while IE is clear
        store_word r3, 0(r2)        ; arm
        add_number r4, r0, 4
        add_number r6, r0, 0
spin:   add_number r6, r6, 1
        subtract   r7, r4, r6
        jump_if_not_equal r7, r0, spin
        allow_interrupts            ; the raised line may deliver now
wait:   load_word r5, 128(r0)
        jump_if_equal r5, r0, wait
        forbid_interrupts
        halt
`)
	board, proc, events := runProgram(t, words, timer.New(0x4020))
	checkHalt(t, proc)
	if proc.Regs[5] != 1 {
		t.Errorf("r5 = %d, want 1 (line refused while forbidden, delivered once after allow)", proc.Regs[5])
	}
	allowMajor, interruptMajor := -1, -1
	for _, e := range events {
		if e.Kind == "status" && strings.Contains(e.Detail, "ie=1") && allowMajor < 0 {
			allowMajor = e.Major
		}
		if e.Kind == "interrupt" && interruptMajor < 0 {
			interruptMajor = e.Major
		}
	}
	if allowMajor < 0 || interruptMajor < 0 || interruptMajor <= allowMajor {
		t.Errorf("delivery (major %d) must come after allow (major %d)", interruptMajor, allowMajor)
	}
	_ = board
}

// Level-triggered, not queued: a line acknowledged before delivery must be
// discarded — the board re-samples the device's current level at the edge.
func TestAckBeforeDeliveryCancelsStaleInterrupt(t *testing.T) {
	words := assemble(t, `
        add_number r1, r0, start
        goto       r1
handler:                            ; 0x0008 = TrapVector
        add_number r5, r5, 1        ; must never run: the line was acked first
        return_from_interrupt
start:
        add_number r2, r0, 16416    ; timer base 0x4020
        forbid_interrupts
        add_number r3, r0, 2
        store_word r3, 0(r2)        ; arm: fires while IE is clear
        add_number r4, r0, 8
spin:   add_number r4, r4, -1
        jump_if_not_equal r4, r0, spin
        store_word r0, 0(r2)        ; disarm first, so no fresh raise can race us
        store_word r0, 4(r2)        ; ACK before delivery: the line drops
        allow_interrupts
        add_number r4, r0, 8
spin2:  add_number r4, r4, -1
        jump_if_not_equal r4, r0, spin2
        forbid_interrupts
        halt
`)
	board, proc, events := runProgram(t, words, timer.New(0x4020))
	checkBoardHalt(t, board)
	if proc.Regs[5] != 0 {
		t.Errorf("handler ran on a stale line: r5 = %d, want 0", proc.Regs[5])
	}
	delivered, cleared := 0, false
	for _, e := range events {
		switch e.Kind {
		case "interrupt":
			delivered++
		case "irq_cleared":
			cleared = true
		}
	}
	if delivered != 0 {
		t.Errorf("interrupt events = %d, want 0 (stale line must be cancelled)", delivered)
	}
	if !cleared {
		t.Error("the board should log the acked line as cleared before delivery")
	}
}

func TestStrayRFIFaults(t *testing.T) {
	words := assemble(t, `
        return_from_interrupt
        halt
`)
	board, proc, _ := runProgram(t, words)
	if faultOf(t, proc.Fault).Kind != "stray_rfi" {
		t.Fatalf("want stray_rfi fault, got %+v", proc.Fault())
	}
	if !board.Halted {
		t.Fatal("machine should be halted")
	}
}

// Interrupt delivery is part of the machine's determinism surface.
func TestInterruptDeterminism(t *testing.T) {
	run := func() []machine.Event {
		words := assemble(t, timerSrc)
		_, _, events := runProgram(t, words, timer.New(0x4020))
		return events
	}
	a, b := run(), run()
	if len(a) != len(b) {
		t.Fatalf("event count differs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("event %d differs:\n%+v\n%+v", i, a[i], b[i])
		}
	}
}
