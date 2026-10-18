package tuikit

// FollowPin is the follow-mode anchor: the tracked line is pinned
// 1/FollowPin of the body height from the top.
const FollowPin = 3

// Scroll is one pane's scroll state. Follow pins the window to what the
// machine is doing (the pc line, the newest events); manual scrolling turns
// follow off until Reset or f. Scroll owns no machine state — it is a window
// over a derivation.
type Scroll struct {
	top    int
	follow bool
}

// NewScroll starts in follow mode.
func NewScroll() Scroll { return Scroll{follow: true} }

// Key applies one scroll key (j/k/up/down/g/G/f); reports whether it was
// consumed.
func (s *Scroll) Key(key string) bool {
	switch key {
	case "j", "down":
		s.top++
		s.follow = false
	case "k", "up":
		s.top = max(s.top-1, 0)
		s.follow = false
	case "g":
		*s = Scroll{}
	case "G":
		*s = Scroll{top: 1 << 30} // clamps to "bottom" at window time
	case "f":
		s.follow = true
	default:
		return false
	}
	return true
}

// Window resolves the first visible line: follow pins the tracked line
// 1/FollowPin from the top; manual scroll clamps to the content.
func (s *Scroll) Window(track, total, height int) int {
	if total <= height {
		return 0
	}
	if s.follow {
		return Clamp(track-height/FollowPin, 0, total-height)
	}
	return Clamp(s.top, 0, total-height)
}

// Reset returns to follow mode (machine reset fans out to every screen).
func (s *Scroll) Reset() { *s = NewScroll() }

// Following reports follow mode.
func (s *Scroll) Following() bool { return s.follow }
