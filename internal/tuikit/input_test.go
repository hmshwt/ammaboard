package tuikit

import (
	"slices"
	"strings"
	"testing"

	"charm.land/bubbletea/v2"
)

func TestRouteConsumesAndRuns(t *testing.T) {
	ran := false
	km := Keymap{
		{Keys: []string{"a"}, Help: "run", Run: func() tea.Cmd { ran = true; return nil }},
		{Keys: []string{"b"}, Help: "consume only"},
	}
	if _, done := km.Route("a"); !done || !ran {
		t.Fatal("a binding with Run must run and report consumption")
	}
	if cmd, done := km.Route("b"); !done || cmd != nil {
		t.Fatal("a binding without Run consumes without a command")
	}
	if _, done := km.Route("z"); done {
		t.Fatal("an unclaimed key must not be consumed")
	}
}

func TestFirstMatchWins(t *testing.T) {
	second := false
	km := Keymap{
		{Keys: []string{"x"}, Help: "first"},
		{Keys: []string{"x"}, Help: "second", Run: func() tea.Cmd { second = true; return nil }},
	}
	km.Route("x")
	if second {
		t.Fatal("Route must take the first match — order is precedence")
	}
}

func TestMergePrecedence(t *testing.T) {
	hit := ""
	km := Merge(
		Keymap{{Keys: []string{"x"}, Help: "left", Run: func() tea.Cmd { hit = "left"; return nil }}},
		Keymap{{Keys: []string{"x"}, Help: "right", Run: func() tea.Cmd { hit = "right"; return nil }}},
	)
	km.Route("x")
	if hit != "left" {
		t.Fatalf("Merge order decided %q, want left", hit)
	}
}

func TestHintsAndHelpGenerated(t *testing.T) {
	km := Keymap{
		{Keys: []string{"space"}, Help: "run / pause"},
		{Keys: []string{"q", "ctrl+c"}, Help: "quit"},
		{Keys: []string{"secret"}, Help: ""},
	}
	hints := km.Hints()
	for _, want := range []string{" space run / pause", "q quit", " · "} {
		if !strings.Contains(hints, want) {
			t.Errorf("hints %q missing %q", hints, want)
		}
	}
	if strings.Contains(hints, "secret") {
		t.Error("bindings without Help stay out of the hints")
	}
	help := km.Help()
	if len(help) != 3 {
		t.Fatalf("help lists %d lines, want 3 (every binding)", len(help))
	}
	if !strings.Contains(help[1].Text, "q/ctrl+c") || !strings.Contains(help[1].Text, "quit") {
		t.Errorf("help line = %q", help[1].Text)
	}
}

func TestCollisionsAndNoShadowing(t *testing.T) {
	global := Keymap{{Keys: []string{"q"}, Help: "quit"}}
	if got := Collisions(global, Keymap{{Keys: []string{"x"}, Help: "ok"}}); len(got) != 0 {
		t.Errorf("disjoint scopes collide: %v", got)
	}
	if got := Collisions(global, Keymap{{Keys: []string{"q"}, Help: "dup"}}); !slices.Equal(got, []string{"q"}) {
		t.Errorf("collisions = %v, want [q]", got)
	}
	violations := NoShadowing(global,
		Keymap{{Keys: []string{"x"}, Help: "fine"}},
		Keymap{{Keys: []string{"q"}, Help: "shadow"}, {Keys: []string{"esc"}, Help: "reserved"}},
	)
	if len(violations) != 2 {
		t.Fatalf("violations = %v, want the shadowed q and the reserved esc", violations)
	}
}

func TestBindBuilder(t *testing.T) {
	b := Bind("space").Help("run / pause").Run(nil)
	if b.Keys[0] != "space" || b.Help != "run / pause" || b.Run != nil {
		t.Fatalf("builder produced %+v", b)
	}
}

func TestScrollKeysDriveTheScroll(t *testing.T) {
	s := NewScroll()
	km := ScrollKeys(&s)
	if _, done := km.Route("j"); !done || s.top != 1 || s.Following() {
		t.Fatal("j scrolls one line and detaches")
	}
	if _, done := km.Route("f"); !done || !s.Following() {
		t.Fatal("f re-follows")
	}
	if _, done := km.Route("G"); !done || s.Window(0, 100, 10) != 90 {
		t.Fatal("G pins the bottom")
	}
}

func TestScrollSemantics(t *testing.T) {
	s := NewScroll()
	if !s.Following() {
		t.Fatal("a new scroll follows")
	}
	if got := s.Window(5, 100, 10); got != 2 {
		t.Errorf("follow window = %d, want 2 (track − height/%d)", got, FollowPin)
	}
	s.Key("j")
	if got := s.Window(50, 100, 10); got != 1 {
		t.Errorf("manual window = %d, want 1 (scrolled once)", got)
	}
	s.Key("k")
	if got := s.Window(50, 100, 10); got != 0 {
		t.Errorf("k at the top clamps to 0, got %d", got)
	}
	s.Key("G")
	if got := s.Window(0, 100, 10); got != 90 {
		t.Errorf("G clamps to the bottom, got %d", got)
	}
	s.Key("g")
	if s.top != 0 || s.Following() {
		t.Error("g jumps to the top and stays manual")
	}
	s.Reset()
	if !s.Following() {
		t.Error("reset returns to follow")
	}
	if got := s.Window(5, 4, 10); got != 0 {
		t.Errorf("short content pins to 0 in any mode, got %d", got)
	}
}

func TestFocusRing(t *testing.T) {
	r := NewFocusRing(3)
	r.Next()
	if !r.Is(1) {
		t.Fatal("next moves to 1")
	}
	r.Next()
	r.Next()
	if !r.Is(0) {
		t.Fatal("next wraps to 0")
	}
	r.Prev()
	if !r.Is(2) {
		t.Fatal("prev wraps to 2")
	}
	r.Reset()
	if r.Index() != 0 {
		t.Fatal("reset returns to the first region")
	}
	zero := NewFocusRing(0)
	zero.Next() // must not panic
	zero.Prev()
}
