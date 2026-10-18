package tuikit

import (
	"fmt"
	"strings"
	"testing"
)

// issuesBy collects the audit findings of one severity.
func issuesBy(issues []Issue, sev Severity) []string {
	var out []string
	for _, is := range issues {
		if is.Severity == sev {
			out = append(out, is.What)
		}
	}
	return out
}

func TestVerifyCleanWiring(t *testing.T) {
	app, _, _ := newTestApp(t)
	if errs := issuesBy(app.Verify(), SeverityError); len(errs) != 0 {
		t.Errorf("a fully wired app must audit clean, got errors: %v", errs)
	}
}

func TestVerifyNoScreens(t *testing.T) {
	app := NewApp()
	issues := app.Verify()
	if len(issues) != 1 || issues[0].Severity != SeverityError {
		t.Fatalf("an empty registry is one error, got %v", issues)
	}
}

func TestVerifyDetectsReservedAndDuplicates(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Keys = Merge(app.Keys, Keymap{{Keys: []string{"q"}, Help: "quit"}}) // q is now a global verb
	app.Screens.All()[0].Keys = Keymap{
		{Keys: []string{"q"}, Help: "shadow the global quit"}, // reserved + global dup
		{Keys: []string{"v"}, Help: "dup of a global key"},    // cross-scope dup
		{Keys: []string{"x"}, Help: "one claim"},
		{Keys: []string{"x"}, Help: "same claim twice"}, // within-scope dup → warn
		{Keys: []string{"y"}, Help: ""},                 // empty Help → warn
	}
	issues := app.Verify()
	errs := strings.Join(issuesBy(issues, SeverityError), "; ")
	for _, want := range []string{
		`reserved key "q"`,
		`key "q" already claimed`,
		`key "v" already claimed`,
	} {
		if !strings.Contains(errs, want) {
			t.Errorf("errors missing %q: %s", want, errs)
		}
	}
	warns := strings.Join(issuesBy(issues, SeverityWarn), "; ")
	if !strings.Contains(warns, "duplicate keys") {
		t.Errorf("want a within-scope duplicate warning, got: %s", warns)
	}
	if !strings.Contains(warns, "no Help") {
		t.Errorf("want an empty-Help warning, got: %s", warns)
	}
}

func TestVerifyDetectsFocusableMismatch(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Screens.All()[0].Layout = func() []Region { return []Region{{Title: "one"}} }
	if errs := issuesBy(app.Verify(), SeverityError); !strings.Contains(strings.Join(errs, "; "), "focusables") {
		t.Errorf("layout/declared mismatch must be an error, got %v", errs)
	}
}

func TestVerifyDetectsUnsolvableFrame(t *testing.T) {
	app, _, _ := newTestApp(t)
	s := app.Screens.All()[0]
	s.Frame = NewFrame(50, 10)
	s.Frame.Row(Fixed(20)) // taller than the declared minimum, and no Fill
	errs := strings.Join(issuesBy(app.Verify(), SeverityError), "; ")
	for _, want := range []string{"fixed rows need", "no Fill row"} {
		if !strings.Contains(errs, want) {
			t.Errorf("errors missing %q: %s", want, errs)
		}
	}
}

func TestVerifyFullBleedWarning(t *testing.T) {
	app, _, _ := newTestApp(t)
	fb := NewScreen("fault", 50, 10, 1)
	fb.FullBleed = true
	fb.Frame.Row(Fill())
	fb.Layout = func() []Region { return []Region{{Title: "fault"}} }
	app.Screens.Register(fb)
	found := false
	for _, w := range issuesBy(app.Verify(), SeverityWarn) {
		if strings.Contains(w, "FullBleed") {
			found = true
		}
	}
	if !found {
		t.Error("a keyless FullBleed screen deserves the escape-hatch warning")
	}
}

func TestVerifyChromeCountWarning(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Chrome = func(w, h int) []string { return []string{"only one line"} }
	found := false
	for _, w := range issuesBy(app.Verify(), SeverityWarn) {
		if strings.Contains(w, "chrome") {
			found = true
		}
	}
	if !found {
		t.Error("a chrome/row mismatch deserves a warning")
	}
}

// fakeT records AssertCleanWiring's verdicts instead of failing the test.
type fakeT struct{ errs, logs []string }

func (f *fakeT) Errorf(format string, args ...any) {
	f.errs = append(f.errs, fmt.Sprintf(format, args...))
}
func (f *fakeT) Logf(format string, args ...any) {
	f.logs = append(f.logs, fmt.Sprintf(format, args...))
}

func TestAssertCleanWiring(t *testing.T) {
	app, _, _ := newTestApp(t)
	verdict := &fakeT{}
	AssertCleanWiring(verdict, app)
	if len(verdict.errs) != 0 {
		t.Errorf("a clean wiring must assert clean, got %v", verdict.errs)
	}

	app.Screens.All()[0].Keys = Keymap{{Keys: []string{"q"}, Help: "shadow"}}
	verdict = &fakeT{}
	AssertCleanWiring(verdict, app)
	if len(verdict.errs) == 0 {
		t.Error("a reserved-key shadow must fail the assert")
	}
}

func TestVerifyDuplicateScreenName(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Screens.Register(NewScreen("main", 50, 10, 1)) // same name again
	errs := issuesBy(app.Verify(), SeverityError)
	if !strings.Contains(strings.Join(errs, "; "), "registered twice") {
		t.Errorf("re-registration must be an error, got %v", errs)
	}
}

func TestVerifyHintsWidthWarning(t *testing.T) {
	app, _, _ := newTestApp(t)
	s := app.Screens.All()[0]
	s.Frame = NewFrame(10, 10) // narrower than the generated hints line
	s.Frame.Row(Fixed(1))
	s.Frame.Row(Fill()).Cols(Fixed(8), Fill())
	s.Frame.Row(Fixed(2))
	found := false
	for _, w := range issuesBy(app.Verify(), SeverityWarn) {
		if strings.Contains(w, "hints") {
			found = true
		}
	}
	if !found {
		t.Error("a hints line wider than the declared minimum deserves a warning")
	}
}
