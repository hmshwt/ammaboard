// Package tui is the AmmaBoard terminal UI, assembled on tuikit
// (docs/tuikit.md): three panes over the live machine plus a command shell.
// The kit owns layout, screens, focus, modals, and key routing; this package
// owns the machine and its control state (running, speed, the shell). Machine
// state is read-only to the views — every pane is a pure per-frame derivation,
// and the only machine writes are the machine's own stepping.
package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ammaboard/internal/machine"
	"ammaboard/internal/machine/cpu"
	"ammaboard/internal/machine/ram"
	"ammaboard/internal/tuikit"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const (
	// minWidth/minHeight are the board screen's declared minimum — the same
	// geometry the frame rows and the audit assume.
	minWidth  = 141
	minHeight = 28

	leftW  = 44
	rightW = 54 // tick log gets the wider column; the machine pane gives up 10

	bottomH = 6 // boxed shell (5) + keyhints (1)

	// maxSpeed bounds cycles-per-frame so a hot loop can still be watched.
	maxSpeed = 1024
)

// frameInterval is the UI heartbeat. One frame every 33ms; a running machine
// advances `speed` major ticks per frame.
const frameInterval = time.Second / 30

// frameMsg drives the run loop: the model re-issues it every frame while the
// program is running.
type frameMsg time.Time

func frame() tea.Cmd {
	return tea.Tick(frameInterval, func(t time.Time) tea.Msg { return frameMsg(t) })
}

// Model is the TUI root: the kit's wiring root (App) plus the machine and its
// control state. Machine ownership is delegated — newMachine (re)builds a
// loaded board, so reset is exact.
type Model struct {
	*tuikit.App

	name       string
	words      []uint32
	board      *machine.Motherboard
	proc       *cpu.CPU
	mem        *ram.RAM
	events     *EventLog
	newMachine func() (*machine.Motherboard, *cpu.CPU, *ram.RAM)

	running bool
	speed   int // major ticks per frame

	panes [3]pane // focus ring order: program, machine, numbers

	input    textinput.Model
	shellOut []string // recent shell replies, newest last
}

// New builds the model. newMachine must return a freshly wired board with the
// program loaded; the events log is shared with the board's sink.
func New(name string, words []uint32, events *EventLog, newMachine func() (*machine.Motherboard, *cpu.CPU, *ram.RAM)) *Model {
	input := textinput.New()
	input.Prompt = ": "
	input.Placeholder = "help"
	m := &Model{
		name:       name,
		words:      words,
		events:     events,
		newMachine: newMachine,
		speed:      1,
		input:      input,
	}
	m.panes = [3]pane{newProgramPane(), newMachinePane(), newNumbersPane()}

	m.App = tuikit.NewApp()
	m.App.Keys = m.globalKeys()
	m.App.Handle = m.handle
	m.App.Chrome = m.chrome
	m.App.Screens.Register(newBoardScreen(m))

	m.board, m.proc, m.mem = newMachine()
	return m
}

// Init starts the frame loop. Everything else (screens, keymaps) is inert
// wiring; the kit starts nothing by itself.
func (m *Model) Init() tea.Cmd { return frame() }

// Update delegates to the kit's routing contract; host state mutates through
// the Handle closure and the keymap Run funcs, so the returned model is
// always m.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.App.Update(msg)
	return m, cmd
}

// View is the kit's frame assembly: chrome rows and region rows, the modal
// over the region area, or the too-small refusal.
func (m *Model) View() tea.View { return m.App.View() }

// handle is the host's pre-routing hook. It owns the run loop (ticks are not
// keys — they always reach here, even under a modal), the input's width, and
// the shell's keyboard grab: a focused input is a capture the host enforces,
// because typing must swallow keys that would otherwise route (e.g. 'q').
func (m *Model) handle(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case frameMsg:
		if m.running {
			if m.board.Halted {
				m.running = false
				m.printf("stopped: %s", m.stateLine())
			} else {
				m.board.Run(m.speed)
			}
		}
		return frame(), true

	case tea.WindowSizeMsg:
		m.input.SetWidth(min(msg.Width-8, 120))
		return nil, false

	case tea.KeyPressMsg:
		if !m.input.Focused() {
			return nil, false // no grab: let the kit route it
		}
		switch msg.String() {
		case "enter":
			line := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			if line == "" {
				return nil, true
			}
			return m.exec(line), true
		case "esc":
			m.input.Blur()
			return nil, true
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd, true
	}
	return nil, false
}

