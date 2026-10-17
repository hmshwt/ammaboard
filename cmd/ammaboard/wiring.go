package main

// wiring.go — the one place where hardware is chosen and wired onto the board.
// Everything downstream (machine, ticklog) is ignorant of this file.
//
// Pluggability contract (docs/implementation.md): adding a component = one type
// + one Attach call here. Nothing else in the codebase changes.

import (
	"ammaboard/internal/machine"
	"ammaboard/internal/machine/cpu"
	"ammaboard/internal/machine/echo"
	"ammaboard/internal/machine/ram"
	"ammaboard/internal/machine/timer"
)

// wiring holds every component instance on this board.
type wiring struct {
	board *machine.Motherboard
	ram   *ram.RAM
	cpu   *cpu.CPU
	timer *timer.Timer
	echo  *echo.Device // optional
}

// buildBoard wires AmmaBoard-2: RAM first (its window owns 0..16KB), then the
// CPU, then the timer, then any optional devices. Order = tick order; windows
// route the bus. The sink receives every machine event: the CLI feeds it to
// the SQLite log, the TUI keeps an in-memory tail.
func buildBoard(sink func(machine.Event), withEcho bool) *wiring {
	w := &wiring{
		ram:   ram.New(ramBase, ramSize),
		cpu:   cpu.New(entryPoint),
		timer: timer.New(timerBase),
	}
	w.board = machine.New(sink)
	w.board.Attach(w.ram)
	w.board.Attach(w.cpu)
	w.board.Attach(w.timer)
	if withEcho {
		w.echo = echo.New(echoBase)
		w.board.Attach(w.echo)
	}
	return w
}
