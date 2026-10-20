# tuikit — Design Contract

**One-liner:** a small, boring kernel over bubbletea/lipgloss that turns layout, screens, and key routing into declarative data — compose, never wrap.
**Status:** implemented — kernel v0.3.0, 2026-10-20 (receipts + `Emit`, D45);
`internal/tui` migrated onto the kernel 2026-10-20 (kernel contract held: one new
pane was one file + one registration; the pty smoke carried over with two updated
strings). Architecture & API surface: `internal/tuikit/ARCHITECTURE.md`.

## Placement & dependency law
- `internal/tuikit` — the kernel.
- Allowed imports: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, stdlib. Nothing else.
- Forbidden: `bubbles`, every `internal/*`, and `ultraviolet` (bubbletea's renderer internals — never imported).
- `internal/tui` — the app (renamed from `ui`; TUI-specific, a future `webui` shares only `machine` + `EventLog`).
- `cmd/ammaboard/tui.go`: import flip only (`ui.New` → `tui.New`); signatures unchanged.

## Core rules
- **Compose, never wrap.** App root stays a plain `tea.Model`; raw `tea.Msg`/`tea.Cmd` everywhere; the kit routes, never rewrites.
- **Kernel, not framework.** Extracted from real usage, sized ≈ 1,200 lines across a
  dozen files (budget revised 2026-10-18 from the sketch's ≈400 when screens and the
  wiring audit landed), no speculative knobs.
- **Immediate mode.** Every region is a pure per-frame derivation of (machine state + event tail + owned view state). No buffers, no dirty flags, no revision counters.
- **One machine truth.** Board/proc/mem/EventLog are read-only to the UI, never copied into components.
- **Ownership:** machine state — nobody's (derive only) · scroll state — panes · focus ring — screen (one per screen) · key precedence — regions · screen selection — app policy · control state (running/speed) — app. *State lives in components; policy lives in composites.*
- **Purity is the cache.** Determinism makes switch-back correctness free; a buffer is a second copy of truth that can lie.
- **Performance policy.** Ultraviolet already diffs cells; derivation is windowed (visible lines only); no animation — "numbers and grids"; slow updates are acceptable, even desirable for learning. Optimize only on a measured hotspot, at the *data* boundary (windowed accessors, immutability slices), never the component boundary.
- **Caching** exists only at explicit immutability boundaries (blue screen: rendered once at fault time). Change detection, if ever needed, uses data-layer clocks (`board.Major`, `EventLog.Total`) — no kit machinery.

## Primitives (per file)
- **`line.go`** — `Line{Text, Style}` · `Plain` / `Styled` / `Lines` · `Render(width)` (pads to width) · `Truncate` · `Clamp`.
- **`frame.go`** — `Rect`, `Inner()` · `Frame`, `NewFrame(minW, minH)`, `Resize`, `Fits`, `TooSmall` · `Row(Span)` · `(*Row).Cols(Span...) []*Cell` · `Cell.Rect()` (mouse hit-testing free later; events out of scope) · `Span` (unexported): `Fixed(n)`, `Fill()` (multiple fills share remainder).
- **`region.go`** — `Region{Title, Header, Body, Focused, Constraints}` · `Render(w, h, Styles)` pads/clips to **exactly h rows** — equal heights are structural · `Constraints{MinW, MinH}` = the smallest useful cell, declared per region and audited at the screen's declared minimum · `Styles{Border, BorderFocused, Title, Dim}` + `DefaultStyles` (no theming engine; app overrides).
- **`scroll.go`** — `Scroll` (top, follow) · `NewScroll` · `Key(k) bool` · `Window(track, total, height) int` · `Reset` · `Following` · const `FollowPin = 3`.
- **`focus.go`** — `FocusRing` · `Next`/`Prev`/`Index`/`Is`/`Reset`. One ring per screen; focus selects the routing entry point, nothing else.
- **`keymap.go`** — `Binding{Keys, Help, Run}` · `Bind(...)` builder · `Keymap.Route / Hints / Help` · `ScrollKeys(&s)` (scroll keys as generated Keymap) · `Merge(kms...)` first-consumer-wins · `Collisions(kms...)` + `NoShadowing` test helpers.
- **`modal.go`** — `Modal` interface · `ModalStack.Push/Pop/Active` (esc pops one level) · `Center(body, w, h, padX)`.
- **`screen.go`** — `Screen{Name, Frame, FullBleed, Keys, RegionResize, FocusRing, Focusables, Layout func() []Region}` · `ScreenSet.Register/Switch/Current`. Flat registry; no stack; no nested screens.
- **`app.go`** — embeddable `App{Screens, Modals, Keys}` implementing `tea.Model`; host hooks: `Handle(msg) (cmd, handled)`, `Chrome(w, h) []string` (bottom block). Kit routes; tui decides. Wiring audit: `Verify() []Issue` + `AssertCleanWiring` (see below).

**Interface count: 3** (`Span` unexported, `Modal`, region `Keys() Keymap`). Everything else is values and functions.

## Layout contract (D38/D39 as types)
```go
frame := tuikit.NewFrame(141, 28)
frame.Row(tuikit.Fixed(1))                                          // status
panes := frame.Row(tuikit.Fill()).Cols(tuikit.Fixed(44), tuikit.Fill(), tuikit.Fixed(54))
frame.Row(tuikit.Fixed(6))                                          // shell box + keyhints
```
- Min size is **per screen**; too-small → honest refusal notice, never clip or scale.

## Screens
- A screen = derived presentation **plus input grammar** over one shared state (the postgres-view analogy); own layout, own focus/scroll state, own keymap scope.
- Switching is atomic repaint; per-screen view state survives switching; machine `reset` fans out to all screens; switching changes nothing else.
- `FullBleed` screens skip global chrome (fault screen hides the shell, adds its own `r → reset`).
- Transitions are app policy (fault → switch; reset → switch back), never kit mechanism.

## Input — two mechanisms, one chain
```
CAPTURE (preempts; esc releases)      ROUTING (first consumer wins)
1. modal active → modal + esc         3. focused region's Keys()
2. keyboard grab → textinput          4. screen Keys()
                                      5. global Keys()
unconsumed → no-op
```
Geometry is not input: `tea.WindowSizeMsg` bypasses capture, resizes the
frame, and cascades each focusable its **own cell viewport** via
`Screen.RegionResize` — children never see terminal numbers.
- **Keymaps are derived per dispatch** — grammar is derived state, same as pixels; activation via composition (`Merge`), never flags or `When` predicates. Help cannot lie.
- Within a region: deliberate precedence = `Merge` order, written once, visible; across scopes: duplicates are errors. `q`/`ctrl+c`/`esc` reserved globally.
- Hints line = global + current screen (stable per screen); help modal = full derived grammar, grouped global / screen / regions.
- Chain depth stays 3; region composition is containment, not a fourth scope; no sub-focus rings (documented extension point, unbuilt).

## Testing contract
- All headless — keymaps, regions, and layouts are plain values.
- Kit: layout properties (rows sum, equal heights), keymap generation, routing table, modal unwind, `NoShadowing`, `TestScreenSwitchPreservesState`, `AssertCleanWiring`.
- App: existing behavior assertions preserved (plumbing mechanically adapted); the glyph-counting layout test is retired on purpose, replaced by frame properties + one row-count/content integration check.
- Scale at v0.2.0 (2026-10-18): 57 tests (52 kit + 5 testrender), all headless; the
  Go-native pty smoke (`cmd/ammaboard/pty_test.go`, creack/pty) landed 2026-10-20
  ahead of the migration and retires the Python scratch harness. Golden files still
  land with `internal/tui`, where goldens have real frames to look at (see D44).

## Wiring audit (`Verify`)
The assembled wiring is a value graph — keymaps, frames, screens are data — so it can be audited. One audit, three exposures:
- **Tests are the real gate:** kit helper `AssertCleanWiring(t, app)` runs `Verify` headlessly; CI fails before anything ships.
- **`ammaboard tui --check`:** prints the full report (issues + wiring summary: screens, regions, key counts) and exits without launching — fast feedback while authoring a screen.
- **Startup fail-fast:** `Verify` runs before `tea.NewProgram()` (microseconds); errors are fatal with the printed report, warnings go to stderr — catches "edited wiring, skipped tests".

```go
type Issue struct{ Severity Severity; Where string; What string } // Error | Warn
func (a *App) Verify() []Issue               // pure derivation over the wiring graph — no reflection, no registry
func AssertCleanWiring(t *testing.T, a *App) // kit test helper
```

- **Errors (fatal):** reserved keys (`q`/`ctrl+c`/`esc`) shadowed in any scope · duplicate key across scopes · duplicate/empty screen names, no screens, missing `Layout` · focus-ring count ≠ focusable regions · frame unsolvable at declared minimum (`Fixed` sum > `minW`, or no `Fill` row) · `FullBleed` screen with no binding that leaves it.
- **Warnings (printed):** collisions within a region's `Merge` (deliberate precedence is allowed, but must be conscious) · `Binding` with empty `Help` (hints/help are generated) · hints line wider than the screen's `minW` · a screen's min size larger than the exercise screen's (documents the real hardware requirement).
- **Limits (honest):** no runtime terminal fit (the `TooSmall` notice owns that), no semantic-duplicate detection (two keys documented for one action), nothing that requires rendering — a static audit of the declared graph, not a simulator.

## Acceptance criteria (the abstraction's own test)
- New pane → one file + one registration line.
- Keybinding → one keymap entry; hints/help regenerate.
- New screen → one file + one registration.
- Key-claiming component → one file + one `Merge` entry.
- Any change that reopens `Update`'s switch or frame arithmetic → fix the kit first.

## Non-goals (fenced in `tuikit.go`)
Mouse events (rects exported; wheel-scroll is the first future candidate) · theming engine · widget zoo (textinput stays bubbles) · animation/transitions · screen stacks · nested screens · nested focus rings · caching/dirty tracking · state management · ultraviolet · anything machine-aware.

## Rollout
- One atomic change: tuikit + componentized `internal/tui` + adapted tests + `cmd` import flip; full gate (gofmt, vet, all packages, build, pty smoke, `ammaboard tui --check` clean).
- Side quest kept cheap: the dependency law is the extraction seam; a publish decision waits on the kit surviving several real additions without breaking change — no community-driven API compromises until then.

## Implementation notes — deviations from the sketch (recorded 2026-10-18, approved)
- `Screen.RegionKeys(focus int) Keymap`: the host wiring point for the region scope — the
  sketch listed screen `Keys` but not how focused-region keys reach the router.
- `App.Styles` field + `NewApp()`: the palette-override point; a zero-value `App` is not
  a supported shape.
- Chrome rows are multi-line: one string per cell-less row, split on newlines and
  padded/clipped to the row's height (the bottom block is a box, not a line).
- `AssertCleanWiring` takes a kit-local `TestingT` interface — the kernel never imports
  the `testing` package.
- FullBleed-without-own-keys is a Warning, not an Error: global verbs always remain
  reachable at routing level 5, so a hard error would be vacuous.
- Focus cycling (tab/shift+tab) is kit-owned in `App.Update`; the host documents it with
  a display-only global binding so generated help stays truthful.
- The mid-discussion `Screen.RenderRegions` sketch was dropped; region rendering lives in
  `App.render` — one renderer, one place to debug.
