// Command ammaboard runs the AmmaBoard-1 PoC: assemble a program, execute it
// on the motherboard, persist the full tick log to SQLite, and export it as
// readable text.
//
// Usage:
//
//	ammaboard run program.amm [-o ammaboard.db] [-trace trace.txt] [-max 100000] [-echo]
//
// The -echo flag wires an extra EchoDevice at 0x4000 — the pluggability proof
// (zero edits anywhere else).
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"ammaboard/internal/asm"
	"ammaboard/internal/machine"
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
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintln(os.Stderr, "usage: ammaboard run program.amm [-o ammaboard.db] [-trace trace.txt] [-max 100000] [-echo] [-dump-full]")
		os.Exit(2)
	}
	var dbPath, tracePath = defaultDB, defaultTrace
	var maxStr string
	var echo, dumpFull bool
	valueFlags := map[string]*string{
		"-o":     &dbPath,
		"-trace": &tracePath,
		"-max":   &maxStr,
	}
	// Hand-rolled scan so flags may appear before or after the program path
	// (Go's flag package stops at the first positional). Classic loop: the
	// body skips flag values by advancing i.
	var pos []string
	args := os.Args[2:]
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
			echo = true
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
	maxMajors := 100000
	if maxStr != "" {
		n, err := strconv.Atoi(maxStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ammaboard: -max wants a number, got %q\n", maxStr)
			os.Exit(2)
		}
		maxMajors = n
	}
	if len(pos) != 1 {
		fmt.Fprintln(os.Stderr, "usage: ammaboard run program.amm [flags]")
		os.Exit(2)
	}
	if err := run(pos[0], dbPath, tracePath, maxMajors, echo, dumpFull); err != nil {
		fmt.Fprintln(os.Stderr, "ammaboard:", err)
		os.Exit(1)
	}
}

func run(srcPath, dbPath, tracePath string, maxMajors int, echo, dumpFull bool) error {
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

	w := buildBoard(log, echo)
	ram, board, cpu := w.ram, w.board, w.cpu
	if err := ram.Load(entryPoint, words); err != nil {
		return err
	}

	if err := log.Snapshot(0, "BOOT", "initial", board.StateString(dumpFull)); err != nil {
		return err
	}
	board.Run(maxMajors)
	if err := log.Snapshot(board.Major, "FROZEN", "final", board.StateString(dumpFull)); err != nil {
		return err
	}
	if err := log.Flush(); err != nil { // persist events before reading them back
		return err
	}

	f := cpu.HaltedBy()
	fmt.Printf("assembled %d words | ran %d major ticks | stopped: %s\n",
		len(words), board.Major, haltSummary(f))
	fmt.Println(strings.Repeat("─", 56))
	if f.Kind != "halt" {
		// Not a clean stop: the kernel explains itself. The panel is also
		// frozen into the log as its own snapshot — a fault is data.
		bs := machine.BlueScreen(cpu, board, ram, readEvents(dbPath))
		fmt.Print(bs)
		if err := log.Snapshot(board.Major, string(f.Phase), "bluescreen", bs); err != nil {
			return err
		}
	} else {
		fmt.Print(board.StateString(dumpFull))
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

func haltSummary(f machine.Fault) string {
	if f.Kind == "halt" {
		return "clean halt"
	}
	return fmt.Sprintf("%s (pc=0x%04x addr=0x%04x phase=%s)", f.Kind, f.PC, f.Addr, f.Phase)
}

func exportText(dbPath, tracePath string) error {
	f, err := os.Create(tracePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return ticklog.Export(dbPath, f)
}
