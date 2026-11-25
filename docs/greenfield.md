# greenfield.md — the from-scratch spec for the kit

> Status: **frozen draft** (pre-tag). This document is the reference the
> extraction ships against. It was produced by a clean-slate exercise:
> "picture a team inside charmland building this layer — what would we offer
> and how would the API look?" — then diffed against what `internal/tuikit`
> grew into. The verdict: the architecture converges; five amendments and a
> vocabulary correction survive the diff. Everything below is the merged
> result.
>
> Working name: **kit** (placeholder — "tuikit" stands pending the Charm
> email). Package paths in examples are illustrative.

---

## 1. The pitch

> Bubbletea gives you the *loop*. This gives you the *shell*: places, verbs,
> and the contracts between them — so that routing, documentation, and
> verification are all projections of one declarative table instead of three
> hand-maintained lies.

**We own:** places (screens/panes), verbs (bindings/actions), capture
(modals), generated docs (help/palette/footer), verification (audit,
receipts), environment facts (size, focus).

**We never own:** your domain data, styling (lipgloss), the render loop
(bubbletea), async work (`tea.Cmd` passes through untouched), mouse, your
state management. The app's data store is the app's own Go struct; the kit
never gains a type for it.

Audience: backend engineers building operational TUIs who think in screens.
Trust anchor: the escape hatch — every layer can be bypassed, and raw
bubbletea is always one delegation away.

## 2. Non-negotiables

1. **Library, not framework.** No global state, no mandatory ownership of
   `main`. Adoption ladder (below) where every rung is independently valuable.
2. **One direction of trust.** The layer consumes `tea.Msg`/`tea.Cmd`, never
   hides them. Messages it doesn't understand fall through to your `Update`
   untouched — and that pass-through is itself an audited invariant, not a
   promise.
3. **Everything is data; the rest is derivation.** Bindings are data → help,
   palette, footer, audit, and README key tables are all generated views of
   the same table. If any two of them can disagree, the design is wrong.
4. **Facts have one owner.** There is exactly one terminal, so exactly one
   size. It lives at the root (`App`). Screens hold *promises*
   (`Min`), never facts. Nothing below the root stores a `w`. (This is the
   0×0-resize lesson, promoted to principle: the bug class is made
   unrepresentable, not caught.)
5. **Determinism first.** Same input → same frames. Receipts carry sequence
   numbers, not timestamps. Tests run headless; golden frames are cheap.
6. **A layer is judged by how few types it needs.** Public-type budget ≈ 12.

## 3. Vocabulary

### The five nouns

| Concept | Is | Carries | Never carries |
|---|---|---|---|
| **Screen** | a named place | `Min` promise, a layout expression, keys | children screens, its own size |
| **Layout** (`Row`/`Cols`) | anonymous split tree | constraints (`Fill`/`Fixed`/`Ratio`), optional `Title` | keys, focus, cursors |
| **Pane** | leaf | `Render(Slot)`, optional `Keys()`, optional `Notify(Slot)` | children panes |
| **Action** | a named verb | binding, help, danger flag | geometry |
| **Modal** | capture | view, one esc level | routing scope, slots, focus ring |

Screens switch, panes tile, cursors point, details stack, modals capture.

### The nesting rule

> **Structure recurses; behavior doesn't.**

Any layout node can carry a title (chrome). Only leaves carry keys, cursors,
focus, and scroll. This makes every hard question boring by construction:

- focus ring = flat list of leaves; containers are never in it
- routing = three scopes, forever: leaf → screen → global (no path precedence)
- capture/esc = stack depth, independent of tree depth
- audit walks leaves for behavior, any node for claims

The README litmus:

> If a box has a cursor, it's a pane. If a box has a title, it can be
> anything. If a box has keys, it must be a pane.

Chrome curtails to exactly one thing: **`Title`** — and that is not
decoration, it is *data* (help names scopes by it). Everything below the
title border is the pane's business.

### Addressing structure: extrapolated IDs

> **Name what has behavior (screens, panes, actions); address what
> doesn't (structure).**

Layout groups (`Row`/`Cols`) never need names, but may take an optional
label. The kit extrapolates it into a canonical ID by appending the group's
positional path:

    kit.Cols(...).ID("main")  →  main__C1xR0
    kit.Cols(...)             →  __C1xR0

(The exact suffix encoding is an implementation detail; the rule is: label,
if any, then `__`, then the position-derived token.)

