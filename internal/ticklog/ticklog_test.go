package ticklog

import (
	"bytes"
	"database/sql"
	"io"
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

// ---- schema_version: stamped, migrated, and enforced on read ----

// querySchemaVersion reads one run row's schema_version with a raw connection.
func querySchemaVersion(t *testing.T, path string, runID int64) int64 {
	t.Helper()
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var v int64
	if err := raw.QueryRow(`SELECT schema_version FROM runs WHERE id = ?`, runID).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestSchemaVersionStamped(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "v.db")
	log, err := Open(db, "AmmaBoard-2", "poc")
	if err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	if got := querySchemaVersion(t, db, 1); got != SchemaVersion {
		t.Errorf("new runs must carry schema_version %d, got %d", SchemaVersion, got)
	}
}

// A file written before versioning existed is migrated in place: the column
// appears, old rows stay 0 (legacy), and new runs carry the current version.
func TestSchemaVersionMigratesLegacyFile(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "legacy.db")
	raw, err := sql.Open("sqlite", db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = raw.Exec(`
CREATE TABLE runs (id INTEGER PRIMARY KEY, chapter TEXT, machine_version TEXT, started_at TEXT);
CREATE TABLE events (id INTEGER PRIMARY KEY AUTOINCREMENT, run_id INTEGER,
  tick_major INTEGER, phase TEXT, component TEXT, kind TEXT, detail TEXT);
CREATE TABLE snapshots (id INTEGER PRIMARY KEY AUTOINCREMENT, run_id INTEGER,
  tick_major INTEGER, tick_minor TEXT, kind TEXT, state_blob TEXT);
INSERT INTO runs (chapter, machine_version, started_at) VALUES ('poc','AmmaBoard-1','2026-01-01T00:00:00Z');
INSERT INTO events (run_id, tick_major, phase, component, kind, detail) VALUES (1,1,'FETCH','cpu','fetch','pc=0x0000');`)
	if err != nil {
		t.Fatal(err)
	}
	raw.Close()

	log, err := Open(db, "AmmaBoard-2", "poc")
	if err != nil {
		t.Fatalf("Open must migrate a legacy file, got: %v", err)
	}
	log.Add(Event{Major: 1, Component: "board", Kind: "major"})
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}

	if got := querySchemaVersion(t, db, 1); got != 0 {
		t.Errorf("legacy run must stay version 0, got %d", got)
	}
	if got := querySchemaVersion(t, db, 2); got != SchemaVersion {
		t.Errorf("new run after migration must be version %d, got %d", SchemaVersion, got)
	}
	// The legacy events are still present (readable, never rewritten).
	raw2, err := sql.Open("sqlite", db)
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err := raw2.QueryRow(`SELECT COUNT(*) FROM events WHERE run_id = 1`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	raw2.Close()
	if n != 1 {
		t.Errorf("legacy event row lost in migration: count=%d, want 1", n)
	}
	// Export still works (it exports the newest run, per the read contract).
	var buf bytes.Buffer
	if err := Export(db, &buf); err != nil {
		t.Fatalf("export after migration: %v", err)
	}
}

// A trace written by a newer build is refused, not misread.
func TestReaderRefusesNewerSchema(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "future.db")
	log, err := Open(db, "AmmaBoard-99", "poc")
	if err != nil {
		t.Fatal(err)
	}
	log.Add(Event{Major: 1, Component: "board", Kind: "major"})
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite", db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`UPDATE runs SET schema_version = ?`, SchemaVersion+1); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	if err := Export(db, io.Discard); err == nil {
		t.Error("Export must refuse a newer schema version")
	}
	if _, err := EventsReadback(db); err == nil {
		t.Error("EventsReadback must refuse a newer schema version")
	}
}
