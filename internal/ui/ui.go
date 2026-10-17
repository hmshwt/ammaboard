// Package ui is the AmmaBoard terminal UI: a pure view over the live machine
// and its event log (docs/ui.md). Three panes above a command shell; the UI
// never mutates machine state except by running the machine's own stepping —
// speed control changes cycles per frame, never semantics.
package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ammaboard/internal/machine"
	"ammaboard/internal/machine/cpu"
	"ammaboard/internal/machine/ram"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// frameInterval is the UI heartbeat. One frame every 33ms; a running machine
// advances `speed` major ticks per frame.
const frameInterval = time.Second / 30

// maxSpeed bounds cycles-per-frame so a hot loop can still be watched.
const maxSpeed = 1024

// frameMsg drives the run loop: the model re-issues it every frame while the
// program is running.
type frameMsg time.Time

func frame() tea.Cmd {
	return tea.Tick(frameInterval, func(t time.Time) tea.Msg { return frameMsg(t) })
}

// scroller is the scroll state of one pane's scrollable body. follow pins the
// window to what the machine is doing (the pc line, the newest events);
// manual scrolling turns follow off until f is pressed (or the machine resets).
type scroller struct {
	top    int
	follow bool
}

// EventLog is the in-memory tail of the machine's event stream. The batch CLI
// writes every event to SQLite; the TUI only ever needs the recent window.
type EventLog struct {
	cap    int
	events []machine.Event
	total  int
}

// NewEventLog keeps the last cap events (and counts the total).
func NewEventLog(cap int) *EventLog { return &EventLog{cap: cap} }

// Add records one event, trimming the tail to capacity.
func (l *EventLog) Add(e machine.Event) {
	l.events = append(l.events, e)
	if len(l.events) > l.cap {
		l.events = l.events[len(l.events)-l.cap:]
	}
	l.total++
}

// Reset empties the log (on machine reset) without changing capacity.
func (l *EventLog) Reset() { l.events = nil; l.total = 0 }

// Tail returns up to n most recent events, oldest first.
func (l *EventLog) Tail(n int) []machine.Event {
	if n > len(l.events) {
		n = len(l.events)
	}
	out := make([]machine.Event, n)
	copy(out, l.events[len(l.events)-n:])
	return out
}

// Total reports how many events the machine has emitted since (re)start.
func (l *EventLog) Total() int { return l.total }

// Model is the TUI root: machine state plus view state. Machine ownership is
// delegated — newMachine (re)builds a loaded board, so reset is exact.
type Model struct {
	name       string
	words      []uint32
	board      *machine.Motherboard
	proc       *cpu.CPU
	mem        *ram.RAM
	events     *EventLog
	newMachine func() (*machine.Motherboard, *cpu.CPU, *ram.RAM)

	running bool
	speed   int // major ticks per frame
	focus   int // pane: 0 program, 1 machine, 2 numbers
	scroll  [3]scroller

	input    textinput.Model
	shellOut []string // recent shell replies, newest last
	help     bool

	width, height int
}

// New builds the model. newMachine must return a freshly wired board with the
// program loaded; the events log is shared with the board's sink.
func New(name string, words []uint32, events *EventLog, newMachine func() (*machine.Motherboard, *cpu.CPU, *ram.RAM)) Model {
	input := textinput.New()
	input.Prompt = ": "
	input.Placeholder = "help"
	m := Model{
		name:       name,
		words:      words,
		events:     events,
		newMachine: newMachine,
		speed:      1,
		scroll:     [3]scroller{{follow: true}, {follow: true}, {follow: true}},
		input:      input,
	}
	m.board, m.proc, m.mem = newMachine()
	return m
}

// Init starts the frame loop.
func (m Model) Init() tea.Cmd { return frame() }

// Update handles the frame heartbeat, keys, and window sizing. Everything the
// shell can do is also a key; everything a key can do is also a shell command.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.input.SetWidth(min(msg.Width-4, 80))
		return m, nil

	case frameMsg:
		if m.running {
			if m.board.Halted {
				m.running = false
				m.printf("stopped: %s", m.stateLine())
			} else {
				m.board.Run(m.speed)
			}
		}
		return m, frame()

	case tea.KeyPressMsg:
		if m.input.Focused() {
			switch msg.String() {
			case "enter":
				line := strings.TrimSpace(m.input.Value())
				m.input.Reset()
				if line == "" {
					return m, nil
				}
				return m.exec(line)
			case "esc":
				m.input.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focus = (m.focus + 1) % 3
		case "shift+tab":
			m.focus = (m.focus + 2) % 3
		case "space":
			if m.board.Halted {
				m.printf("machine is stopped: %s (reset to run again)", m.stateLine())
			} else {
				m.running = !m.running
			}
		case "s":
			return m.exec("step")
		case "+", "=":
			m.speed = min(m.speed*2, maxSpeed)
		case "-":
			m.speed = max(m.speed/2, 1)
		case "j", "down":
			m.scroll[m.focus].top++
			m.scroll[m.focus].follow = false
		case "k", "up":
			m.scroll[m.focus].top = max(0, m.scroll[m.focus].top-1)
			m.scroll[m.focus].follow = false
		case "g":
			m.scroll[m.focus] = scroller{top: 0}
		case "G":
			// 1<<30 clamps to "bottom" at render time for any body size.
			m.scroll[m.focus] = scroller{top: 1 << 30}
		case "f":
			m.scroll[m.focus].follow = true
		case "?":
			m.help = !m.help
		case "esc":
			// esc unwinds one level: close the help overlay (the shell's esc
			// blurs the input and is handled in the focused branch above).
			if m.help {
				m.help = false
			}
		case ":":
			return m, m.input.Focus()
		}
		return m, nil
	}
	return m, nil
}

// exec runs one shell command line. It is the same surface as the keys —
// the acceptance checker will script this surface later.
func (m Model) exec(line string) (tea.Model, tea.Cmd) {
	c, err := parseShell(line)
	if err != nil {
		m.printf("%v", err)
		return m, nil
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
		m.scroll = [3]scroller{{follow: true}, {follow: true}, {follow: true}}
		m.printf("machine reset: %d words, event log cleared", len(m.words))
	case "speed":
		m.speed = c.arg
		m.printf("speed: %d major ticks per frame", m.speed)
	case "help":
		m.help = true
	case "quit":
		return m, tea.Quit
	}
	return m, nil
}

// stateLine is the one-line verdict in the status bar's vocabulary.
func (m Model) stateLine() string {
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
