package tuikit

import "testing"

func TestScreenSetRegistrationAndSwitch(t *testing.T) {
	set := NewScreenSet()
	s1 := NewScreen("main", 50, 10, 1)
	s2 := NewScreen("alt", 50, 10, 1)
	set.Register(s1)
	set.Register(s2)
	if set.Current() != s1 {
		t.Fatal("the first registration becomes current")
	}
	if got := set.All(); len(got) != 2 || got[0] != s1 || got[1] != s2 {
		t.Fatal("All lists screens in registration order")
	}
	if set.Switch("nope") {
		t.Fatal("switching to an unknown screen fails")
	}
	if !set.Switch("alt") || set.Current() != s2 {
		t.Fatal("switch selects by name")
	}
}

func TestNewScreenDefaults(t *testing.T) {
	s := NewScreen("x", 10, 5, 3)
	if s.Frame == nil {
		t.Fatal("NewScreen builds the frame")
	}
	s.Frame.Resize(10, 5)
	if !s.Frame.Fits() {
		t.Error("the declared minimum fits itself")
	}
	for i := 0; i < 3; i++ {
		s.FocusRing.Next()
	}
	if s.FocusRing.Index() != 0 {
		t.Errorf("the ring wraps at %d focusables", s.Focusables)
	}
}
