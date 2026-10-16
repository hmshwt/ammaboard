package ticklog

import (
	"bytes"
	"os"
	"path/filepath"
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
