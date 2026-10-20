package tui

// eventlog.go — the in-memory tail of the machine's event stream. The batch
// CLI writes every event to SQLite; the TUI only ever needs the recent window.

import "ammaboard/internal/machine"

// EventLog keeps the last cap events (and counts the total).
type EventLog struct {
	cap    int
	events []machine.Event
	total  int
}

// NewEventLog keeps the last cap events (and counts the total).
func NewEventLog(cap int) *EventLog { return &EventLog{cap: cap} }

// Add records one event, trimming the tail to capacity.
func (l *EventLog) Add(e machine.Event) {
	l.events = append(l.events, e)
	if len(l.events) > l.cap {
		l.events = l.events[len(l.events)-l.cap:]
	}
	l.total++
}

// Reset empties the log (on machine reset) without changing capacity.
func (l *EventLog) Reset() { l.events = nil; l.total = 0 }

// Tail returns up to n most recent events, oldest first.
func (l *EventLog) Tail(n int) []machine.Event {
	if n > len(l.events) {
		n = len(l.events)
	}
	out := make([]machine.Event, n)
	copy(out, l.events[len(l.events)-n:])
	return out
}

// Total reports how many events the machine has emitted since (re)start.
func (l *EventLog) Total() int { return l.total }
