package tuikit

// emit_test.go — the upward channel: facts queue, land at the next dispatch
// boundary in FIFO order, are safe from goroutines, and carry provenance
// into the receipts. Facts are notices, never payloads.

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"charm.land/bubbletea/v2"
)

type factMsg struct{ what string }

func TestEmitLandsAtNextDispatch(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})

	var got []string
	app.Handle = func(msg tea.Msg) (tea.Cmd, bool) {
		if f, ok := msg.(factMsg); ok {
			got = append(got, f.what)
			return nil, true
		}
		return nil, false
	}

	app.Emitter("numbers").Send(factMsg{what: "stale"})
	if len(got) != 0 {
		t.Fatal("the kit must not dispatch inline — facts land at a dispatch boundary")
	}

	app.Update(tickMsg{}) // the boundary: the queue drains first
	if !slices.Equal(got, []string{"stale"}) {
		t.Fatalf("facts = %v", got)
	}

	// The receipt pair: sent with provenance, dispatched to Handle.
	var sent, delivered bool
	for _, ev := range app.Trace.Snapshot() {
		if ev.Kind == TraceEmit && ev.From == "numbers" && strings.Contains(ev.What, "factMsg") {
			sent = true
		}
		if ev.Kind == TraceMsg && ev.From == "emit:numbers" && strings.Contains(ev.What, "consumed by Handle") {
			delivered = true
		}
	}
	if !sent || !delivered {
		t.Errorf("receipt pair missing: sent=%v delivered=%v", sent, delivered)
	}
}

func TestEmitIsFIFO(t *testing.T) {
	app, _, _ := newTestApp(t)
	var got []string
	app.Handle = func(msg tea.Msg) (tea.Cmd, bool) {
		if f, ok := msg.(factMsg); ok {
			got = append(got, f.what)
			return nil, true
		}
		return nil, false
	}
	emit := app.Emitter("find")
	emit.Send(factMsg{what: "a"})
	emit.Send(factMsg{what: "b"})
	app.Update(tickMsg{})
	if !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("FIFO order violated: %v", got)
	}
}

func TestEmitSafeFromGoroutines(t *testing.T) {
	app, _, _ := newTestApp(t)
	var mu sync.Mutex
	n := 0
	app.Handle = func(msg tea.Msg) (tea.Cmd, bool) {
		if _, ok := msg.(factMsg); ok {
			mu.Lock()
			n++
			mu.Unlock()
			return nil, true
		}
		return nil, false
	}
	emit := app.Emitter("async")
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			emit.Send(factMsg{what: "x"})
		}()
	}
	wg.Wait()
	app.Update(tickMsg{})
	mu.Lock()
	defer mu.Unlock()
	if n != 50 {
		t.Fatalf("facts delivered = %d, want 50", n)
	}
}

func TestEmitFromAKeyClaim(t *testing.T) {
	// The realistic shape: a binding's Run sends a fact; it lands at the next
	// dispatch boundary — one frame later, receipted both ways.
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	emit := app.Emitter("help")
	app.Keys = Keymap{{Keys: []string{"?"}, Help: "toggle help",
		Run: func() tea.Cmd { emit.Send(factMsg{what: "toggle"}); return nil }}}

	var got int
	app.Handle = func(msg tea.Msg) (tea.Cmd, bool) {
		if _, ok := msg.(factMsg); ok {
			got++
			return nil, true
		}
		return nil, false
	}
	key(app, '?', "?") // the claim fires, the fact queues
	if got != 0 {
		t.Fatal("the fact must not land in the same dispatch that sent it")
	}
	app.Update(tickMsg{}) // the boundary
	if got != 1 {
		t.Fatalf("the fact must land exactly once, got %d", got)
	}
}