// exec runs one shell command line. It is the same surface as the keys —
// the acceptance checker will script this surface later.
func (m *Model) exec(line string) tea.Cmd {
	c, err := parseShell(line)
	if err != nil {
		m.printf("%v", err)
		return nil
	}
	switch c.name {
	case "run":
		if m.board.Halted {
			m.printf("machine is stopped: %s (reset to run again)", m.stateLine())
		} else {
			m.running = true
		}
	case "step":
		if m.board.Halted {
			m.printf("machine is stopped: %s (reset to run again)", m.stateLine())
		} else {
			m.board.Run(c.arg)
		}
	case "reset":
		m.events.Reset()
		m.board, m.proc, m.mem = m.newMachine()
		m.running = false
		m.resetPanes()
		m.printf("machine reset: %d words, event log cleared", len(m.words))
	case "speed":
		m.speed = c.arg
		m.printf("speed: %d major ticks per frame", m.speed)
	case "help":
		if !m.Modals.Active() {
			m.Modals.Push(helpModal{m})
		}
	case "quit":
		return tea.Quit
	}
	return nil
}

// Control-state helpers — the Run bodies behind the screen's keymap.

func (m *Model) toggleRun() tea.Cmd {
	if m.board.Halted {
		m.printf("machine is stopped: %s (reset to run again)", m.stateLine())
	} else {
		m.running = !m.running
	}
	return nil
}

func (m *Model) stepOne() tea.Cmd {
	if m.board.Halted {
		m.printf("machine is stopped: %s (reset to run again)", m.stateLine())
	} else {
		m.board.Run(1)
	}
	return nil
}

func (m *Model) faster() tea.Cmd { m.speed = min(m.speed*2, maxSpeed); return nil }

func (m *Model) slower() tea.Cmd { m.speed = max(m.speed/2, 1); return nil }

func (m *Model) focusShell() tea.Cmd { return m.input.Focus() }

// toggleHelp flips the help modal; the binding is global, so it also answers
// while the modal is captured — '?' closes what '?' opened.
func (m *Model) toggleHelp() tea.Cmd {
	if m.Modals.Active() {
		m.Modals.Pop()
	} else {
		m.Modals.Push(helpModal{m})
	}
	return nil
}

// resetPanes fans a machine reset out to the panes' view state (the kit's
// focus ring and modals are not the panes' business and stay put).
func (m *Model) resetPanes() {
	for _, p := range m.panes {
		p.reset()
	}
}

// cursorAddr is the address the panes track: the next fetch while running
// (the pc convention), backing up to the halted instruction once stopped so
// the highlight sits on the word that stopped the machine.
func (m *Model) cursorAddr() uint32 {
	cur := m.proc.PC
	if m.board.Halted && cur >= 4 {
		cur -= 4
	}
	return cur
}

// stateLine is the one-line verdict in the status bar's vocabulary.
func (m *Model) stateLine() string {
	switch m.board.StopReason() {
	case machine.StopHalt:
		return "clean halt"
	case machine.StopFault:
		f := m.board.Fault()
		return fmt.Sprintf("%s (pc=0x%04x addr=0x%04x)", f.Kind, f.PC, f.Addr)
	default:
		return "not started"
	}
}

// runState names the run loop: running, paused, or the stop verdict.
func (m *Model) runState() string {
	if m.board.Halted {
		return m.stateLine()
	}
	if m.running {
		return "running"
	}
	if m.board.Major == 0 {
		return "not started"
	}
	return "paused"
}

// printf appends a shell reply (kept to the last few).
func (m *Model) printf(format string, args ...any) {
	line := strings.TrimSpace(fmt.Sprintf(format, args...))
	if line == "" {
		return
	}
	m.shellOut = append(m.shellOut, line)
	if len(m.shellOut) > 4 {
		m.shellOut = m.shellOut[len(m.shellOut)-4:]
	}
}

// shellCmd is one parsed shell command with its integer argument.
type shellCmd struct {
	name string
	arg  int
}

// parseShell parses the v1 command set: run, step [n], reset, speed n, quit,
// help. Arguments are range-checked — step never runs uncapped batches.
func parseShell(line string) (shellCmd, error) {
	f := strings.Fields(line)
	if len(f) == 0 {
		return shellCmd{}, fmt.Errorf("empty command")
	}
	name := f[0]
	arg := 1
	switch name {
	case "run", "reset", "quit", "help":
		if len(f) > 1 {
			return shellCmd{}, fmt.Errorf("%s takes no argument", name)
		}
	case "step":
		if len(f) > 1 {
			n, err := strconv.Atoi(f[1])
			if err != nil || n < 1 || n > 100000 {
				return shellCmd{}, fmt.Errorf("step: want a count 1..100000, got %q", f[1])
			}
			arg = n
		}
	case "speed":
		if len(f) != 2 {
			return shellCmd{}, fmt.Errorf("speed: want a number 1..%d", maxSpeed)
		}
		n, err := strconv.Atoi(f[1])
		if err != nil || n < 1 || n > maxSpeed {
			return shellCmd{}, fmt.Errorf("speed: want a number 1..%d, got %q", maxSpeed, f[1])
		}
		arg = n
	default:
		return shellCmd{}, fmt.Errorf("unknown command %q — try help", name)
	}
	return shellCmd{name, arg}, nil
}
