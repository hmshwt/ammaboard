package main

// wiring.go — the one place where hardware is chosen and wired onto the board.
// Everything downstream (machine, ticklog) is ignorant of this file.
//
// Pluggability contract (poc.md criterion 5): adding a component = one type
// + one Attach call here. Nothing else in the codebase changes.

import (
	"ammaboard/internal/machine"
	"ammaboard/internal/ticklog"
)

// wiring holds every component instance on this board.
type wiring struct {
	board *machine.Motherboard
	ram   *machine.RAM
	cpu   *machine.CPU
	timer *machine.Timer
	echo  *machine.EchoDevice // optional
}

// buildBoard wires AmmaBoard-2: RAM first (its window owns 0..16KB), then the
// CPU, then the timer, then any optional devices. Order = tick order; windows
// route the bus.
func buildBoard(log *ticklog.Log, echo bool) *wiring {
	w := &wiring{
		ram:   machine.NewRAM(ramBase, ramSize),
		cpu:   machine.NewCPU(entryPoint),
		timer: machine.NewTimer(timerBase),
	}
	w.board = machine.New(func(e machine.Event) { log.Add(ticklog.Event(e)) })
	w.board.Attach(w.ram)
	w.board.Attach(w.cpu)
	w.board.Attach(w.timer)
	if echo {
		w.echo = machine.NewEchoDevice(echoBase)
		w.board.Attach(w.echo)
	}
	return w
}
