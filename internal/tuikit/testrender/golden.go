package testrender

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// update regenerates golden files; `go test -update` rewrites them with the
// current output, and a diff on mismatch makes every visual change a
// reviewable one.
var update = flag.Bool("update", false, "rewrite golden files with the current output")

var goldenDir = "testdata"

// SetGoldenDir points the golden helpers at another directory (tests use it
// to sandbox their fixtures).
func SetGoldenDir(dir string) { goldenDir = dir }

// TB is the subset of *testing.T the helpers need — testrender never imports
// the testing package.
type TB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// Golden compares got against <dir>/<name>.golden, failing with a diff on
// mismatch.
func Golden(t TB, name, got string) {
	t.Helper()
	path := filepath.Join(goldenDir, name+".golden")
	if *update {
		if err := os.MkdirAll(goldenDir, 0o755); err != nil {
			t.Fatalf("golden: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden %s: %v — run `go test ./... -update` to generate it", path, err)
	}
	if got != string(want) {
		t.Fatalf("golden mismatch in %s:\n%s", path, diff(string(want), got))
	}
}

// GoldenChars writes or compares the level-1 golden: characters only.
func GoldenChars(t TB, name string, g Grid) { Golden(t, name, strings.Join(g.Chars(), "\n")) }

// GoldenGrid writes or compares the level-2 golden: attr lines over char lines.
func GoldenGrid(t TB, name string, g Grid) { Golden(t, name, g.String()) }

// diff renders a small window around the first mismatching line.
func diff(want, got string) string {
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	i := 0
	for i < len(wl) && i < len(gl) && wl[i] == gl[i] {
		i++
	}
	var b strings.Builder
	fmt.Fprintf(&b, "first mismatch at line %d (want %d lines, got %d)\n", i+1, len(wl), len(gl))
	for j := max(i-2, 0); j < i; j++ {
		b.WriteString("  " + wl[j] + "\n")
	}
	for j := i; j < min(i+3, len(wl)); j++ {
		fmt.Fprintf(&b, "-%s\n", wl[j])
	}
	for j := i; j < min(i+3, len(gl)); j++ {
		fmt.Fprintf(&b, "+%s\n", gl[j])
	}
	return b.String()
}
