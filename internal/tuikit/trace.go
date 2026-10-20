package tuikit

// trace.go — the receipt ring: every dispatch records what arrived, what was
// consulted, and what answered. It is the dynamic twin of the wiring audit:
// Verify says what would happen with this wiring, the Trace says what
// actually happened on this keypress. Identical consecutive receipts
// compress into one entry, so a 30fps heartbeat cannot drown the log.

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

// TraceKind grades a receipt.
type TraceKind int

const (
	TraceKey  TraceKind = iota // a key dispatch through the ladder
	TraceMsg                   // a non-key message dispatch (facts, ticks, geometry)
	TraceEmit                  // a fact queued by a component (see emit.go)
	TraceCmd                   // a command produced a message
)

func (k TraceKind) String() string {
	switch k {
	case TraceKey:
		return "key"
	case TraceMsg:
		return "msg"
	case TraceEmit:
		return "emit"
	case TraceCmd:
		return "cmd"
	}
	return "?"
}

// TraceEvent is one receipt.
type TraceEvent struct {
	Seq   int64
	At    time.Time
	Kind  TraceKind
	Key   string // the key, for TraceKey
	From  string // provenance: the winning claim, "Handle", "emit:<label>", "terminal"
	What  string // the resolution: the ladder walk, the verdict
	Count int    // identical consecutive receipts compressed into this entry
}

func (e TraceEvent) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "#%d %s %-4s", e.Seq, e.At.Format("15:04:05.000"), e.Kind)
	if e.Key != "" {
		sb.WriteString(" " + strconv.Quote(e.Key))
	}
	if e.From != "" {
		fmt.Fprintf(&sb, " · %s", e.From)
	}
	if e.What != "" {
		fmt.Fprintf(&sb, " · %s", e.What)
	}
	if e.Count > 1 {
		fmt.Fprintf(&sb, " (×%d)", e.Count)
	}
	return sb.String()
}

// Trace is a bounded ring of receipts. Recording is cheap and optional:
// NewApp enables it, assigning nil silences it.
type Trace struct {
	mu     sync.Mutex
	events []TraceEvent
	cap    int
	seq    int64
}

func newTrace(capacity int) *Trace {
	if capacity < 1 {
		capacity = 1
	}
	return &Trace{cap: capacity}
}

// add appends one receipt, compressing identical consecutive repeats.
func (t *Trace) add(ev TraceEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.seq++
	ev.Seq, ev.At, ev.Count = t.seq, time.Now(), 1
	if n := len(t.events); n > 0 {
		last := &t.events[n-1]
		if last.Kind == ev.Kind && last.Key == ev.Key && last.From == ev.From && last.What == ev.What {
			last.Count++
			last.At = ev.At
			return
		}
	}
	if len(t.events) >= t.cap { // bounded: the oldest receipt falls off
		copy(t.events, t.events[1:])
		t.events = t.events[:t.cap-1]
	}
	t.events = append(t.events, ev)
}

// Snapshot copies the receipts, oldest first.
func (t *Trace) Snapshot() []TraceEvent {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]TraceEvent, len(t.events))
	copy(out, t.events)
	return out
}

// Log renders the receipts, one per line — the debugging surface.
func (t *Trace) Log() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var sb strings.Builder
	for _, ev := range t.events {
		sb.WriteString(ev.String())
		sb.WriteByte('\n')
	}
	return sb.String()
}

// record, recordKey, recordMsg — the dispatch-side receipt writers. All are
// no-ops when the ring is silenced (App.Trace == nil).
func (a *App) record(ev TraceEvent) {
	if a.Trace == nil {
		return
	}
	a.Trace.add(ev)
}

func (a *App) recordKey(key, from, what string) {
	a.record(TraceEvent{Kind: TraceKey, Key: key, From: from, What: what})
}

func (a *App) recordMsg(from, what string) {
	a.record(TraceEvent{Kind: TraceMsg, From: from, What: what})
}

// teaKeyPress builds a KeyPressMsg for a single printable key — the test
// face for driving the ladder headlessly.
func teaKeyPress(key string) tea.Msg {
	return tea.KeyPressMsg{Code: rune(key[0]), Text: key}
}

// AssertRoute routes key through a headlessly and asserts that the receipt's
// resolution carries want — the routing contract as a test. key must be a
// single printable rune; special keys are driven by hand against App.Update
// and asserted through App.Trace.
func AssertRoute(t TestingT, a *App, key, want string) {
	if a.Trace == nil {
		a.Trace = newTrace(64)
	}
	before := int64(0)
	for _, ev := range a.Trace.Snapshot() {
		if ev.Seq > before {
			before = ev.Seq
		}
	}
	a.Update(teaKeyPress(key))
	for _, ev := range a.Trace.Snapshot() {
		if ev.Kind == TraceKey && ev.Key == key && ev.Seq > before {
			if strings.Contains(ev.What, want) {
				return
			}
			t.Errorf("route %q: got %q, want it to contain %q", key, ev.What, want)
			return
		}
	}
	t.Errorf("route %q produced no receipt", key)
}
