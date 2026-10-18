# tuikit — architecture & API surface

**v0.1.0 · 2026-10-18** · built against bubbletea/v2 v2.0.9 · lipgloss/v2 v2.0.6
Contract: `docs/tuikit.md` (D41–D43). Version bumps with the API surface: minor
for additions, major for breaks. Update this file whenever either changes.

## Shape
≈1,200 lines, 11 files, 3 interfaces (`Span`, `Modal`, `KeySource` via
`Keys() Keymap`). One idea: layout, screens, and keys are declarative data —
assembled once by the host, derived fresh every frame.

## Layers (bottom → top)
`line` → `frame`/`region` → `scroll`/`focus` → `keymap`/`modal` → `screen` →
`app` → `audit`. Lower layers never know the ones above; the app layer is the
only one that assembles.

## Frame of a tick
```
tea.WindowSizeMsg ─→ Frame.Resize
Layout() []Region ─→ focused region marked ─→ region rows render into cell rects
Chrome hook       ─→ fills cell-less rows (padded/clipped to row height)
Modal active      ─→ replaces the region area only; esc pops one level
Frame too small   ─→ refusal notice, never clip or scale
```

## Key routing (the whole contract)
```
capture:  modal active → esc pops, global verbs answer, nothing leaks
          host grab    → Handle returns handled=true (focused input, ticks)
routing:  focused region Keys() → screen Keys() → global Keys(); first wins
focus:    tab / shift+tab cycle the current screen's ring (kit-owned)
```

## API by file
- **line.go** — `Line{Text, Style}`; `Plain` `Styled` `Lines`; `Line.Render(w)`
  (pads so backgrounds span); `Truncate`; `Clamp`.
- **frame.go** — `Rect{X,Y,W,H}.Inner`; `Span`: `Fixed(n)` `Fill()`; `Frame`:
  `NewFrame(minW,minH)` `Resize` `Fits` `TooSmall` `Row(span)`; `Row`:
  `Cols(...Span) []*Cell` `Rect`; `Cell.Rect`.
- **region.go** — `Region{Title,Header,Body,Focused}.Render(w,h,Styles)` — pads
  or clips to exactly h rows; `Styles` + `DefaultStyles`.
- **scroll.go** — `Scroll`: `NewScroll` `Key(j,k,up,down,g,G,f)` `Window(track,
  total,h)` `Reset` `Following`; const `FollowPin = 3`.
- **focus.go** — `FocusRing`: `NewFocusRing(n)` `Next` `Prev` `Index` `Is`
  `Reset`. One per screen; owned by the screen.
- **keymap.go** — `Binding{Keys,Help,Run}`; `Bind(...).Help(...).Run(...)`;
  `Keymap.Route/Hints/Help/Keys`; `Merge` (earlier wins); `Collisions`;
  `NoShadowing`; `ScrollKeys`; `ReservedKeys = q, ctrl+c, esc`.
- **modal.go** — `Modal.View(w,h)`; `ModalStack.Push/Pop/Active/Top`; `Center`.
- **screen.go** — `Screen{Name, Frame, FullBleed, Keys, Layout, RegionKeys,
  FocusRing, Focusables}`; `NewScreen(name,minW,minH,focusables)`;
  `ScreenSet.Register/Switch/Current/All` (flat registry, no stack).
- **app.go** — `App{Screens, Modals, Keys, Styles, Handle, Chrome}`; `NewApp`;
  `Init/Update/View` (implements `tea.Model`; embeddable).
- **audit.go** — `Issue`, `Severity`; `App.Verify() []Issue`;
  `AssertCleanWiring(TestingT, App)`.

## Host wiring recipe (the acceptance test)
1. Component file: a pure view over machine state → `[]Line`/`Region`; owns its
   `Scroll`. No machine mutation, ever.
2. Screen: `NewScreen(...)`, declare frame rows (`Fixed` chrome, `Fill` regions,
   `Cols` widths), `Keys` (screen scope), `RegionKeys` (region scope, composed
   with `Merge`), `Layout` closure.
3. App: `Screens.Register`, global `Keymap`, `Chrome` hook, `Handle` for machine
   messages and input grabs.
4. `AssertCleanWiring` in tests; `Verify` again at startup; fix the kit first if
   a change reopens routing or frame arithmetic.

## Invariants worth defending
- Regions are values built per frame; the kit stores no rendered output
  (purity is the cache).
- State lives in components (scrolls), policy in composites (rings, scopes,
  screens); machine state belongs to no one.
- Only the global scope may claim `q`/`ctrl+c`/`esc`; cross-scope duplicates are
  errors, within-scope duplicates are warnings.
- Hidden screens cost nothing; switching screens changes no state.
- The kernel imports only bubbletea, lipgloss, and stdlib — never `bubbles`,
  never `ultraviolet`, never `internal/*`.
