package tuikit

// FocusRing is a screen's pane focus: one ring per screen, owned by the
// screen (not the panes). The index selects which region enters key routing
// first and gets the accent border; unfocused regions contribute nothing to
// routing.
type FocusRing struct {
	i, n int
}

// NewFocusRing rings count regions.
func NewFocusRing(count int) FocusRing { return FocusRing{n: count} }

// Next cycles forward (tab).
func (f *FocusRing) Next() {
	if f.n > 0 {
		f.i = (f.i + 1) % f.n
	}
}

// Prev cycles backward (shift+tab).
func (f *FocusRing) Prev() {
	if f.n > 0 {
		f.i = (f.i + f.n - 1) % f.n
	}
}

// Index is the focused region.
func (f *FocusRing) Index() int { return f.i }

// Is reports whether region i is focused.
func (f *FocusRing) Is(i int) bool { return i == f.i }

// Reset returns focus to the first region.
func (f *FocusRing) Reset() { f.i = 0 }
