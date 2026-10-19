package tuikit

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"charm.land/bubbletea/v2"
)

// recorder collects which scope answered a key: the routing tests read it.
type recorder struct{ hits []string }

// pane is a host-side component owning its own interaction state (a scroll) —
// exactly the kit's ownership rule: state lives in components.
type pane struct{ scroll Scroll }

// newTestApp wires one screen (status row / two regions / bottom rows) with a
// key at every scope. minW 50 keeps the generated hints inside the minimum.
func newTestApp(t *testing.T) (*App, *recorder, *pane) {
	t.Helper()
	rec := &recorder{}
	p0 := &pane{scroll: NewScroll()}
	app := NewApp()

	s := NewScreen("main", 50, 10, 2)
	s.Frame.Row(Fixed(1))
	s.Frame.Row(Fill()).Cols(Fixed(8), Fill())
	s.Frame.Row(Fixed(2))
	s.Keys = Keymap{{Keys: []string{"m"}, Help: "screen verb",
		Run: func() tea.Cmd { rec.hits = append(rec.hits, "screen"); return nil }}}
	s.RegionKeys = func(focus int) Keymap {
		verb := Keymap{{Keys: []string{"r"}, Help: "region verb",
			Run: func() tea.Cmd { rec.hits = append(rec.hits, "region1"); return nil }}}
		if focus == 0 {
			verb[0].Run = func() tea.Cmd { rec.hits = append(rec.hits, "region0"); return nil }
			return Merge(verb, ScrollKeys(&p0.scroll))
		}
		return verb
	}
	s.Layout = func() []Region { return []Region{{Title: "left"}, {Title: "right"}} }
	app.Screens.Register(s)

	app.Keys = Keymap{
		{Keys: []string{"v"}, Help: "global verb",
			Run: func() tea.Cmd { rec.hits = append(rec.hits, "global"); return nil }}, // NB: not 'g' — the scroll scope owns g/G
		{Keys: []string{"tab", "shift+tab"}, Help: "cycle focus"},
	}
	app.Chrome = func(w, h int) []string { return []string{"STATUS", "BOTTOM"} }
	return app, rec, p0
}

func key(app *App, code rune, text string) {
	app.Update(tea.KeyPressMsg{Code: code, Text: text})
}

func TestRoutingChainOrder(t *testing.T) {
	app, rec, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})

	key(app, 'r', "r") // focused region first
	if !slices.Equal(rec.hits, []string{"region0"}) {
		t.Fatalf("hits = %v, want the focused region", rec.hits)
	}
	rec.hits = nil
	key(app, 'm', "m") // screen scope
	key(app, 'v', "v") // global scope
	if !slices.Equal(rec.hits, []string{"screen", "global"}) {
		t.Fatalf("hits = %v, want screen then global", rec.hits)
	}
	rec.hits = nil
	app.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // focus moves to region 1
	key(app, 'r', "r")
	if !slices.Equal(rec.hits, []string{"region1"}) {
		t.Fatalf("unfocused region keys must not fire: %v", rec.hits)
	}
}

func TestHandlePreemptsRouting(t *testing.T) {
	app, rec, _ := newTestApp(t)
	app.Handle = func(msg tea.Msg) (tea.Cmd, bool) {
		if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "m" {
			rec.hits = append(rec.hits, "handle")
			return nil, true // the host's keyboard grab swallows the chain
		}
		return nil, false
	}
	key(app, 'm', "m")
	if !slices.Equal(rec.hits, []string{"handle"}) {
		t.Fatalf("a handled key must stop the chain: %v", rec.hits)
	}
}

func TestModalCapturesKeyboard(t *testing.T) {
	app, rec, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	app.Modals.Push(staticModal{body: "HELPBODY"})

	key(app, 'r', "r")
	if len(rec.hits) != 0 {
		t.Fatal("region keys must not leak through an open modal")
	}
	key(app, 'v', "v")
	if !slices.Equal(rec.hits, []string{"global"}) {
		t.Fatal("global verbs still answer under a modal")
	}
	rec.hits = nil
	app.Update(tea.KeyPressMsg{Code: tea.KeyEscape}) // one press pops one level
	key(app, 'r', "r")
	if !slices.Equal(rec.hits, []string{"region0"}) {
		t.Fatal("esc unwinds the modal and keys flow again")
	}
}

type staticModal struct{ body string }

func (m staticModal) View(w, h int) string { return m.body }

func TestScreenSwitchPreservesState(t *testing.T) {
	app, rec, p0 := newTestApp(t)
	key(app, 'j', "j")                            // region 0's scroll moves, state lives in the pane
	app.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // focus moves to region 1 — and stays there

	s2 := NewScreen("alt", 50, 10, 1)
	s2.Frame.Row(Fill())
	s2.Layout = func() []Region { return []Region{{Title: "alt"}} }
	s2.Keys = Keymap{{Keys: []string{"z"}, Help: "alt verb",
		Run: func() tea.Cmd { rec.hits = append(rec.hits, "alt"); return nil }}}
	app.Screens.Register(s2)

	if !app.Screens.Switch("alt") {
		t.Fatal("switch to alt")
	}
	key(app, 'z', "z")
	if !slices.Equal(rec.hits, []string{"alt"}) {
		t.Fatalf("the alt screen's scope must answer: %v", rec.hits)
	}
	rec.hits = nil
	if _, done := app.Screens.Current().Keys.Route("m"); done {
		t.Fatal("the main screen's scope must not leak across a switch")
	}
	if !app.Screens.Switch("main") {
		t.Fatal("switch back")
	}
	key(app, 'r', "r")
	if !slices.Equal(rec.hits, []string{"region1"}) {
		t.Fatalf("back on main, the focused region answers: %v", rec.hits)
	}
	if p0.scroll.top != 1 {
		t.Fatalf("component state must survive travel, scroll top = %d", p0.scroll.top)
	}
	if got := app.Screens.Current().FocusRing.Index(); got != 1 {
		t.Fatalf("focus must survive travel, index = %d", got)
	}
}

