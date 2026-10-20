package tuikit

import (
	"fmt"
	"strings"
	"sync"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// App is the embeddable wiring root: it owns the screen registry, the modal
// stack, the global keymap, the receipt ring, and the emission queue, and
// implements tea.Model so the host stays a plain bubbletea app. The host
// keeps everything machine-specific in Handle; the kit routes, never rewrites.
type App struct {
	Screens *ScreenSet
	Modals  ModalStack
	Keys    Keymap // global scope — the only scope allowed to claim ReservedKeys
	Styles  Styles // region furniture palette; DefaultStyles unless overridden

	// Handle sees every message before routing; returning handled stops the
	// chain. Machine ticks live here, and so does the shell's keyboard grab:
	// a focused input is a capture the host enforces (it must swallow typing
	// that would otherwise route, e.g. 'q').
	Handle func(msg tea.Msg) (tea.Cmd, bool)

	// Chrome renders the content of the frame's cell-less rows (status bar,
	// bottom block), in frame order — one string per row, split on newlines
	// and padded or clipped to the row's height.
	Chrome func(w, h int) []string

	// Trace is the receipt ring (see trace.go): every dispatch records what
	// was consulted and what answered. nil silences it; NewApp enables it.
	Trace *Trace

	w, h int

	// The upward channel (see emit.go): facts queued by components, drained
	// and dispatched at the next Update boundary.
	mu      sync.Mutex
	pending []emission
}

// NewApp wires an empty app with the default palette and the receipt ring on.
func NewApp() *App {
	return &App{Screens: NewScreenSet(), Styles: DefaultStyles, Trace: newTrace(512)}
}

// Init starts nothing by itself; the host batches its own heartbeat with it.
func (a *App) Init() tea.Cmd { return nil }

// Update is the routing contract (docs/tuikit.md):
//
//	drain:    queued facts (Emit) dispatch first — FIFO, provenance stamped,
//	          at a dispatch boundary; never inline, never re-entrant
//	geometry: tea.WindowSizeMsg resizes the frame and cascades each region's
//	          own cell viewport (Screen.RegionResize). Geometry is not input —
//	          it bypasses capture, and non-key messages always reach Handle
//	          (the machine keeps stepping under the help overlay)
//	capture:  keys only — an open modal (esc pops one level; global verbs
//	          answer; nothing leaks) and host grabs (Handle, handled=true)
//	routing:  focused region Keys() → screen Keys() → global Keys(); first
//	          consumer wins
//	receipts: every dispatch leaves a TraceEvent — the ladder walk, the
//	          capture verdict, the provenance of any command it produced
//
// tab/shift+tab cycle the current screen's focus ring — the kit's reflex;
// the host documents it with a display-only binding so help stays truthful.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := a.dispatchPending()
	if cmd := a.dispatch(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return a, batchCmds(cmds)
}

// dispatchPending drains the emission queue and dispatches each queued fact
// before the current message.
func (a *App) dispatchPending() []tea.Cmd {
	a.mu.Lock()
	pending := a.pending
	a.pending = nil
	a.mu.Unlock()
	if len(pending) == 0 {
		return nil
	}
	var cmds []tea.Cmd
	for _, em := range pending {
		if c := a.dispatch(em); c != nil {
			cmds = append(cmds, c)
		}
	}
	return cmds
}

// batchCmds joins commands, dropping nils.
func batchCmds(cmds []tea.Cmd) tea.Cmd {
	var out []tea.Cmd
	for _, c := range cmds {
		if c != nil {
			out = append(out, c)
		}
	}
	switch len(out) {
	case 0:
		return nil
	case 1:
		return out[0]
	default:
		return tea.Batch(out...)
	}
}

// dispatch routes one message through the contract, leaving a receipt:
// geometry cascades (not input), capture preempts (modal, grab), Handle sees
// everything, the ladder answers keys first-wins. from carries provenance
// ("emit:<label>" for queued facts, "runtime" otherwise).
func (a *App) dispatch(msg tea.Msg) tea.Cmd {
	from := "runtime"
	if em, ok := msg.(emission); ok {
		from, msg = "emit:"+em.from, em.msg
	}

	key, isKey := msg.(tea.KeyPressMsg)

	note := ""
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		a.w, a.h = ws.Width, ws.Height
		if scr := a.current(); scr != nil {
			scr.Frame.Resize(a.w, a.h)
			a.resizeRegions(scr)
		}
		note = fmt.Sprintf("resize %d×%d → cascade", ws.Width, ws.Height)
	}

	if isKey && a.Modals.Active() {
		k := key.String()
		if k == "esc" {
			a.Modals.Pop()
			a.recordKey(k, "modal", "captured: esc popped one level")
			return nil
		}
		if b := a.Keys.find(k); b != nil {
			label := claimLabel("global", b)
			a.recordKey(k, label, "captured: modal · global verb answered")
			return a.wrapCmd(label, runBinding(b))
		}
		a.recordKey(k, "modal", "captured: nothing leaked")
		return nil
	}

	if a.Handle != nil {
		if cmd, done := a.Handle(msg); done {
			if isKey {
				a.recordKey(key.String(), "Handle", "captured: grab")
			} else {
				a.recordMsg(from, dispatchWhat(note, "consumed by Handle"))
			}
			return a.wrapCmd("Handle", cmd)
		}
	}
	if !isKey {
		a.recordMsg(from, dispatchWhat(note, "unconsumed"))
		return nil
	}
	k := key.String()
	var walk []string

	if scr := a.current(); scr != nil {
		switch {
		case k == "shift+tab":
			scr.FocusRing.Prev()
			a.recordKey(k, "kit", "shift+tab → focus ring moved")
			return nil
		case k == "tab" || key.Code == tea.KeyTab:
			scr.FocusRing.Next()
			a.recordKey(k, "kit", "tab → focus ring moved")
			return nil
		}
		if scr.RegionKeys != nil {
			focus := scr.FocusRing.Index()
			if b := scr.RegionKeys(focus).find(k); b != nil {
				label := claimLabel(fmt.Sprintf("region %d", focus), b)
				walk = append(walk, fmt.Sprintf("region %d:HIT %q", focus, strings.Join(b.Keys, "/")))
				a.recordKey(k, label, strings.Join(walk, " · "))
				return a.wrapCmd(label, runBinding(b))
			}
			walk = append(walk, fmt.Sprintf("region %d:miss", focus))
		}
		if b := scr.Keys.find(k); b != nil {
			label := claimLabel("screen", b)
			walk = append(walk, fmt.Sprintf("screen:HIT %q", strings.Join(b.Keys, "/")))
			a.recordKey(k, label, strings.Join(walk, " · "))
			return a.wrapCmd(label, runBinding(b))
		}
		walk = append(walk, "screen:miss")
	}
	if b := a.Keys.find(k); b != nil {
		label := claimLabel("global", b)
		walk = append(walk, fmt.Sprintf("global:HIT %q", strings.Join(b.Keys, "/")))
		a.recordKey(k, label, strings.Join(walk, " · "))
		return a.wrapCmd(label, runBinding(b))
	}
	walk = append(walk, "global:miss")
	a.recordKey(k, "", strings.Join(walk, " · ")+" → unconsumed")
	return nil
}

