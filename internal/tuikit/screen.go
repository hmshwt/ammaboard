package tuikit

// Screen is a named full view: derived presentation plus input grammar over
// one shared state (the postgres-view analogy). A screen owns its frame (and
// minimum size), its focus ring, and its keymap scope; the machine beneath is
// read-only and never copied.
type Screen struct {
	Name      string
	Frame     *Frame
	FullBleed bool   // skip global chrome (the fault screen hides the shell)
	Keys      Keymap // screen-scope bindings
	Layout    func() []Region

	// RegionKeys is the keys of the i-th focusable region — the routing
	// chain's entry point, composed by the host with Merge. nil means the
	// screen has no region scope.
	RegionKeys func(focus int) Keymap

	// RegionResize is the resize cascade's leaf: on tea.WindowSizeMsg the kit
	// translates the terminal size into each region's own cell viewport and
	// hands it here. The host forwards to stateful children (an input's
	// width). nil is fine — pure regions derive from their cell every frame
	// and need no events.
	RegionResize func(focus int, w, h int)

	FocusRing  FocusRing
	Focusables int // declared region count; the ring and the audit key off it
}

// NewScreen declares a screen with its minimum size and region count.
func NewScreen(name string, minW, minH, focusables int) *Screen {
	return &Screen{
		Name:       name,
		Frame:      NewFrame(minW, minH),
		FocusRing:  NewFocusRing(focusables),
		Focusables: focusables,
	}
}

// ScreenSet is the flat registry of screens plus the current pointer. No
// stack: transitions are host policy (fault → switch, reset → switch back).
type ScreenSet struct {
	byName  map[string]*Screen
	order   []string // registration order — deterministic audits and reports
	current string
	dupes   []string
}

// NewScreenSet is an empty registry.
func NewScreenSet() *ScreenSet {
	return &ScreenSet{byName: map[string]*Screen{}}
}

// Register adds a screen; the first registration becomes current.
// Re-registering a name overwrites and is recorded for the audit.
func (s *ScreenSet) Register(scr *Screen) {
	if _, ok := s.byName[scr.Name]; ok {
		s.dupes = append(s.dupes, scr.Name)
	} else {
		s.order = append(s.order, scr.Name)
	}
	if s.current == "" {
		s.current = scr.Name
	}
	s.byName[scr.Name] = scr
}

// Switch selects the current screen by name; false for an unknown name.
// Switching changes nothing else — every screen's view state is its own.
func (s *ScreenSet) Switch(name string) bool {
	if _, ok := s.byName[name]; !ok {
		return false
	}
	s.current = name
	return true
}

// Current is the active screen, or nil before the first registration.
func (s *ScreenSet) Current() *Screen { return s.byName[s.current] }

// All lists the screens in registration order.
func (s *ScreenSet) All() []*Screen {
	out := make([]*Screen, 0, len(s.order))
	for _, n := range s.order {
		out = append(out, s.byName[n])
	}
	return out
}