func TestViewAssembly(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	rows := strings.Split(app.View().Content, "\n")
	if len(rows) != 12 {
		t.Fatalf("frame is %d rows, want 12 (1 status + 9 regions + 2 bottom):\n%q", len(rows), rows)
	}
	if !strings.Contains(rows[0], "STATUS") {
		t.Errorf("the first row is chrome: %q", rows[0])
	}
	if !strings.Contains(rows[10], "BOTTOM") {
		t.Errorf("the bottom block closes the frame: %q", rows[10])
	}
	if !strings.Contains(app.View().Content, "left") || !strings.Contains(app.View().Content, "right") {
		t.Error("both regions render into their cells")
	}
}

func TestModalReplacesRegionArea(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	app.Modals.Push(staticModal{body: "HELPBODY"})
	out := app.View().Content
	if !strings.Contains(out, "HELPBODY") {
		t.Fatal("the modal renders")
	}
	if strings.Contains(out, "left") || strings.Contains(out, "right") {
		t.Error("regions stay hidden while the modal is open")
	}
	if !strings.Contains(out, "STATUS") || !strings.Contains(out, "BOTTOM") {
		t.Error("chrome stays visible under a modal")
	}
}

func TestTooSmallView(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 10, Height: 5})
	if out := app.View().Content; !strings.Contains(out, "terminal too small") {
		t.Errorf("an undersized terminal must be refused, got %q", out)
	}
}

func TestWindowSizeResizesFrame(t *testing.T) {
	app, _, _ := newTestApp(t)
	if strings.Contains(app.View().Content, "left") {
		t.Fatal("before any size message the frame cannot fit its minimum")
	}
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	if !app.Screens.Current().Frame.Fits() {
		t.Fatal("the size message must resize the current screen's frame")
	}
	if !strings.Contains(app.View().Content, "left") {
		t.Fatal("regions render once the frame fits")
	}
}

func TestUnconsumedKeyIsHarmless(t *testing.T) {
	app, rec, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	key(app, 'z', "z")
	if len(rec.hits) != 0 {
		t.Fatalf("an unclaimed key must be a no-op, got %v", rec.hits)
	}
}

func TestEscWithNoModalIsHarmless(t *testing.T) {
	app, rec, _ := newTestApp(t)
	app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if len(rec.hits) != 0 {
		t.Fatalf("esc with nothing open changes nothing: %v", rec.hits)
	}
}

func TestViewWithoutChromeOrScreensIsBlank(t *testing.T) {
	if out := NewApp().View().Content; out != "" {
		t.Errorf("an app without screens renders nothing, got %q", out)
	}
	app, _, _ := newTestApp(t)
	app.Chrome = nil
	if out := app.View().Content; out != "" {
		t.Errorf("without the chrome hook there is nothing to draw, got %q", out)
	}
}

func TestChromeMultilineClipAndPad(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	app.Chrome = func(w, h int) []string { return []string{"A\nB", "C"} }
	rows := strings.Split(app.View().Content, "\n")
	if len(rows) != 12 {
		t.Fatalf("frame is %d rows, want 12", len(rows))
	}
	if strings.Contains(rows[0], "B") {
		t.Errorf("a height-1 chrome row clips multi-line content: %q", rows[0])
	}
	if !strings.Contains(rows[10], "C") {
		t.Errorf("the bottom row renders its content: %q", rows[10])
	}
	if strings.TrimSpace(rows[11]) != "" {
		t.Errorf("short chrome is padded to the row height: %q", rows[11])
	}
}

func TestResizeCascade(t *testing.T) {
	app, rec, _ := newTestApp(t)
	app.Screens.All()[0].RegionResize = func(focus, w, h int) {
		rec.hits = append(rec.hits, fmt.Sprintf("r%d:%dx%d", focus, w, h))
	}
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	want := []string{"r0:8x9", "r1:72x9"} // each region sees its own cell, not the terminal
	if !slices.Equal(rec.hits, want) {
		t.Fatalf("cascade = %v, want %v", rec.hits, want)
	}
}

func TestResizeBypassesModalCapture(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Modals.Push(staticModal{body: "help"})
	saw := false
	app.Handle = func(msg tea.Msg) (tea.Cmd, bool) {
		if _, ok := msg.(tea.WindowSizeMsg); ok {
			saw = true
		}
		return nil, false
	}
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	if !saw {
		t.Fatal("geometry is not input: resize must reach the host under a modal")
	}
	if !app.Screens.Current().Frame.Fits() {
		t.Fatal("the frame must resize under a modal")
	}
}

func TestTicksFlowUnderModal(t *testing.T) {
	app, rec, _ := newTestApp(t)
	app.Modals.Push(staticModal{body: "help"})
	app.Handle = func(msg tea.Msg) (tea.Cmd, bool) {
		if _, ok := msg.(tickMsg); ok {
			rec.hits = append(rec.hits, "tick")
		}
		return nil, false
	}
	app.Update(tickMsg{})
	if !slices.Equal(rec.hits, []string{"tick"}) {
		t.Fatal("non-key messages are not captured — the machine keeps stepping under help")
	}
}

type tickMsg struct{}
