# tuikit — architecture & API surface

**v0.2.0 · 2026-10-18** · built against bubbletea/v2 v2.0.9 · lipgloss/v2 v2.0.6
Contract: `docs/tuikit.md` (D41–D43). Version bumps with the API surface: minor
for additions, major for breaks. Update this file whenever either changes.

## Shape
≈1,200 lines, 11 files, 3 interfaces (`Span`, `Modal`, `KeySource` via
`Keys() Keymap`). One idea: layout, screens, and keys are declarative data —
assembled once by the host, derived fresh every frame.

## Layers (bottom → top)
`line` → `frame`/`region` → `scroll`/`focus` → `keymap`/`modal` → `screen` →
`app` → `audit`. Lower layers never know the ones above; the app layer is the
only one that assembles. Test-support beside the kernel: `testrender`
(stdlib-only decode → grid → golden).

## Frame of a tick
```
tea.WindowSizeMsg ─→ Frame.Resize ─→ RegionResize(i, cellW, cellH) per focusable
                      (geometry is not input: it bypasses capture, always reaches Handle)
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
- **region.go** — `Region{Title,Header,Body,Focused,Constraints}.Render(w,h,
  Styles)` — pads or clips to exactly h rows; `Constraints{MinW,MinH}` = the
  smallest useful cell, audited at the screen's declared minimum; `Styles` +
  `DefaultStyles`.
- **scroll.go** — `Scroll`: `NewScroll` `Key(j,k,up,down,g,G,f)` `Window(track,
  total,h)` `Reset` `Following`; const `FollowPin = 3`.
- **focus.go** — `FocusRing`: `NewFocusRing(n)` `Next` `Prev` `Index` `Is`
  `Reset`. One per screen; owned by the screen.
- **keymap.go** — `Binding{Keys,Help,Run}`; `Bind(...).Help(...).Run(...)`;
  `Keymap.Route/Hints/Help/Keys`; `Merge` (earlier wins); `Collisions`;
  `NoShadowing`; `ScrollKeys`; `ReservedKeys = q, ctrl+c, esc`.
- **modal.go** — `Modal.View(w,h)`; `ModalStack.Push/Pop/Active/Top`; `Center`.
- **screen.go** — `Screen{Name, Frame, FullBleed, Keys, Layout, RegionKeys,
  RegionResize, FocusRing, Focusables}`; `NewScreen(name,minW,minH,
  focusables)`; `ScreenSet.Register/Switch/Current/All` (flat registry, no
  stack).
- **app.go** — `App{Screens, Modals, Keys, Styles, Handle, Chrome}`; `NewApp`;
  `Init/Update/View` (implements `tea.Model`; embeddable).
- **audit.go** — `Issue`, `Severity`; `App.Verify() []Issue`;
  `AssertCleanWiring(TestingT, App)`.
- **testrender/** — stdlib-only visual-testing support: `StripANSI`;
  `Decode(s, Palette) (Grid, error)` into `Cell{R, A}` grids (level 2 = the
  VGA attr+char pair; `Attr` is the kitty-era slot); `GoldenChars`/
  `GoldenGrid` with `-update`. The need-to-develop fence lives in its package
  doc.

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
- Geometry is not input: resize bypasses capture; keys are the only thing
  capture ever swallows.
- The kernel imports only bubbletea, lipgloss, and stdlib — never `bubbles`,
  never `ultraviolet`, never `internal/*`.


# Report from the Last agent on finishing the tuikit task

Done — tuikit **v0.2.0** implemented and green. All 12 packages pass, vet/gofmt clean, binary builds, pty smoke passes, and the diff is contained to `internal/tuikit/` + the two docs.

## A. Kernel additions

- **`Constraints{MinW, MinH}`** on `Region` — a component's declared smallest useful cell. `Verify` resolves the frame **at the screen's declared minimum** (where fill-splitting is deterministic, via the new `cellRectsAt`) and errors when a promise can't be met there: *"at the declared minimum the cell is 42×9, the region declares ≥60×0"*. Per-pane geometry is now data — the geometry twin of the keymap audit.
- **The resize cascade** — `Screen.RegionResize(focus, w, h)`: on `WindowSizeMsg` the frame resizes and every focusable receives **its own cell viewport**, never terminal numbers. Geometry is not input: resize happens *before* capture and always reaches `Handle`.
- **Two real bugs fixed en route**, both regression-tested: the old code swallowed `WindowSizeMsg` entirely under an open modal (host never resized), and it also swallowed *all* non-key messages — meaning the machine would have frozen while help was open. Now keys are the only thing capture ever swallows; ticks and geometry always flow.

## B. `internal/tuikit/testrender` (stdlib-only)

- **`grid.go`** — `Cell{Rune, Attr}` (the VGA attr+char pair; `Attr` is the kitty-era slot), `Grid.Chars()` (level 1), `Grid.String()` (level 2: attr lines over char lines).
- **`decode.go`** — `StripANSI`; `Decode(s, Palette)` walks the SGR vocabulary the kit actually emits (fg/bg truecolor, bold, reset), maps styles through a host-registered `Palette`, and **fails loudly on unregistered colors** — no silent `?`.
- **`golden.go`** — `Golden`/`GoldenChars`/`GoldenGrid` with `-update` and a first-mismatch diff; `TB` interface so the package never imports `testing`. The need-to-develop fence is the package doc's second paragraph.

The proof test: `TestRegionRenderDecodesToGrid` renders two kit regions through real lipgloss, decodes the truecolor output, and asserts the attr grid — `D` on the dim border, `A` on the focused border, `T` on bold titles, `N` on bodies.

## Tests & docs

57 tests total (52 kit + 5 testrender), including the new contracts: `TestResizeCascade` (regions see `8×9`/`72×9`, not `80×12`), `TestResizeBypassesModalCapture`, `TestTicksFlowUnderModal`, `TestVerifyConstraints`, golden update/compare round-trip, decoder edge cases.

Docs: `ARCHITECTURE.md` → v0.2.0 (new layers line, resize cascade in the tick flow, testrender API entry, "geometry is not input" invariant), `docs/tuikit.md` (status, primitives, geometry note, test scale), and **D44** in `decisions.md` recording all three additions plus the fence — the list of what we refused, so future-us behaves.

Still queued for the migration pass, per D44: the five golden states + the `creack/pty` Go smoke that retires the Python harness.
