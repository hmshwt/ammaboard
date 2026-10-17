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
	"ammaboard/internal/ticklog"
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
// route the bus.
func buildBoard(log *ticklog.Log, withEcho bool) *wiring {
	w := &wiring{
		ram:   ram.New(ramBase, ramSize),
		cpu:   cpu.New(entryPoint),
		timer: timer.New(timerBase),
	}
	w.board = machine.New(func(e machine.Event) { log.Add(ticklog.Event(e)) })
	w.board.Attach(w.ram)
	w.board.Attach(w.cpu)
	w.board.Attach(w.timer)
	if withEcho {
		w.echo = echo.New(echoBase)
		w.board.Attach(w.echo)
	}
	return w
}
