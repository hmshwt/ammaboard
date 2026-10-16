// Package ticklog persists the machine's event stream to SQLite (pure-Go driver)
// and exports it as plain text. The log is the observability layer: same inputs
// + same cycles = byte-identical event tables, which is what the determinism
// test asserts.
package ticklog

import (
	"database/sql"
	"fmt"
	"io"
	"time"

	_ "modernc.org/sqlite" // pure Go; keeps CGO_ENABLED=0 releases possible
)

// Log is an open, writable trace database.
type Log struct {
	db    *sql.DB
	runID int64
	buf   []Event
}

// Event is one logged interaction: (major, phase, component, kind, detail).
type Event struct {
	Major     int
	Phase     string
	Component string
	Kind      string
	Detail    string
}

// Open creates/overwrites the trace database and stamps a run row.
func Open(path, machineVersion, chapter string) (*Log, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	schema := `
PRAGMA journal_mode=MEMORY;
CREATE TABLE IF NOT EXISTS runs (
  id INTEGER PRIMARY KEY, chapter TEXT, machine_version TEXT, started_at TEXT);
CREATE TABLE IF NOT EXISTS events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id INTEGER, tick_major INTEGER, phase TEXT, component TEXT, kind TEXT, detail TEXT);
CREATE TABLE IF NOT EXISTS snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id INTEGER, tick_major INTEGER, tick_minor TEXT, kind TEXT, state_blob TEXT);
`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	res, err := db.Exec(`INSERT INTO runs (chapter, machine_version, started_at) VALUES (?,?,?)`,
		chapter, machineVersion, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		db.Close()
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Log{db: db, runID: id}, nil
}

// Add buffers one event; events are written in batches per major tick.
func (l *Log) Add(e Event) { l.buf = append(l.buf, e) }

// Flush writes all buffered events in one transaction.
func (l *Log) Flush() error {
	if len(l.buf) == 0 {
		return nil
	}
	tx, err := l.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO events
		(run_id, tick_major, phase, component, kind, detail) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, e := range l.buf {
		if _, err := stmt.Exec(l.runID, e.Major, e.Phase, e.Component, e.Kind, e.Detail); err != nil {
			stmt.Close()
			tx.Rollback()
			return err
		}
	}
	if err := stmt.Close(); err != nil {
		return err
	}
	l.buf = l.buf[:0]
	return tx.Commit()
}

// Snapshot stores a full-state checkpoint (chapter start/end, faults).
func (l *Log) Snapshot(major int, minor, kind, stateBlob string) error {
	_, err := l.db.Exec(`INSERT INTO snapshots (run_id, tick_major, tick_minor, kind, state_blob)
		VALUES (?,?,?,?,?)`, l.runID, major, minor, kind, stateBlob)
	return err
}

// Close flushes and closes the database.
func (l *Log) Close() error {
	if err := l.Flush(); err != nil {
		return err
	}
	return l.db.Close()
}

// Export reads every event of the latest run, in order, as readable text.
// Golden-trace tests diff exactly this output between two identical runs.
func Export(path string, w io.Writer) error {
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	rows, err := db.Query(`
		SELECT tick_major, phase, component, kind, detail FROM events
		WHERE run_id = (SELECT MAX(id) FROM runs)
		ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var major int
		var phase, comp, kind, detail string
		if err := rows.Scan(&major, &phase, &comp, &kind, &detail); err != nil {
			return err
		}
		fmt.Fprintf(w, "%6d %-8s %-6s %-8s %s\n", major, phase, comp, kind, detail)
	}
	return rows.Err()
}

// EventsReadback returns the ordered event stream of the latest run (tests use
// this to compare two runs for byte-identical determinism).
func EventsReadback(path string) ([]Event, error) {
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`
		SELECT tick_major, phase, component, kind, detail FROM events
		WHERE run_id = (SELECT MAX(id) FROM runs)
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.Major, &e.Phase, &e.Component, &e.Kind, &e.Detail); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
