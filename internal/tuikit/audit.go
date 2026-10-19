package tuikit

import (
	"fmt"
	"strings"
)

// Severity grades a wiring issue.
type Severity int

const (
	// SeverityWarn is printed but not fatal.
	SeverityWarn Severity = iota
	// SeverityError is fatal at startup and fails AssertCleanWiring.
	SeverityError
)

// String names the severity for reports.
func (s Severity) String() string {
	if s == SeverityError {
		return "error"
	}
	return "warn"
}

// Issue is one finding of the wiring audit.
type Issue struct {
	Severity Severity
	Where    string
	What     string
}

// String renders one report line: "error: screen main: …".
func (i Issue) String() string {
	return fmt.Sprintf("%s: %s: %s", i.Severity, i.Where, i.What)
}

// Verify audits the assembled wiring: key scopes, screen registry, frames,
// chrome shape. It is a pure derivation over wiring values — the same
// philosophy as rendering — so tests, `--check`, and startup all run the
// exact same audit. Errors are fatal at startup; warnings print.
func (a *App) Verify() []Issue {
	var out []Issue
	if a.Screens == nil || len(a.Screens.All()) == 0 {
		return append(out, Issue{SeverityError, "app", "no screens registered"})
	}
	for _, name := range a.Screens.dupes {
		out = append(out, Issue{SeverityError, "screens",
			fmt.Sprintf("screen %q registered twice — the last registration wins", name)})
	}
	for _, scr := range a.Screens.All() {
		a.verifyScreen(scr, &out)
	}
	if scr := a.current(); scr != nil {
		if hints := Merge(a.Keys, scr.Keys).Hints(); hints != "" && len(hints) > scr.Frame.minW {
			out = append(out, Issue{SeverityWarn, scr.where("hints"),
				fmt.Sprintf("hints line is %d columns, minimum width is %d", len(hints), scr.Frame.minW)})
		}
	}
	if a.Chrome != nil {
		want := 0
		if scr := a.current(); scr != nil {
			for _, row := range scr.Frame.rows {
				if len(row.cells) == 0 {
					want++
				}
			}
		}
		if want > 0 {
			if got := len(a.Chrome(a.w, a.h)); got != want {
				out = append(out, Issue{SeverityWarn, "app",
					fmt.Sprintf("chrome returns %d lines, the current screen has %d chrome rows", got, want)})
			}
		}
	}
	return out
}

// verifyScreen audits one screen: registry sanity, frame solvability at the
// declared minimum, focusable count, and key-scope discipline. Regions are
// mutually exclusive (only the focused one routes), so two regions may claim
// the same key; global vs screen vs region may not.
func (a *App) verifyScreen(scr *Screen, out *[]Issue) {
	if scr.Frame == nil {
		*out = append(*out, Issue{SeverityError, scr.Name, "no frame"})
		return
	}
	if scr.Layout == nil {
		*out = append(*out, Issue{SeverityError, scr.Name, "no layout"})
	}
	scr.Frame.audit(scr.Name, out)
	var regions []Region
	if scr.Layout != nil {
		regions = scr.Layout()
		if n := len(regions); n != scr.Focusables {
			*out = append(*out, Issue{SeverityError, scr.Name,
				fmt.Sprintf("layout renders %d regions, screen declares %d focusables", n, scr.Focusables)})
		}
	}
	// Constraints: the frame resolves deterministically at the declared
	// minimum, so every region's promise is checkable exactly there.
	if len(regions) > 0 {
		cells := scr.Frame.cellRectsAt(scr.Frame.minW, scr.Frame.minH)
		for i, reg := range regions {
			if i >= len(cells) {
				break // count mismatch already reported
			}
			c, cell := reg.Constraints, cells[i]
			if cell.W < c.MinW || cell.H < c.MinH {
				*out = append(*out, Issue{SeverityError, scr.where(fmt.Sprintf("region %d", i)),
					fmt.Sprintf("at the declared minimum the cell is %d×%d, the region declares ≥%d×%d",
						cell.W, cell.H, c.MinW, c.MinH)})
			}
		}
	}

	reserved := map[string]bool{}
	for _, k := range ReservedKeys {
		reserved[k] = true
	}
	seen := map[string]bool{}
	for _, k := range a.Keys.Keys() {
		seen[k] = true
	}
	checkScope := func(name string, km Keymap) {
		for _, k := range km.Keys() {
			if seen[k] {
				*out = append(*out, Issue{SeverityError, scr.where(name),
					fmt.Sprintf("key %q already claimed by the global or screen scope", k)})
			}
			if reserved[k] {
				*out = append(*out, Issue{SeverityError, scr.where(name),
					fmt.Sprintf("reserved key %q claimed outside the global scope", k)})
			}
		}
	}
	checkScope("screen", scr.Keys)
	if dup := duplicateKeys(scr.Keys); len(dup) > 0 {
		*out = append(*out, Issue{SeverityWarn, scr.where("screen"),
			fmt.Sprintf("duplicate keys inside one keymap (Merge order decides): %s",
				strings.Join(dup, ", "))})
	}
	for _, k := range scr.Keys.Keys() {
		seen[k] = true
	}
	if scr.RegionKeys != nil {
		for i := 0; i < scr.Focusables; i++ {
			km := scr.RegionKeys(i)
			checkScope(fmt.Sprintf("region %d", i), km)
			if dup := duplicateKeys(km); len(dup) > 0 {
				*out = append(*out, Issue{SeverityWarn, scr.where(fmt.Sprintf("region %d", i)),
					fmt.Sprintf("duplicate keys inside one keymap (Merge order decides): %s",
						strings.Join(dup, ", "))})
			}
		}
	}
	for _, b := range scr.Keys {
		if b.Help == "" {
			*out = append(*out, Issue{SeverityWarn, scr.where("screen"),
				fmt.Sprintf("binding %v carries no Help — the generated help has a hole", b.Keys)})
		}
	}
	if scr.FullBleed && len(scr.Keys) == 0 && (scr.RegionKeys == nil || scr.Focusables == 0) {
		*out = append(*out, Issue{SeverityWarn, scr.Name,
			"FullBleed screen defines no own keys — leaving it depends entirely on global bindings"})
	}
}

// duplicateKeys lists keys claimed twice inside one keymap.
func duplicateKeys(km Keymap) []string {
	seen := map[string]bool{}
	var out []string
	for _, b := range km {
		for _, k := range b.Keys {
			if seen[k] {
				out = append(out, k)
			}
			seen[k] = true
		}
	}
	return out
}

func (s *Screen) where(part string) string {
	if part == "" {
		return "screen " + s.Name
	}
	return fmt.Sprintf("screen %s/%s", s.Name, part)
}

// TestingT is the subset of *testing.T the kit needs, so the kernel never
// imports the testing package.
type TestingT interface {
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
}

// AssertCleanWiring is the audit's test face: any error fails the test
// (warnings are logged for the dev to judge). Call it on the fully wired app
// in the host package's tests — CI is the real pre-ship gate.
func AssertCleanWiring(t TestingT, a *App) {
	for _, is := range a.Verify() {
		if is.Severity == SeverityError {
			t.Errorf("wiring: %s", is)
		} else {
			t.Logf("wiring: %s", is)
		}
	}
}