Because the suffix comes from the group's position in the tree, IDs are
**unique by construction** — the audit performs no uniqueness checks and
collision is impossible. Labels make deep trees readable in diagnostics;
the bare positional token keeps every group addressable even when
unlabeled. If the tree is edited and positions shift, IDs shift with them —
correct behavior, since diagnostics-only identifiers *should* track the
structure they describe.

IDs are **diagnostics only**: audit violations, the tree dump, and receipts
use them. They are never routing scopes, never appear in help or palette,
and can never be key- or focus-targets — that fence keeps
behavior-on-containers from sneaking back in through the diagnostics door.

Slots are never named at all: each slot has exactly one pane, so **the
pane's name is the slot's address**. Two names for one rectangle would be
two namespaces that can drift.

### Items carry identity; panes carry behavior

Panes contain **items** — rows of content with stable IDs. The cursor anchors
to an item ID and the index is derived against the live list each frame
(killing the last row can never panic or drift). Items never register keys:
if keys lived on items, the routing table would become data-dependent and all
three projections would collapse — help couldn't enumerate them, the audit
couldn't check collisions. The rule: **the pane declares keys statically;
every handler reads the cursor's item.** Different item types needing
different verbs dispatch by type *inside* the handler (lazygit's
context-bindings model).

### The slots pipeline

```
App (facts: size) ──► Screen (promise: Min) ──► Layout (split tree) ──► Slot ──► Pane draws it
```

```go
type Slot struct {
    W, H    int
    Focused bool
}
```

The layout deals slots; panes draw their slot; the kit draws chrome. One
struct, three fields — the entire per-frame contract is "here's your slot;
draw it."

## 4. Verbs and stacks

### Reserved universal verbs

Exactly three, reserved at the root, app-rebindable only at its peril:

- `esc` — unwind (pops the innermost pushable, then blurs focus)
- `?` — help (generated; cannot lie)
- `:` — palette (generated from the table)

