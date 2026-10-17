package ticklog

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPersistedTraceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "amma.db")

	log, err := Open(db, "AmmaBoard-1", "poc")
	if err != nil {
		t.Fatal(err)
	}
	log.Add(Event{Major: 1, Phase: "FETCH", Component: "cpu", Kind: "fetch", Detail: "pc=0x0000"})
	log.Add(Event{Major: 1, Phase: "EXECUTE", Component: "board", Kind: "latch", Detail: "1 write(s) committed"})
	if err := log.Snapshot(1, "FROZEN", "final", "state-blob"); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}

	events, err := EventsReadback(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Kind != "fetch" || events[1].Kind != "latch" {
		t.Fatalf("roundtrip mismatch: %+v", events)
	}

	var buf bytes.Buffer
	if err := Export(db, &buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("fetch")) {
		t.Error("text export should contain the fetch event")
	}
}

func TestExportDeterministicAcrossRuns(t *testing.T) {
	// PoC criterion: two identical runs produce byte-identical event tables.
	dir := t.TempDir()
	write := func(name string) {
		log, err := Open(filepath.Join(dir, name), "AmmaBoard-1", "poc")
		if err != nil {
			t.Fatal(err)
		}
		for major := 1; major <= 5; major++ {
			log.Add(Event{Major: major, Phase: "FETCH", Component: "cpu", Kind: "fetch", Detail: "pc=0x0000"})
			log.Add(Event{Major: major, Component: "board", Kind: "latch", Detail: "end of major tick"})
		}
		log.Close()
	}
	write("a.db")
	write("b.db")

	var a, b bytes.Buffer
	if err := Export(filepath.Join(dir, "a.db"), &a); err != nil {
		t.Fatal(err)
	}
	if err := Export(filepath.Join(dir, "b.db"), &b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Error("identical runs must produce identical trace exports")
	}
	if _, err := os.Stat(filepath.Join(dir, "a.db")); err != nil {
		t.Error("db file should exist")
	}
}

// The trace is the human-readable surface: legend, column header, one
// indented block per clock cycle, and a plain-language summary per cycle.
func TestExportReadableFormat(t *testing.T) {
	dir := t.TempDir()
	log, err := Open(filepath.Join(dir, "t.db"), "AmmaBoard-2", "poc")
	if err != nil {
		t.Fatal(err)
	}
	log.Add(Event{Major: 1, Component: "board", Kind: "major"})
	log.Add(Event{Major: 1, Phase: "FETCH", Component: "cpu", Kind: "fetch", Detail: "pc=0x0000"})
	log.Add(Event{Major: 1, Phase: "DECODE", Component: "cpu", Kind: "decode", Detail: "add_number rd=r1 rs=r0 rt=r0 imm=0"})
	log.Add(Event{Major: 1, Phase: "EXECUTE", Component: "cpu", Kind: "alu", Detail: "r1 <= 0 + 0 = 0 (was 0)"})
	log.Add(Event{Major: 1, Component: "board", Kind: "latch", Detail: "end of major tick"})
	log.Add(Event{Major: 2, Component: "board", Kind: "major"})
	log.Add(Event{Major: 2, Phase: "EXECUTE", Component: "cpu", Kind: "halt", Detail: "clean stop at pc=0x0000 (next fetch 0x0004)"})
	log.Close()

	var out bytes.Buffer
	if err := Export(filepath.Join(dir, "t.db"), &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"# AmmaBoard-2 trace",
		"; cycle 1 — add_number: register_1 was 0, now is 0",
		strings.Repeat("-", 60),
		"       1 FETCH    cpu    fetch    pc=0x0000",
		"       1 EXECUTE  cpu    alu      r1 <= 0 + 0 = 0",
		"; cycle 2 — clean stop at program counter 0x0000 (next fetch 0x0004)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("trace export missing %q\n%s", want, s)
		}
	}
	// The comment announces the cycle before its events; the events are indented.
	comment := strings.Index(s, "; cycle 1")
	events := strings.Index(s, "1 FETCH")
	if comment < 0 || events < 0 || comment > events {
		t.Errorf("cycle comment must precede its indented events:\n%s", s)
	}
	if !strings.Contains(s, "\n       1 FETCH") {
		t.Error("cycle contents must be indented under their dashed rule")
	}
}
