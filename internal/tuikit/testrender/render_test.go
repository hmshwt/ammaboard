package testrender

import (
	"fmt"
	"strings"
	"testing"
)

var testPal = Palette{
	{}:                                   'N',
	{Fg: [3]int{0, 255, 0}, HasFg: true}: 'G',
	{Fg: [3]int{255, 0, 0}, HasFg: true, Bold: true}: 'R',
	{Bg: [3]int{0, 0, 255}, HasBg: true}:             'B',
}

func TestStripANSI(t *testing.T) {
	in := "\x1b[1;38;2;1;2;3mab\x1b[m\x1b[48;2;0;0;9m c\x1b[0m"
	if got := StripANSI(in); got != "ab c" {
		t.Errorf("strip = %q, want %q", got, "ab c")
	}
}

func TestDecodeAttrsAndChars(t *testing.T) {
	in := "ab\x1b[38;2;0;255;0mCD\x1b[m\nef"
	g, err := Decode(in, testPal)
	if err != nil {
		t.Fatal(err)
	}
	if g.W != 4 || g.H != 2 {
		t.Fatalf("grid = %dx%d, want 4×2", g.W, g.H)
	}
	if got := g.Cell(0, 0).A; got != 'N' {
		t.Errorf("unstyled cell = %c, want N", got)
	}
	if got := g.Cell(2, 0); got.A != 'G' || got.R != 'C' {
		t.Errorf("styled cell = %+v, want G/C", got)
	}
	if got := g.Cell(0, 1).A; got != 'N' {
		t.Errorf("style must reset across lines, got %c", got)
	}
	chars := strings.Join(g.Chars(), "\n")
	if chars != "abCD\nef  " {
		t.Errorf("chars = %q, want %q", chars, "abCD\nef  ")
	}
	s := g.String()
	if !strings.Contains(s, "attr: NNGG\nchar: abCD") {
		t.Errorf("level-2 view = %q", s)
	}
}

func TestDecodeBoldAndBackground(t *testing.T) {
	in := "\x1b[1;38;2;255;0;0mR\x1b[m\x1b[48;2;0;0;255mB"
	g, err := Decode(in, testPal)
	if err != nil {
		t.Fatal(err)
	}
	if got := g.Cell(0, 0).A; got != 'R' {
		t.Errorf("bold+fg = %c, want R", got)
	}
	if got := g.Cell(1, 0).A; got != 'B' {
		t.Errorf("background = %c, want B", got)
	}
}

func TestDecodeUnknownStyleFails(t *testing.T) {
	_, err := Decode("\x1b[38;2;9;9;9m?", testPal)
	if err == nil || !strings.Contains(err.Error(), "unknown styles") {
		t.Fatalf("unknown colors must fail loudly, got %v", err)
	}
}

type fakeRenderTB struct{ fatals []string }

func (f *fakeRenderTB) Helper() {}
func (f *fakeRenderTB) Fatalf(format string, args ...any) {
	f.fatals = append(f.fatals, fmt.Sprintf(format, args...))
}

func TestGoldenUpdateAndCompare(t *testing.T) {
	dir := t.TempDir()
	old := goldenDir
	goldenDir = dir
	defer func() { goldenDir = old }()

	ft := &fakeRenderTB{}
	*update = true
	Golden(ft, "sample", "v1\nv2")
	*update = false
	if len(ft.fatals) != 0 {
		t.Fatalf("writing a golden must not fail: %v", ft.fatals)
	}

	Golden(ft, "sample", "v1\nv2")
	if len(ft.fatals) != 0 {
		t.Fatalf("identical output must pass: %v", ft.fatals)
	}

	Golden(ft, "sample", "v1\nCHANGED")
	if len(ft.fatals) != 1 || !strings.Contains(ft.fatals[0], "CHANGED") {
		t.Errorf("a mismatch must fail with a diff, got %v", ft.fatals)
	}
}
