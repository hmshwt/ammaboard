package tuikit

import (
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// App is the embeddable wiring root: it owns the screen registry, the modal
// stack, and the global keymap, and implements tea.Model so the host stays a
// plain bubbletea app. The host keeps everything machine-specific in Handle;
// the kit routes, the host decides.
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

	w, h int
}

// NewApp wires an empty app with the default palette.
func NewApp() *App {
	return &App{Screens: NewScreenSet(), Styles: DefaultStyles}
}

// Init starts nothing by itself; the host batches its own heartbeat with it.
func (a *App) Init() tea.Cmd { return nil }

// Update is the routing contract (docs/tuikit.md):
//
//	geometry: tea.WindowSizeMsg resizes the frame and cascades each region's
//	          own cell viewport (Screen.RegionResize). Geometry is not input —
//	          it bypasses capture, and non-key messages always reach Handle
//	          (the machine keeps stepping under the help overlay)
//	capture:  keys only — an open modal (esc pops one level; global verbs
//	          answer; nothing leaks) and host grabs (Handle, handled=true)
//	routing:  focused region Keys() → screen Keys() → global Keys(); first
//	          consumer wins
//
// tab/shift+tab cycle the current screen's focus ring — the kit's reflex;
// the host documents it with a display-only binding so help stays truthful.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		a.w, a.h = ws.Width, ws.Height
		if scr := a.current(); scr != nil {
			scr.Frame.Resize(a.w, a.h)
			a.resizeRegions(scr)
		}
	}
	key, isKey := msg.(tea.KeyPressMsg)
	if isKey && a.Modals.Active() {
		if key.String() == "esc" {
			a.Modals.Pop()
			return a, nil
		}
		if cmd, done := a.Keys.Route(key.String()); done {
			return a, cmd
		}
		return a, nil // captured: keys do not leak to the host or the regions
	}
	if a.Handle != nil {
		if cmd, done := a.Handle(msg); done {
			return a, cmd
		}
	}
	if !isKey {
		return a, nil
	}
	k := key.String()
	if scr := a.current(); scr != nil {
		switch {
		case k == "shift+tab":
			scr.FocusRing.Prev()
			return a, nil
		case k == "tab" || key.Code == tea.KeyTab:
			scr.FocusRing.Next()
			return a, nil
		}
		if scr.RegionKeys != nil {
			if cmd, done := scr.RegionKeys(scr.FocusRing.Index()).Route(k); done {
				return a, cmd
			}
		}
		if cmd, done := scr.Keys.Route(k); done {
			return a, cmd
		}
	}
	if cmd, done := a.Keys.Route(k); done {
		return a, cmd
	}
	return a, nil
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
