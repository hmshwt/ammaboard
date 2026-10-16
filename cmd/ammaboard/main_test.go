package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const program = `
        add_number r1, r0, 777
        add_number r3, r0, 2048
        store_word r1, 0(r3)
        load_word  r2, 0(r3)
        halt
`

// End-to-end smoke through the real wiring: assemble → run → SQLite → text export.
func TestRunEndToEnd(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "prog.amm")
	db := filepath.Join(dir, "amma.db")
	trace := filepath.Join(dir, "trace.txt")
	if err := os.WriteFile(src, []byte(program), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(src, db, trace, 1000, false, false); err != nil {
		t.Fatal(err)
	}
	tb, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"fetch", "decode", "load_word", "bus_write_post", "latch"} {
		if !strings.Contains(string(tb), want) {
			t.Errorf("trace.txt missing %q", want)
		}
	}
	if _, err := os.Stat(db); err != nil {
		t.Errorf("trace db missing: %v", err)
	}
}

// Pluggability through the real wiring path: -echo flips one boolean in wiring.
func TestRunWithEchoDevice(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "prog.amm")
	db := filepath.Join(dir, "amma.db")
	trace := filepath.Join(dir, "trace.txt")
	if err := os.WriteFile(src, []byte(program), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(src, db, trace, 1000, true, false); err != nil {
		t.Fatal(err)
	}
	// Same program, same result — the extra device changed nothing else.
	tb, _ := os.ReadFile(trace)
	if !strings.Contains(string(tb), "clean stop") {
		t.Error("program should still halt cleanly with the echo device attached")
	}
}