Nothing else is reserved. Digits are the app's — the default wiring is
**1–9 = switch screens** in registration order (digits go to the coarsest
navigation: screens, not slots; slot-jumping is tab's job). `ctrl+c` quits by
default via a rebindable quit action.

### Capture order

1. **Modal stack** — the top modal captures all keys. Universal verbs still
   answer during capture (esc pops one level; nothing leaks).
2. **Focused pane keys** → **screen keys** → **global verbs** — first
   consumer wins.

Non-key messages bypass capture entirely (the machine keeps stepping under
any overlay). Geometry bypasses capture.

### Two pushable stacks

- **Screen stack** — drill-down places; they *replace* the viewport and
  capture nothing extra; esc pops.
- **Modal stack** — capturing overlays; esc pops; the screen stays rendered
  underneath.

**Confirm is a derived modal** — Danger actions compile to a confirm push;
confirm flows are never hand-wired. The palette grabs input (type-to-filter)
while open; before it's open, `:` is just a global verb.

**No depth limits are enforced on either stack** — drill-down and overlay
chains may grow arbitrarily, and the kit adds no cap. Stacking depth is a
runtime pattern (the audit only sees static wiring), and deep chains are a
design smell, not a wiring error — the receipts and debug palette will show
the stack growing, which is the right nudge. Revisit limits in a later
version if real apps demonstrate a need.

## 5. One table, many projections

The routing table is a static, enumerable data structure. Everything below is
a pure function of it (plus, for footer/debug, the live receipt stream):

| Projection | What it is | Why it can't lie |
|---|---|---|
| **Routing** | the dispatch itself, with a `Receipt` per dispatch | the table *is* the router |
| **Dry-run** | `Route(key, scope)` → Receipt without executing | "what happens if I press X here?" is assertable without mutation |
| **Help** (`?`) | generated columns per scope | derived from bindings |
| **Palette** (`:`) | type-to-filter command list, keys shown inline | derived from bindings |
| **Footer** | current scope's verbs (focused pane + globals) | derived — user-maintained footer text is the drift the kit exists to kill |
| **Audit** | `Validate()` → `[]Violation` (collisions, dead keys, lies, danger-without-confirm, geometry) | walks the table |
| **Receipts** | per-dispatch trace records | recorded, bounded, sequence-numbered |
| **Debug view** | the receipt stream, human-visible (§8) | same stream tests assert on |

## 6. Geometry

### Constraints

`Fill` / `Fixed(n)` / `Ratio(a, b)` — relative, deterministic arithmetic.
Runtime distribution: `Fixed`/`Ratio` carve, `Fill` takes the remainder. No
shrink negotiation, no overflow resolution, ever. If the actual terminal is
smaller than a screen's promise, the TooSmall gate fires — that is the *only*
runtime size gate.

### Minimums: promises, checked at audit time

The **screen** declares `Min(w, h)`. Panes may declare optional minimums —
audit-time truthfulness only, never runtime layout inputs. The check is a
direction-aware fold over the tree:

```
required(Row)    = { width: max(children.w),  height: sum(children.h) }
required(Cols)   = { width: sum(children.w),  height: max(children.h) }
required(pane)   = { width: pane.minW,        height: pane.minH }   // optional
required(screen) = required(rootLayout) ≤ screen.Min   → else violation
```

- `Fixed(42)` implies a minimum of 42 — the fold takes
  `max(declared min, implied by Fixed)`, so devs never double-declare.
- Chrome costs (a title costs its rows) are kit-known constants, so the fold
  is exact.
- The audit reports the **derived honest minimum** per screen:
  "your screen's honest minimum is 108×22; you promised 80×20."
  Screen `Min` stops being a hand-tuned guess and becomes a checked claim.

## 7. The API surface

### Day one — one screen, two panes, three verbs

```go
app := kit.New(
    kit.Screen("jobs", kit.Min(100, 20),
        kit.Row(
            kit.Pane("queue", &QueuePane{jobs: store}).Fill(),
            kit.Cols(
                kit.Pane("detail", &DetailPane{}).Fixed(42),
                kit.Pane("log", &LogPane{}).Fill(),
            ),
        ),
    ),
    kit.Action("refresh", "r", doRefresh).Help("reload from the API"),
    kit.Action("kill", "K", doKill).Help("kill job").Danger(),
)
kit.Run(app) // constructs tea.Program, wires Init/Update/View, validates, runs
```

You compose `Row`/`Cols`; the tree is anonymous. Chrome on any node:

```go
kit.Cols(
    kit.Pane("staged", s),
    kit.Pane("unstaged", u),
).Title("Staging") // the lazygit-staging pattern: titled split, two leaves
```

Groups may take an optional label — `kit.Row(...).ID("main")` — which the
kit extrapolates into a canonical, position-derived ID (`main__C1xR0`),
unique by construction, used in violations, dumps, and receipts.

### Panes

```go
type Pane interface {
    Render(s kit.Slot) string
}

// optional:
type Keyer interface{ Keys() kit.Bindings }        // static declarations only
type Notifier interface{ Notify(s kit.Slot) }      // stateful children only (courtesy, not truth)
```

`Notify` fires when the pane's screen is current during a resize and when the
screen becomes current (children may have missed resizes in the background).
Pure panes need no events — they derive from their slot every frame.

### Cursors

```go
type Cursor struct{ /* anchored to item IDs */ }

func (c *Cursor) Resolve(liveIDs []string) int // nearest live row; never panics
```

### Projections

```go
app.Route("K", kit.On("jobs")) // → Receipt{Action: kill, Scope: "jobs", Confirm: true}
app.Help(kit.Global)           // → generated columns
app.Validate()                 // → []Violation — run in dev startup AND CI
app.Debugf("queue drained: %d jobs", n) // app events join the same timeline
```

### Type budget

`App, Screen, Layout, Slot, Pane, Bindings, Key, Action, Cursor, Modal,
Receipt, Violation` — twelve, and it should stay near twelve.

## 8. Debug mode (dev-only, `~` by default)

Browser-devtools-lite: a quick, big, live event log for the moment you hit
something unexpected.

- **Toggle**: one global binding, default `~` (Quake-console muscle memory).
  Not reserved — a rebindable default verb.
- **Shape: a big palette-style overlay, not a slot-filling pane.** Default
  size `screen.w-5 × screen.h-5`, centered. It captures keys while open (it
  is palette-shaped); **the receipt stream keeps recording underneath**, so
  you reproduce the scenario, pop with esc, and inspect what happened — and
  re-open live whenever you want. (A non-capturing split variant was
  considered and deferred; the overlay is simpler and the ring buffer makes
  nothing lost.)
- **Contents**: routing receipts, screen/modal push-pop, focus moves, resize
  facts, `Validate()` violations at startup, and the app's own `Debugf`
  lines — one timeline for "the kit did X" and "my code did Y".
- **Determinism**: sequence numbers, not timestamps; tests can assert on the
  stream.
- **Recording is always on** (bounded ring buffer, last ~500 events) — cheap,
  and release builds simply don't wire the toggle.
- **v0 fence**: scrollable, auto-follow, newest at bottom, default scroll
  keys. No filtering, no search, no pause — if the debug view needs a filter,
  that's the palette's job wearing a hat; don't build it twice.

The README-worthy symmetry: **the stream you watch in dev mode is the same
stream your tests assert on.** Debugging and CI see the same truth.

## 9. Scrolling

Panes are scrollable by default. The kit owns the mechanism
(track/total/height math) and a default key set that cannot collide with
cursor keys: `pgup`/`pgdn`, `ctrl-u`/`ctrl-d`, `home`/`end`. Defaults appear
in help like every other binding (they *are* bindings). A pane with nothing
to scroll just never reports a total — no-op. Scroll offset is pane state;
the kit provides the arithmetic.

## 10. Testing story

- **Headless driver**: render is pure — `f(screen, data, wiring, env)` — so
  tests drive keys and assert frames with no pty.
- **Golden frames**: `kittest.Expect(keys...).Frame(golden)`.
- **Targeted assertions** (no golden brittleness): address structure through
  pane names and group IDs —
  `kittest.SlotOf(app, "jobs", "detail")` asserts resolved geometry,
  `kittest.Region(frame, app, "jobs", "queue")` crops the composed frame to
  one pane's slot for content assertions, and `kittest.Dump(app)` prints the
  annotated tree with resolved rects and extrapolated IDs.
- **Dry-run routing**: assert dispatch decisions without executing.
- **Receipts**: assert the event chain (the same stream debug mode shows).
- **`Validate()` in CI**: wiring rot is a red build, not a code-review catch.
- pty tests remain for the true end-to-end smoke (exit codes, real terminal
  path), but they are the exception, not the workflow.

## 11. Refusals

Naming — each refusal ships with its one-line justification in the README:

| Word | Why refused |
|---|---|
| window | OS-window collision; also implies overlap |
| view | overloaded to meaninglessness |
| widget | that's bubbles' job; we host, never rebuild |
| tab | means tab key; also implies persistence |
| region | means nothing specific to anyone |
| grid | promises CSS-grid semantics (tracks, spans) we don't have — it's a split tree |
| cell | in every TUI library (tcell, termbox) means one character cell — an active trap |

Scope refusals: no widgets, no theming, no mouse, no state management, no
runtime minimum enforcement, no behavior on layout containers, no
per-item keys, no nested screens, no slot naming (the pane is the slot's
address), no mandatory group labels, no ID-based routing.

## 12. Adoption ladder

Every rung independently valuable; you can stop anywhere:

1. **Kill the Model boilerplate** — `kit.Run(app)` wires Init/Update/View.
2. **Add screens** — places, switching, `Min` promises.
3. **Add verbs** — actions, danger→confirm, universal verbs.
4. **Adopt the audit** — `Validate()` in CI.
5. **Adopt headless tests** — golden frames, dry-run routing, receipts.

## 13. Delta vs `internal/tuikit` (what the exercise changed)

| # | Greenfield decision | Current tuikit | Disposition |
|---|---|---|---|
| 1 | Size root-owned; render pure `f(env)` | per-screen `Frame` state (0×0 bug) | **adopt** — already agreed |
| 2 | Kit owns the program (`kit.Run`) | user-written Model boilerplate | **adopt** |
| 3 | Cursor anchored to item IDs | cursor is an index in a slice | **adopt** (robustness upgrade) |
| 4 | Router as dry-runnable query | dispatch produces traces, not queryable | **adopt** |
| 5 | `Validate()` at dev startup + CI | audit is test-only | **adopt both** |
| 6 | Headless driver + golden frames as shipped API | `testrender` internal | **adopt** (promote to `kittest`) |
| 7 | Vocabulary (screens/panes/cursors/details/modals), capture semantics | same | **survives verbatim** — clean slate converges |
| 8 | `Region`→`Pane`, `Frame`→internal `Layout`, `Cell`→`Slot{W,H,Focused}` | `Region`/`Frame`/`Cell` | **rename + merge** (two types → one) |
| 9 | `Title` on any layout node | chrome is pane-only | **adopt** (enables titled splits) |
| 10 | Pane minimums: audit-time fold | audit checks declared minimum only | **adopt** |
| 11 | Debug palette (`~`, w-5×h-5 overlay) | doesn't exist | **new** |
| 12 | Footer as generated projection | doesn't exist | **new** |
| 13 | Refusals list incl. grid/cell | partial | **adopt** |
| 14 | Optional group labels → extrapolated IDs (`main__C1xR0`), unique by construction; `kittest` addressing API (`SlotOf`/`Region`/`Dump`) | anonymous structure; golden-frame tests only | **adopt** |

## 14. Ship sequencing (pre-tag window)

Ordered so each step lands green before the next begins:

1. **Size-as-fact** — frames stop owning `w/h`; render takes root size;
   `resolve` memo keyed by (w,h). Regression gate: pty probe — switch every
   screen, assert no TooSmall.
2. **Vocabulary pass** — Region→Pane, Frame→Layout (internal), Cell→Slot;
   `Title` on layout nodes; `Render(Slot)`/`Notify(Slot)` merge Env;
   optional group labels with extrapolated IDs.
3. **Kit owns Run** — `kit.Run(app)`; embedded-Model escape hatch preserved.
4. **Cursor-anchor** — ID-anchored cursors; lazylite migrates as proof.
5. **Router-as-query** — exported `Route()` receipts; help/palette/audit
   refactored onto it as projections.
6. **Pane-min fold + derived honest minimum** in the audit.
7. **`kittest`** — promote `testrender`; headless driver, golden frames, and
   the addressing API (`SlotOf`/`Region`/`Dump`).
8. **Debug palette + footer** — the two new projections.
9. **Tag v0.1.0** — once. Then vanity path / v0.2.0 per the publish plan.

## Appendix — decision-log seeds

- **D46 — Greenfield convergence.** A from-scratch exercise ("team inside
  charmland") converged on the existing vocabulary and capture semantics;
  the diff produced amendments 1–6, 8–10 above rather than a redesign. The
  architecture was not accident-driven.
- **D47 — Facts are global, promises are per-screen.** The terminal size is
  root-owned; screens hold minimums; nothing below the root stores a `w`.
  Closes the 0×0 multi-screen resize bug by making the bug class
  unrepresentable. (Revises the D44 cascade contract: RegionResize becomes a
  courtesy notification — delivered on resize-while-current and on
  becoming-current — never a source of truth.)
- **D48 — Structure recurses; behavior doesn't.** Chrome (`Title`) is legal
  on any layout node; keys/focus/cursors/scroll exist only on panes. Litmus:
  cursor → pane; title → anything; keys → pane.
- **D49 — Items carry identity, panes carry behavior.** Item-level key
  registration is refused (data-dependent routing tables collapse help,
  palette, and audit). Handlers read the cursor's anchored item.
- **D50 — Two pushable stacks.** Screen stack (replace, don't capture) and
  modal stack (capture); esc pops the innermost; universal verbs answer
  during capture; Danger compiles to a Confirm modal.
- **D51 — Minimums are audit-time.** Pane minimums feed a direction-aware
  fold checked against the screen's `Min` at audit time; runtime layout is
  pure arithmetic with the TooSmall gate as the only size gate. The audit
  reports each screen's derived honest minimum.
- **D52 — Debug is a projection.** Dev mode (`~`, rebindable) renders the
  receipt stream as a big palette overlay (default w-5×h-5); recording is
  always on, sequence-numbered; `Debugf` joins app events to the same
  timeline; the stream tests assert on is the stream developers watch.
- **D53 — Naming verdicts.** grid/cell refused (CSS-grid overpromise;
  character-cell collision); Slot adopted for the per-frame handshake
  ({W,H,Focused}); Layout internal; refuse list gains grid and cell with
  one-line README justifications.
- **D54 — IDs are extrapolated, labels optional.** Layout groups take an
  optional label; the kit extrapolates the canonical ID by appending the
  positional path (`main` → `main__C1xR0`; unlabeled → `__C1xR0`). Unique by
  construction — no uniqueness audit exists. IDs are diagnostics-only
  addresses (violations, tree dump, receipts), never routing scopes, never
  help/palette entries, never key/focus targets; positions shifting with
  tree edits is correct, since diagnostics must track the structure they
  describe. Slots are never named: the pane's name is the slot's address.
