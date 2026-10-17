// Package ticklog persists the machine's event stream to SQLite (pure-Go driver)
// and exports it as plain text. The log is the observability layer: same inputs
// + same cycles = byte-identical event tables, which is what the determinism
// test asserts.
package ticklog

import (
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
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

// exportEv is one row of the events table, as the exporter sees it.
type exportEv struct {
	major                     int
	phase, comp, kind, detail string
}

// Export renders the latest run as the human-readable trace: a legend, a
// column header, then one indented block per clock cycle between the two
// clock edges, closed by a one-line summary of what the cycle accomplished.
// Rendering only — the events table is untouched, so golden-trace diffs and
// the determinism test see the same rows.
func Export(path string, w io.Writer) error {
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()

	var chapter, machineVersion string
	err = db.QueryRow(`SELECT chapter, machine_version FROM runs
		WHERE id = (SELECT MAX(id) FROM runs)`).Scan(&chapter, &machineVersion)
	if err != nil {
		return err
	}
	rows, err := db.Query(`
		SELECT tick_major, phase, component, kind, detail FROM events
		WHERE run_id = (SELECT MAX(id) FROM runs)
		ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var events []exportEv
	for rows.Next() {
		var e exportEv
		if err := rows.Scan(&e.major, &e.phase, &e.comp, &e.kind, &e.detail); err != nil {
			return err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	fmt.Fprintf(w, "# %s trace — chapter %q\n", machineVersion, chapter)
	fmt.Fprintln(w, "# columns: tick | phase | component | event | detail")
	fmt.Fprintln(w, "# one tick = one clock cycle: FETCH, DECODE, EXECUTE, then the latch —")
	fmt.Fprintln(w, "# values written during the cycle become visible to every component there.")
	fmt.Fprintln(w, "# each cycle is announced by a comment (its number and what it")
	fmt.Fprintln(w, "# accomplishes), then a dashed rule, then the cycle's events.")
	fmt.Fprintln(w, "# comments use full terms: register_0..register_7, program counter,")
	fmt.Fprintln(w, "# memory[…], interrupts enabled/disabled.")

	var cycles [][]exportEv
	cur := []exportEv{}
	for _, e := range events {
		if e.kind == "major" { // clock edge: the previous cycle is complete
			cycles = append(cycles, cur)
			cur = []exportEv{}
			continue
		}
		cur = append(cur, e)
	}
	cycles = append(cycles, cur)

	for i, c := range cycles {
		if len(c) == 0 {
			continue
		}
		if i > 0 {
			fmt.Fprintln(w)
		}
		if s := cycleSummary(c); s != "" {
			fmt.Fprintf(w, "; cycle %d — %s\n", c[0].major, s)
		} else {
			fmt.Fprintf(w, "; cycle %d\n", c[0].major)
		}
		fmt.Fprintln(w, strings.Repeat("-", 60))
		for _, e := range c {
			if e.kind == "latch" && e.detail == "end of major tick" {
				continue // the dashed rule is the cycle boundary; the row is bookkeeping
			}
			phase := e.phase
			if phase == "" {
				phase = "clock" // clock-edge rows carry no CPU phase
			}
			fmt.Fprintf(w, "  %6d %-8s %-6s %-8s %s\n", e.major, phase, e.comp, e.kind, e.detail)
		}
	}
	return nil
}

// cycleSummary condenses one cycle into the leading comment: the instruction
// decoded plus its effect, phrased as "X was A, now is B". Interrupt delivery
// outranks both — the notable event of such a cycle is the delivery itself.
func cycleSummary(cycle []exportEv) string {
	instr, effectKind, effectDetail := "", "", ""
	for _, e := range cycle {
		if e.kind == "interrupt" {
			return expandTerms("interrupt: " + e.detail)
		}
		if instr == "" && e.kind == "decode" {
			if f := strings.Fields(e.detail); len(f) > 0 {
				instr = f[0]
			}
		}
		if effectKind == "" {
			switch e.kind {
			case "halt", "fault", "rfi", "branch", "call", "goto",
				"store", "store_byte", "load", "load_byte", "alu", "status":
				effectKind, effectDetail = e.kind, e.detail
			}
		}
	}
	if effectKind == "" {
		return ""
	}
	if s, prefixed := effectComment(effectKind, effectDetail, cycle); prefixed {
		if instr != "" {
			return instr + ": " + s
		}
		return s
	}
	// No human phrasing for this kind: fall back to the raw detail, expanded.
	if instr != "" && effectDetail != "" {
		return expandTerms(instr + ": " + effectDetail)
	}
	if effectDetail != "" {
		return expandTerms(effectDetail)
	}
	return instr
}

// effectComment phrases an effect as "subject was A, now is B" — the subject
// is whatever the instruction changed. prefixed=false marks kinds whose raw
// detail already reads fine (halt, fault, status).
func effectComment(kind, detail string, cycle []exportEv) (string, bool) {
	switch kind {
	case "alu":
		if m := reALU.FindStringSubmatch(detail); m != nil {
			return fmt.Sprintf("register_%s was %s, now is %s",
				m[1], humanReg(parseUint(m[3])), humanReg(parseUint(m[2]))), true
		}
	case "load", "load_byte":
		if m := reLoad.FindStringSubmatch(detail); m != nil {
			return fmt.Sprintf("register_%s was %s, now is %s (read memory 0x%s)",
				m[1], humanReg(parseUint(m[4])), humanReg(parseUint(m[3])), m[2]), true
		}
	case "store", "store_byte":
		byteForm := kind == "store_byte"
		if m := reCommit.FindStringSubmatch(cycleCommitDetail(cycle, byteForm)); m != nil {
			s := fmt.Sprintf("memory 0x%s was %s, now is %s", m[1], m[4], m[3])
			if m[2] != "" {
				s += " (byte)"
			}
			return s, true
		}
		return expandTerms(detail), true // store never committed: show the post as-is
	case "branch":
		if m := reBranchTaken.FindStringSubmatch(detail); m != nil {
			return fmt.Sprintf("program counter was 0x%s, now is 0x%s (branch taken)", m[1], m[2]), true
		}
		if m := reBranchStay.FindStringSubmatch(detail); m != nil {
			return fmt.Sprintf("program counter stays 0x%s (branch not taken)", m[1]), true
		}
	case "call":
		if m := reCall.FindStringSubmatch(detail); m != nil {
			return fmt.Sprintf("program counter was 0x%s, now is 0x%s — return address register_7 = 0x%s",
				m[1], m[3], m[2]), true
		}
	case "goto":
		if m := reGoto.FindStringSubmatch(detail); m != nil {
			return fmt.Sprintf("program counter was 0x%s, now is 0x%s (from register_%s)",
				m[1], m[3], m[2]), true
		}
	case "rfi":
		if m := reRFI.FindStringSubmatch(detail); m != nil {
			state := "disabled"
			if v, err := strconv.ParseUint(m[3], 16, 32); err == nil && v&1 != 0 {
				state = "enabled"
			}
			return fmt.Sprintf("program counter restored to 0x%s — interrupts %s (status 0x%s)",
				m[2], state, m[3]), true
		}
	}
	return "", false
}

// cycleCommitDetail returns the commit event's detail for a byte or word
// store posted in this cycle (the commit carries the old value).
func cycleCommitDetail(cycle []exportEv, byteForm bool) string {
	for _, e := range cycle {
		if e.kind != "latch" || !strings.HasPrefix(e.detail, "commit ") {
			continue
		}
		if strings.Contains(e.detail, " byte ") == byteForm {
			return e.detail
		}
	}
	return ""
}

var (
	reALU         = regexp.MustCompile(`^r([0-7]) <= .* = (-?\d+)(?: \(was (\d+)\))?$`)
	reLoad        = regexp.MustCompile(`^r([0-7]) <= mem\[0x([0-9a-f]+)\] = (\d+)(?: \(was (\d+)\))?$`)
	reCommit      = regexp.MustCompile(`^commit \w+\[0x([0-9a-f]+)\] <= (byte )?(0x[0-9a-f]+) \(was (0x[0-9a-f]+)\)$`)
	reBranchTaken = regexp.MustCompile(`^at=0x([0-9a-f]+) next=0x([0-9a-f]+) -> 0x([0-9a-f]+) \(taken\)$`)
	reBranchStay  = regexp.MustCompile(`^at=0x([0-9a-f]+) next=0x([0-9a-f]+) \(not taken\)$`)
	reCall        = regexp.MustCompile(`^at=0x([0-9a-f]+) r7=0x([0-9a-f]+) -> 0x([0-9a-f]+)$`)
	reGoto        = regexp.MustCompile(`^at=0x([0-9a-f]+) -> r([0-7]) \(0x([0-9a-f]+)\)$`)
	reRFI         = regexp.MustCompile(`^at=0x([0-9a-f]+) pc -> 0x([0-9a-f]+) \(status 0x([0-9a-f]+)\)$`)
)

// parseUint parses a decimal event field; negative values (only possible in
// older rows) wrap through int32 like the machine's arithmetic does.
func parseUint(s string) uint32 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	if v < 0 {
		return uint32(int32(v))
	}
	return uint32(v)
}

// humanReg renders a register value with every interpretation visible: the
// unsigned decimal the machine computes, plus signed and hex when they differ.
func humanReg(u uint32) string {
	if u >= 0x80000000 {
		return fmt.Sprintf("%d (= %d signed, 0x%08x)", u, int32(u), u)
	}
	return strconv.FormatUint(uint64(u), 10)
}

// commentReplacements spell out machine shorthand in comments — the trace
// rows stay compact, the prose layer onboards: r3 → register_3,
// pc → program counter, mem[ → memory[, ie=1 → interrupts enabled.
var commentReplacements = []struct {
	re   *regexp.Regexp
	repl string
}{
	{regexp.MustCompile(`pc=0x`), "program counter 0x"},
	{regexp.MustCompile(`\bpc\b`), "program counter"},
	{regexp.MustCompile(`\br([0-7])\b`), "register_${1}"},
	{regexp.MustCompile(`\bmem\[`), "memory["},
	{regexp.MustCompile(`\bie=1\b`), "interrupts enabled"},
	{regexp.MustCompile(`\bie=0\b`), "interrupts disabled"},
}

// expandTerms applies the comment shorthand expansions.
func expandTerms(s string) string {
	for _, r := range commentReplacements {
		s = r.re.ReplaceAllString(s, r.repl)
	}
	return s
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
