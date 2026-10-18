package tuikit

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbletea/v2"
)

// ReservedKeys are global verbs no other scope may shadow.
var ReservedKeys = []string{"q", "ctrl+c", "esc"}

// Binding is one key claim: the keys it answers, the help line that shows up
// in hints and help (generated — help cannot lie), and the command it runs.
// Run nil means the key is consumed but produces no command (display-only, or
// handled deeper in the app).
type Binding struct {
	Keys []string
	Help string
	Run  func() tea.Cmd
}

// Bind starts a fluent binding: Bind("space").Help("run / pause").Run(fn).
// Run terminates the chain and returns the Binding; display-only bindings end
// with Run(nil).
func Bind(keys ...string) bindReq { return bindReq{Binding{Keys: keys}} }

type bindReq struct{ b Binding }

// Help sets the generated help line.
func (r bindReq) Help(h string) bindReq { r.b.Help = h; return r }

// Run sets the command and completes the binding.
func (r bindReq) Run(fn func() tea.Cmd) Binding { r.b.Run = fn; return r.b }

// Keymap is an ordered set of bindings. Order is precedence: Route takes the
// first match, and Merge concatenates — earlier keymaps win.
type Keymap []Binding

// Route resolves one key (the tea.KeyPressMsg.String() form). Reports whether
// a binding consumed it.
func (km Keymap) Route(key string) (tea.Cmd, bool) {
	for _, b := range km {
		for _, k := range b.Keys {
			if k != key {
				continue
			}
			if b.Run != nil {
				return b.Run(), true
			}
			return nil, true
		}
	}
	return nil, false
}

// Hints renders the keymap as one bottom-line string: " tab focus · s step …".
func (km Keymap) Hints() string {
	parts := make([]string, 0, len(km))
	for _, b := range km {
		if b.Help == "" || len(b.Keys) == 0 {
			continue
		}
		parts = append(parts, b.Keys[0]+" "+b.Help)
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " · ")
}

// Help renders the keymap as aligned two-column lines for the help modal.
func (km Keymap) Help() []Line {
	w := 0
	for _, b := range km {
		if n := len(strings.Join(b.Keys, "/")); n > w {
			w = n
		}
	}
	out := make([]Line, 0, len(km))
	for _, b := range km {
		out = append(out, Plain(fmt.Sprintf("  %-*s   %s", w, strings.Join(b.Keys, "/"), b.Help)))
	}
	return out
}

// Keys lists the distinct key strings a keymap answers, sorted.
func (km Keymap) Keys() []string {
	set := map[string]bool{}
	for _, b := range km {
		for _, k := range b.Keys {
			set[k] = true
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Merge concatenates keymaps: earlier children/s scopes win on conflicts.
func Merge(kms ...Keymap) Keymap {
	var out Keymap
	for _, km := range kms {
		out = append(out, km...)
	}
	return out
}

// Collisions reports key strings claimed by more than one of the given
// keymaps — the region author's own test for accidental double claims among
// its children. Deliberate precedence is fine; it must be visible in Merge
// order and pinned by a test.
func Collisions(kms ...Keymap) []string {
	seen := map[string]int{}
	for _, km := range kms {
		for _, k := range km.Keys() {
			seen[k]++
		}
	}
	out := make([]string, 0, len(seen))
	for k, n := range seen {
		if n > 1 {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// NoShadowing checks scope discipline: no scope outside the global one may
// duplicate a global key or claim a ReservedKey. Returns readable violations.
func NoShadowing(global Keymap, scopes ...Keymap) []string {
	reserved := map[string]bool{}
	for _, k := range ReservedKeys {
		reserved[k] = true
	}
	globalKeys := map[string]bool{}
	for _, k := range global.Keys() {
		globalKeys[k] = true
	}
	var out []string
	for i, km := range scopes {
		for _, k := range km.Keys() {
			switch {
			case globalKeys[k]:
				out = append(out, fmt.Sprintf("scope %d shadows global key %q", i, k))
			case reserved[k]:
				out = append(out, fmt.Sprintf("scope %d claims reserved key %q", i, k))
			}
		}
	}
	sort.Strings(out)
	return out
}

// ScrollKeys is the generated pane-scroll keymap backed by s: pane keys stay
// documented because they are data.
func ScrollKeys(s *Scroll) Keymap {
	press := func(k string) func() tea.Cmd {
		return func() tea.Cmd { s.Key(k); return nil }
	}
	return Keymap{
		Bind("j", "down").Help("scroll down").Run(press("j")),
		Bind("k", "up").Help("scroll up").Run(press("k")),
		Bind("g").Help("jump to top").Run(press("g")),
		Bind("G").Help("jump to bottom").Run(press("G")),
		Bind("f").Help("re-follow").Run(press("f")),
	}
}
