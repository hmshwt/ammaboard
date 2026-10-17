// Command ammaboard runs and inspects the AmmaBoard machine.
//
// Usage:
//
//	ammaboard run program.amm [-o ammaboard.db] [-trace trace.txt] [-max N] [-echo] [-dump-full]
//	ammaboard tui [program.amm]
//
// run assembles a program, executes it, persists the full tick log to SQLite,
// and exports it as readable text. The -echo flag wires an extra echo device
// at 0x4000 — the pluggability proof (zero edits anywhere else). -max caps the
// run (default 10000 major ticks, 0 = no cap). A run that hits the cap without
// halting is paused, not stopped: it reports "budget exhausted (program did
// not halt)".
//
// tui is the interactive terminal UI (docs/ui.md): three panes over the live
// machine plus a command shell. Without a program it loads the built-in
// counter demo.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"ammaboard/internal/asm"
	"ammaboard/internal/machine"
	"ammaboard/internal/machine/bluescreen"
	"ammaboard/internal/ticklog"
)

const (
	machineVersion = machine.MachineVersion
	ramBase        = 0x0000
	ramSize        = 16 * 1024 // 16KB default by the sizing rule
	echoBase       = 0x4000    // just past RAM: the broker routes by window order
	timerBase      = 0x4020    // day-8 timer MMIO (interval @+0, pending/ack @+4)
	entryPoint     = 0x0000
	defaultDB      = "ammaboard.db"
	defaultTrace   = "trace.txt"
	// defaultMaxMajors caps a run so a runaway program fails fast without
	// flooding the log: an order of magnitude above the longest chapter
	// program, and about a second of work in the worst case. 0 = no cap.
	defaultMaxMajors = 10000
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ammaboard run program.amm [-o ammaboard.db] [-trace trace.txt] [-max 10000] [-echo] [-dump-full]")
	fmt.Fprintln(os.Stderr, "       ammaboard tui [program.amm]   interactive UI; no file = built-in counter demo")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	args := os.Args[2:]
	switch os.Args[1] {
	case "run":
		runMain(args)
	case "tui":
		if len(args) > 1 {
			usage()
			os.Exit(2)
		}
		src := ""
		if len(args) == 1 {
			src = args[0]
		}
		if err := tui(src); err != nil {
			fmt.Fprintln(os.Stderr, "ammaboard:", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

// runMain is the batch path: assemble, run to a stop or the -max cap, log to
// SQLite, export text, print the verdict.
func runMain(args []string) {
	var dbPath, tracePath = defaultDB, defaultTrace
	var maxStr string
	var echoOn, dumpFull bool
	valueFlags := map[string]*string{
		"-o":     &dbPath,
		"-trace": &tracePath,
		"-max":   &maxStr,
	}
	// Hand-rolled scan so flags may appear before or after the program path
	// (Go's flag package stops at the first positional). Classic loop: the
	// body skips flag values by advancing i.
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if target, ok := valueFlags[a]; ok {
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "ammaboard: %s needs a value\n", a)
				os.Exit(2)
			}
			i++
			*target = args[i]
			continue
		}
		switch a {
		case "-echo":
			echoOn = true
		case "-dump-full":
			dumpFull = true
		default:
			if strings.HasPrefix(a, "-") {
				fmt.Fprintf(os.Stderr, "ammaboard: unknown flag %q\n", a)
				os.Exit(2)
			}
			pos = append(pos, a)
		}
	}
	maxMajors := defaultMaxMajors
	if maxStr != "" {
		n, err := strconv.Atoi(maxStr)
		if err != nil || n < 0 {
			fmt.Fprintf(os.Stderr, "ammaboard: -max wants a non-negative number, got %q (0 = no cap)\n", maxStr)
			os.Exit(2)
		}
		maxMajors = n
	}
	if len(pos) != 1 {
		fmt.Fprintln(os.Stderr, "usage: ammaboard run program.amm [flags]")
		os.Exit(2)
	}
	if err := run(pos[0], dbPath, tracePath, maxMajors, echoOn, dumpFull); err != nil {
		fmt.Fprintln(os.Stderr, "ammaboard:", err)
		os.Exit(1)
	}
}

func run(srcPath, dbPath, tracePath string, maxMajors int, echoOn, dumpFull bool) error {
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	words, err := asm.Assemble(string(src))
	if err != nil {
		return fmt.Errorf("assemble: %w", err)
	}

	log, err := ticklog.Open(dbPath, machineVersion, "poc")
	if err != nil {
		return err
	}
	defer log.Close()

	w := buildBoard(func(e machine.Event) { log.Add(ticklog.Event(e)) }, echoOn)
	if err := w.ram.Load(entryPoint, words); err != nil {
		return err
	}

	if err := log.Snapshot(0, "BOOT", "initial", w.board.StateString(dumpFull)); err != nil {
		return err
	}
	w.board.Run(maxMajors)
	reason := w.board.StopReason()
	snapKind := "PAUSED" // budget cut-off: mid-flight, resumable
	if reason == machine.StopHalt || reason == machine.StopFault {
		snapKind = "FROZEN"
	}
	if err := log.Snapshot(w.board.Major, snapKind, "final", w.board.StateString(dumpFull)); err != nil {
		return err
	}
	if err := log.Flush(); err != nil { // persist events before reading them back
		return err
	}

	fmt.Printf("assembled %d words | ran %d major ticks | %s\n",
		len(words), w.board.Major, stopSummary(w.board))
	fmt.Println(strings.Repeat("─", 56))
	if reason == machine.StopFault {
		// Not a clean stop: the kernel explains itself. The panel is also
		// frozen into the log as its own snapshot — a fault is data.
		f := w.board.Fault()
		bs := bluescreen.Render(w.cpu, w.board, w.ram, readEvents(dbPath))
		fmt.Print(bs)
		if err := log.Snapshot(w.board.Major, string(f.Phase), "bluescreen", bs); err != nil {
			return err
		}
	} else {
		fmt.Print(w.board.StateString(dumpFull))
	}
	fmt.Println(strings.Repeat("─", 56))
	fmt.Printf("trace: %s + %s\n", dbPath, tracePath)
	return exportText(dbPath, tracePath)
}

// readEvents converts the persisted event stream into machine events for the
// Blue Screen cause chain. Best effort: an empty chain beats a failed render.
func readEvents(dbPath string) []machine.Event {
	te, err := ticklog.EventsReadback(dbPath)
	if err != nil {
		return nil
	}
	out := make([]machine.Event, len(te))
	for i, e := range te {
		out[i] = machine.Event{Major: e.Major, Phase: e.Phase, Component: e.Component, Kind: e.Kind, Detail: e.Detail}
	}
	return out
}

// stopSummary is the one-line verdict, in the same vocabulary as StopReason.
// A budget cut-off is "paused", not "stopped": the program neither halted nor
// faulted — it just ran out of harness.
func stopSummary(board *machine.Motherboard) string {
	switch board.StopReason() {
	case machine.StopHalt:
		return "stopped: clean halt"
	case machine.StopFault:
		f := board.Fault()
		return fmt.Sprintf("stopped: %s (pc=0x%04x addr=0x%04x phase=%s)", f.Kind, f.PC, f.Addr, f.Phase)
	case machine.StopBudget:
		return "paused: budget exhausted (program did not halt)"
	default:
		return "not started"
	}
}

func exportText(dbPath, tracePath string) error {
	f, err := os.Create(tracePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return ticklog.Export(dbPath, f)
}
