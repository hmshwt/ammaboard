package tuikit

// trace_test.go — the receipts: every dispatch leaves a TraceEvent, the
// ladder walk is visible, capture is visible, commands carry provenance, and
// identical consecutive receipts compress so heartbeats cannot drown the log.

import (
	"strings"
	"testing"

	"charm.land/bubbletea/v2"
)

func lastReceipt(t *testing.T, app *App, kind TraceKind) TraceEvent {
	t.Helper()
	evs := app.Trace.Snapshot()
	for i := len(evs) - 1; i >= 0; i-- {
		if evs[i].Kind == kind {
			return evs[i]
		}
	}
	t.Fatalf("no %v receipt", kind)
	return TraceEvent{}
}

func TestTraceRecordsTheLadder(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})

	key(app, 'r', "r") // the focused region answers first — the walk stops there
	ev := lastReceipt(t, app, TraceKey)
	if ev.Key != "r" || !strings.Contains(ev.From, "region 0") {
		t.Fatalf("receipt = %+v", ev)
	}
	if !strings.Contains(ev.What, `region 0:HIT`) {
		t.Errorf("receipt walk = %q", ev.What)
	}
	if strings.Contains(ev.What, "screen") || strings.Contains(ev.What, "global") {
		t.Errorf("first consumer wins: nothing past the hit is consulted: %q", ev.What)
	}

	key(app, 'v', "v") // global answers after two misses
	ev = lastReceipt(t, app, TraceKey)
	for _, want := range []string{`region 0:miss`, "screen:miss", `global:HIT`} {
		if !strings.Contains(ev.What, want) {
			t.Errorf("receipt walk = %q, want %q in it", ev.What, want)
		}
	}
}

func TestTraceRecordsCapture(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	app.Handle = func(msg tea.Msg) (tea.Cmd, bool) {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return nil, true // the grab swallows every key
		}
		return nil, false
	}
	key(app, 'q', "q")
	ev := lastReceipt(t, app, TraceKey)
	if ev.From != "Handle" || !strings.Contains(ev.What, "grab") {
		t.Errorf("a grab must be receipted: %+v", ev)
	}
}

func TestTraceRecordsModalCapture(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	app.Modals.Push(staticModal{body: "help"})

	key(app, 'r', "r")
	if ev := lastReceipt(t, app, TraceKey); !strings.Contains(ev.What, "nothing leaked") {
		t.Errorf("captured keys must say so: %+v", ev)
	}
	app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if ev := lastReceipt(t, app, TraceKey); !strings.Contains(ev.What, "esc popped") {
		t.Errorf("esc under a modal must be receipted: %+v", ev)
	}
}

func TestTraceRecordsResize(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	ev := lastReceipt(t, app, TraceMsg)
	if !strings.Contains(ev.What, "resize 80×12") || !strings.Contains(ev.What, "cascade") {
		t.Errorf("geometry is receipted too: %+v", ev)
	}
}

func TestTraceCmdProvenance(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	app.Keys = Keymap{{Keys: []string{"p"}, Help: "produce",
		Run: func() tea.Cmd { return func() tea.Msg { return tickMsg{} } }}}

	_, cmd := app.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	if cmd == nil {
		t.Fatal("the winning binding returned a command")
	}
	if _, ok := cmd().(tickMsg); !ok {
		t.Fatal("the command must produce its message")
	}
	for _, ev := range app.Trace.Snapshot() {
		if ev.Kind == TraceCmd && strings.Contains(ev.From, "global") && strings.Contains(ev.What, "tickMsg") {
			return // the produced message carries its producer's label
		}
	}
	t.Error("no provenance receipt for the produced message")
}

func TestTraceCompressesRepeats(t *testing.T) {
	app, _, _ := newTestApp(t)
	saw := 0
	app.Handle = func(msg tea.Msg) (tea.Cmd, bool) {
		if _, ok := msg.(tickMsg); ok {
			saw++
		}
		return nil, true
	}
	for i := 0; i < 3; i++ {
		app.Update(tickMsg{})
	}
	if saw != 3 {
		t.Fatalf("every tick must still be delivered, got %d", saw)
	}
	ev := lastReceipt(t, app, TraceMsg)
	if ev.Count != 3 {
		t.Fatalf("identical receipts must compress into one with Count=3: %+v", ev)
	}
}

func TestTraceCanStaySilent(t *testing.T) {
	app, rec, _ := newTestApp(t)
	app.Trace = nil
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	key(app, 'v', "v")
	if len(rec.hits) != 1 {
		t.Fatal("silenced receipts must not change behavior")
	}
}

func TestAssertRoute(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	AssertRoute(t, app, "r", "region 0:HIT")
	AssertRoute(t, app, "m", "screen:HIT")
	AssertRoute(t, app, "v", "global:HIT")
	AssertRoute(t, app, "z", "unconsumed")
}
