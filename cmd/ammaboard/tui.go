package main

// tui.go — the interactive path: assemble a program (or the built-in counter
// demo), wire a machine with an in-memory event tail, and hand both to the
// bubbletea model in internal/tui. No SQLite here — the TUI is a view over the
// live machine, and logging remains the batch path's job.

import (
	"fmt"
	"os"
	"path/filepath"

	"ammaboard/internal/asm"
	"ammaboard/internal/machine"
	"ammaboard/internal/machine/cpu"
	"ammaboard/internal/machine/ram"
	tuiapp "ammaboard/internal/tui"
	tea "charm.land/bubbletea/v2"
)

// counterDemo mirrors examples/counter.amb — the fallback program when tui
// starts without a file. examples/counter.amb is the source of truth; keep in
// sync by hand.
const counterDemo = `
        add_number r1, r0, 0        ; i = 0            (r0 is always zero)
        add_number r2, r0, 5        ; n = 5
        add_number r3, r0, 64       ; data base = 64

loop:   store_word r1, 0(r3)        ; mem[r3] = i
        add_number r1, r1, 1        ; i++
        subtract   r4, r2, r1       ; r4 = n - i
        jump_if_not_equal r4, r0, loop   ; loop while n != i

        load_word  r5, 0(r3)        ; r5 = mem[64] = 4 (the last store)
        halt
`

// tui runs the terminal UI (docs/ui.md). An empty srcPath loads the built-in
// counter demo.
func tui(srcPath string) error {
	name := "counter.amb (built-in)"
	src := counterDemo
	if srcPath != "" {
		b, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		name, src = filepath.Base(srcPath), string(b)
	}
	words, err := asm.Assemble(src)
	if err != nil {
		return fmt.Errorf("assemble: %w", err)
	}

	events := tuiapp.NewEventLog(512)
	// newMachine is the reset path too: wiring + program load, nothing else.
	// The events log is shared across resets — the sink closure keeps pointing
	// at it, and reset() clears its contents.
	newMachine := func() (*machine.Motherboard, *cpu.CPU, *ram.RAM) {
		w := buildBoard(func(e machine.Event) { events.Add(e) }, false)
		if err := w.ram.Load(entryPoint, words); err != nil {
			panic(err) // assembling fit check is a programming error, not runtime
		}
		return w.board, w.cpu, w.ram
	}

	m := tuiapp.New(name, words, events, newMachine)
	_, err = tea.NewProgram(m).Run()
	return err
}
