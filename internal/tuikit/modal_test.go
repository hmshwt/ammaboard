package tuikit

import "testing"

func TestModalStackLifecycle(t *testing.T) {
	var s ModalStack
	if s.Active() || s.Pop() {
		t.Fatal("an empty stack is inactive and cannot pop")
	}
	if _, ok := s.Top(); ok {
		t.Fatal("an empty stack has no top")
	}
	first, second := staticModal{body: "1"}, staticModal{body: "2"}
	s.Push(first)
	s.Push(second)
	if !s.Active() {
		t.Fatal("push activates the stack")
	}
	if top, _ := s.Top(); top != second {
		t.Fatal("top is the newest modal")
	}
	if !s.Pop() || !s.Pop() || s.Pop() {
		t.Fatal("pop unwinds one level at a time, then reports empty")
	}
	if s.Active() {
		t.Fatal("the unwound stack is inactive")
	}
}