// dispatchWhat joins a dispatch's notes into one receipt line.
func dispatchWhat(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}

// find returns the first binding answering key, or nil.
func (km Keymap) find(key string) *Binding {
	for i := range km {
		for _, k := range km[i].Keys {
			if k == key {
				return &km[i]
			}
		}
	}
	return nil
}

// claimLabel names a winning claim for receipts: scope + keys.
func claimLabel(scope string, b *Binding) string {
	return fmt.Sprintf("%s %s", scope, strings.Join(b.Keys, "/"))
}

// runBinding executes a winning binding: Run nil means consumed, no command.
func runBinding(b *Binding) tea.Cmd {
	if b.Run == nil {
		return nil
	}
	return b.Run()
}

// wrapCmd returns cmd with a receipt wrapper: when it runs, the message it
// produces is stamped with the producer's label.
func (a *App) wrapCmd(from string, cmd tea.Cmd) tea.Cmd {
	if cmd == nil || a.Trace == nil {
		return cmd
	}
	return func() tea.Msg {
		msg := cmd()
		if msg != nil {
			a.record(TraceEvent{Kind: TraceCmd, From: from, What: fmt.Sprintf("produced %T", msg)})
		}
		return msg
	}
}

// resizeRegions runs the cascade's leaf: every focusable region learns its
// own cell viewport at the new size — the screen translates window
// coordinates into region coordinates so children never see terminal numbers.
func (a *App) resizeRegions(scr *Screen) {
	if scr.RegionResize == nil {
		return
	}
	cells := scr.Frame.cellRectsAt(a.w, a.h)
	for i := 0; i < scr.Focusables && i < len(cells); i++ {
		scr.RegionResize(i, cells[i].W, cells[i].H)
	}
}

// View assembles the frame: chrome rows and region rows in frame order, the
// modal over the region area, or the too-small refusal.
func (a *App) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	return v
}

func (a *App) render() string {
	scr := a.current()
	if scr == nil || a.Chrome == nil {
		return ""
	}
	if !scr.Frame.Fits() {
		return scr.Frame.TooSmall()
	}
	regions := scr.Layout()
	if i := scr.FocusRing.Index(); i >= 0 && i < len(regions) {
		regions[i].Focused = true // the accent border follows the ring
	}

	// The modal replaces the region area (rows with cells); chrome stays.
	var modal Modal
	if a.Modals.Active() {
		modal, _ = a.Modals.Top()
	}
	regionH := 0
	for _, row := range scr.Frame.rows {
		if len(row.cells) > 0 {
			regionH += row.Rect().H
		}
	}

	chrome := a.Chrome(a.w, a.h)
	chromeIdx := 0
	nextChrome := func(width, height int) string {
		var content string
		if chromeIdx < len(chrome) {
			content = chrome[chromeIdx]
			chromeIdx++
		}
		if height <= 0 {
			return ""
		}
		lines := strings.Split(content, "\n")
		for len(lines) < height {
			lines = append(lines, "")
		}
		lines = lines[:height]
		for i := range lines {
			lines[i] = padRight(lines[i], width)
		}
		return strings.Join(lines, "\n")
	}

	var pieces []string
	region := 0
	modalPlaced := false
	for _, row := range scr.Frame.rows {
		r := row.Rect()
		if len(row.cells) == 0 {
			pieces = append(pieces, nextChrome(r.W, r.H))
			continue
		}
		switch {
		case modal != nil && !modalPlaced:
			pieces = append(pieces, modal.View(a.w, regionH))
			modalPlaced = true
		case modal != nil:
			pieces = append(pieces, "")
		default:
			cells := make([]string, 0, len(row.cells))
			for _, cell := range row.cells {
				cr := cell.Rect()
				rr := Region{}
				if region < len(regions) {
					rr = regions[region]
				}
				cells = append(cells, rr.Render(cr.W, cr.H, a.Styles))
				region++
			}
			pieces = append(pieces, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, pieces...)
}

func (a *App) current() *Screen {
	if a.Screens == nil {
		return nil
	}
	return a.Screens.Current()
}
