package tuikit

// emit.go — the upward channel. Components hold their own state and derive
// their regions; the one thing they may lack is a way to say "something
// happened" to whoever owns policy. Emit is that way: a provenance-stamped
// queue with exactly one consumer — the root stream. Facts are notices,
// never payloads: state moves through stores, messages say that something
// happened.

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// emission is the envelope that carries a queued fact back through dispatch.
type emission struct {
	from string
	msg  tea.Msg
}

// Emit is the upward channel handed to a component at construction. Send is
// safe from any goroutine — including a long-running Cmd's.
type Emit struct {
	app  *App
	from string
}

// Emitter returns the upward channel stamped with a provenance label; every
// fact sent through it carries the label into the receipts.
func (a *App) Emitter(from string) Emit { return Emit{app: a, from: from} }

// Send queues a fact for the root. The kit does not self-wake: the fact is
// dispatched at the next Update boundary, in FIFO order — a heartbeat host
// drains it within one frame. Never re-entrant: a Send during a dispatch
// cannot re-enter that dispatch. Emissions are plain tea.Msg values; there
// are no subscriptions, no targeting, no priorities.
func (e Emit) Send(msg tea.Msg) {
	if e.app == nil || msg == nil {
		return
	}
	e.app.mu.Lock()
	e.app.pending = append(e.app.pending, emission{from: e.from, msg: msg})
	e.app.mu.Unlock()
	e.app.record(TraceEvent{Kind: TraceEmit, From: e.from, What: fmt.Sprintf("emitted %T", msg)})
}
