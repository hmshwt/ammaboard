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

## 0. Why this exists

Everyone building a serious TUI on bubbletea reinvents the same layer badly:
screen management, key routing with capture semantics, help text that drifts
from the code, hand-wired confirm flows, and a testing story that starts
with "run it and look." This kit extracts that layer — the *shell* — once,
so the hundredth operational TUI starts from vocabulary and invariants
instead of from `main.go`.

**The focused use case** is tools that are mostly *views over some data*,
driven by keys: take actions on what the cursor points at, drill down into
detail views, switch between places. lazygit-shaped tools — operational
dashboards, list/detail navigators, log explorers, queue inspectors.
Priorities follow from that: **DX for designing such tools first; reliable
testing, CI-based audits, and quick debugging second; runtime performance
explicitly low priority.**

### Non-goals (for now)

- **Scale.** Showing 10,000 items in a pane is not a kit problem. The
  per-frame contract — draw exactly your `Slot` — pushes panes toward
  windowed rendering by construction: a pane with 26 rows of slot draws a
  window into its data, and virtualization lives in the pane, where it can
  be as domain-specific as it needs to be.
- **Async domain work.** Goroutines updating stores, background polling,
  workers — the kit moves keys, geometry, and focus; it never touches
  domain data and never schedules work. That is the developer's domain,
  arriving through bubbletea's own contract (`tea.Cmd` → `tea.Msg`), which
  the kit passes through untouched. The data store is deliberately outside
  the kit.
- **Complex cases.** Mouse-first apps, rich text editing, animation,
  dashboards of hundreds of simultaneous widgets — out of scope.

These are no-goals, not eternal truths. The project came out of view-based
needs with reliable data on real use cases, so concerns outside that
evidence were left to the developer rather than guessed at. Immediate-mode
rendering was chosen for the same reason: every frame is a pure derivation
of data, wiring, and environment — which is what keeps the kit simple,
reliable, and correct, and what makes its testing story possible at all.
The project is open to discussion and accepts concrete use cases as the
basis for evolving any of these boundaries in future versions.

## 1. The pitch

> Bubbletea gives you the *loop*. This gives you the *shell*: places, verbs,
> and the contracts between them — so that routing, documentation, and
> verification are all projections of one declarative table instead of three
> hand-maintained lies.

Stated as a thesis: **everything terminal-app authors reinvent badly —
screens, key routing, capture, confirm flows, scroll/cursor edge cases,
help text, tests — extracted once as a thin layer that treats bubbletea as
the foundation it already is.**

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

The layer stays out of the way by design, in three tiers: **utilities** you
grab or skip (scroll, cursor, wrap math), **conventions** you adopt because
they pay (the vocabulary, the Track handshake, placement-based scoping),
and **contracts** that deliberately get in your way (reserved verbs,
capture semantics, danger→confirm, the audit) — because those are promises
to the app's *users*, not restrictions on the developer. It stays out of
your way except where it's standing guard, and the guards are for your
users.

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
| **Action** | a verb (named or mechanic) | key, handler, help; optional `Name`, `NeedsCursor`, `Danger` | geometry |
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

### The Track handshake — the kit's only sync contract

Kit state that must stay coupled to your data (the cursor's selection, the
scroll's anchor) holds **pointers, never contents**. The sync contract is
one shape, used exactly twice:

- **Declare in** — once per render, the pane hands the fresh content:
  `cursor.Track(ids)` / `scroll.TrackLines(total, viewH)`.
- **Derive out** — pure functions: `Index()`, `Offset(viewH)`. No side
  effects, safe to call anywhere.

State persists between frames but is **re-validated every frame** against
the fresh declaration — clamped, re-anchored, confirmed. Deliberate writes
(a search seek, a jump to a line) go through the same guarded transitions
as kit actions, so every write gets the boundary safety. And the refusal
that follows: **no change hooks anywhere in the kit.** "Notify me when data
changes so I can fix up my state" is the retained-mode reflex; Track
replaces it, and staleness has a maximum lifetime of one frame.

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

### Reserved universal verbs — pinned catalog entries

Exactly three, reserved at the root. They are catalog *entries* whose keys
are frozen — the pinned class of the catalog (§7):

- `GLOBAL_UNWIND` — `esc`: pops the innermost pushable (modal stack first,
  then screen stack), else blurs focus. One verb, contextual effect; the
  receipt records which happened (`popped: modal:confirm` /
  `popped: screen:detail` / `effect: blur`). Inside a derived confirm
  modal, popping *is* the dismissal — `CONFIRM_NO` remains the explicit
  deny key.
- `GLOBAL_HELP` — `?`: generated help (cannot lie)
- `GLOBAL_PALETTE` — `:`: generated palette

Nothing else is reserved. Digits are the app's — the default wiring is
**1–9 = switch screens** in registration order (digits go to the coarsest
navigation: screens, not slots; slot-jumping is tab's job). `ctrl+c` quits
by default via a rebindable quit action (`GLOBAL_QUIT`).

**Why the three are pinned, never rekeyable:**

1. *They are promises to the app's users, not config for the author.* esc
   means "back out" in every terminal app ever written; kit apps honor that
   before their authors get an opinion. Rekeyable defaults let apps
   specialize; pinned verbs keep the ecosystem uniform.
2. *Liveness by construction.* The capture system's correctness rests on
   "every capturable state has exactly one guaranteed exit." If unwind were
   rebindable, that guarantee becomes conditional and would need a hard
   liveness audit across all runtime states; pinned, no wiring can break it
   — you never audit what cannot be misconfigured.
3. *Capture is implemented through them.* Universal verbs are the exception
   path inside the capture machinery; pinned, the dispatcher knows
   statically what esc does, with zero lookup and zero drift.
4. *The pinned exit forces honest mode modeling.* Nuance is expressed by
   pushing a level, never by rekeying the exit — modes become visible,
   pushable, receipt-recorded things instead of hidden booleans with
   private key handlers.

Enforced by one audit rule: **binding `esc`, `?`, or `:` anywhere is a
violation.**

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

**The palette's scoping — one resolver, never two.** The palette does not
show "all actions registered"; it shows the routing table **intersected
with the current context** (focused pane + current screen + global). The
invariant: **every palette entry, when selected, executes exactly as if its
key had been pressed right now** — the receipt cannot tell the two triggers
apart (except the trigger field). The only way this cannot lie is
structural: **palette and dispatch resolve scope through the same
function.** A palette with its own narrowing logic is a lie vector.

Consequences: the palette *shrinks as you drill* — context-relevant verbs,
not a command dump. Unfocused sibling panes' actions do not appear (v0 is
strict; routing only delivers keys to the focused pane). A future opt-in
"reachable" mode may list sibling actions with stated prerequisites
("kill container · switches to containers"), receipting the focus move —
the invariant survives because the entry states what will happen.
"Search everything across screens" is refused: the palette is an answer,
not a search index.

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

### The verb layer — one Action type

```go
type Action struct {
    Key         string   // "j", "K", "ctrl-d", "?"
    Run         Handler  // kit-provided or yours
    Help        string   // footer/help text — every action has one
    Name        string   // optional: empty = mechanic, set = verb identity
    NeedsCursor bool     // optional: Run reads the focused pane's cursor
    Danger      bool     // optional: compiles to a confirm push
}

type Bindings = []Action // an alias — the table everything derives from
```

One type replaces the old Key/Bindings/Action trio. `Key` dies as a type —
it was always "Action without a name" — and the mechanic-vs-verb line
survives as `Name` empty vs set: the palette lists `Name != ""`, kit
navigation actions ship unnamed so the default path is right by default,
and naming is opt-in. Palette spam (`j — move down`) is a taste choice the
kit permits but the defaults avoid.

**Scope is placement, not a field.** An action's scope is where it is
registered — a pane's `Keys()`, a screen's block, the app's global block.
The registration site is self-verifying; a scope *field* would be a claim
needing its own audit rule, and centralized declaration would break pane
reuse: a pane must carry its own verbs.

**`NeedsCursor` makes the handler/target contract static** — an audited
property, not a runtime guess. Audit rule: `NeedsCursor` ⇒ pane scope (a
global or screen action claiming a cursor target is rejected — "kill the
focused thing" as a global verb is exactly the anomaly this rule catches).
Receipts gain a uniform `target` field, recorded when such actions run.
Handlers receive one context:

```go
type Ctx struct {
    App    *App
    ViewH  int   // focused pane's current viewport height — kit-owned geometry
    Target Item  // {ID string; Index int} — valid iff NeedsCursor
}
```

### Cursor wiring — embed, Track, resolve

```go
type QueuePane struct {
    kit.Cursor            // ① the pane's selection: one ID + scroll offset
    store *Store
}

func (p *QueuePane) Render(s kit.Slot) string {
    p.cursor.Track(idsOf(p.store.Open())) // ② the handshake: live IDs, every frame
    i := p.cursor.Index()                 //    derived: nearest live row
    // draw; highlight where item.ID == p.cursor.Selected() via kit.Selected
}

func (p *QueuePane) Keys() kit.Bindings {
    return kit.Bind(
        kit.Act(kit.CURSOR_NEXT, "j"),    // ③ kit nav — runs against the embedded cursor
        kit.Action("kill", "K", p.kill).Help("kill job").Danger().NeedsCursor(),
    )
}

func (p *QueuePane) kill(c kit.Ctx) { p.store.Kill(c.Target.ID) } // ④ identity, not index
```

- **Embed is the wiring**: `kit.Cursor` carries `Cursor() *kit.Cursor`, so
  embedding satisfies the kit's discovery interface via Go promotion — the
  kit finds the focused pane's cursor with zero methods written.
- **Track** hands the fresh ID list each render; the selected ID is
  re-anchored against it — kill the last row and the selection clamps to
  the nearest live row. Never panics, never drifts.
- **Resolve happens at dispatch**: the kit converts "the selected ID" into
  `Ctx.Target` the moment an action actually runs — keypress and palette
  triggers take the identical path.
- **Edge rules**: a `NeedsCursor` action with no selection doesn't fire —
  the receipt records `target: none`; the handler never nil-checks.

### The kit catalog

**Guardrail: the kit ships actions only for machinery the kit owns** — the
cursor, scroll, both stacks, palette, help, debug, confirm compilation. It
never ships data verbs (no `refresh`, no `delete`) because it has no store.

| Scope | Actions | Defaults |
|---|---|---|
| global | `GLOBAL_HELP`, `GLOBAL_PALETTE`, `GLOBAL_UNWIND` | `?`, `:`, `esc` — **pinned** (§4) |
| global | `GLOBAL_QUIT`, `GLOBAL_DEBUG`, `DEBUG_WHERE` | `ctrl+c`, `~` — overridable |
| screen (generated) | `goto <screen>` — one per registered screen | digits `1`–`9` — digits become data: palette lists them, footer hints them, the audit sees them |
| pane | `CURSOR_NEXT/PREV/FIRST/LAST/PAGE_UP/PAGE_DOWN` | `j`/`k`, `g`/`G`, page steps — mechanics |
| pane | `SCROLL_UP/DOWN/TOP/BOTTOM` | `pgup`/`pgdn`, `ctrl-u`/`ctrl-d` — mechanics |
| modal | `CONFIRM_YES`/`CONFIRM_NO` + palette internals | `enter`/`y`, `esc`/`n` — overridable confirm grammar |

Override rule: registering the same name+scope as a **kit default**
replaces it (visible, intended); a duplicate name+scope of **your own**
action is a violation — palette and receipt identity must be unambiguous.
Drill-down (`enter` opens detail) is deliberately not kit-provided — what
"open" means is your semantics. Layout mutation and per-item verbs are
refused (§11).

### Projections

```go
app.Route("K", kit.On("jobs")) // → Receipt{Action: kill, Scope: "jobs", Confirm: true}
app.Help(kit.Global)           // → generated columns
app.Validate()                 // → []Violation — run in dev startup AND CI
app.Debugf("queue drained: %d jobs", n) // app events join the same timeline
```

### Theming — four roles, made of lipgloss

The kit themes exactly what it draws, through four semantic roles; a
consumer's theme is four lipgloss styles — there is no theming system to
learn:

```go
type Theme struct {
    Border   lipgloss.Style // pane outline + modal boxes
    Title    lipgloss.Style // title text
    Chrome   lipgloss.Style // chrome background; footer, help, palette surfaces
    Selected lipgloss.Style // highlighted item fg/bg — applied by kit.Selected
}

kit.New(app, kit.WithTheme(myTheme)) // tasteful default otherwise
```

The one rule that keeps the theme truthful: **the kit can only theme what
the kit draws.** Three of the four roles are kit-drawn surfaces. The fourth
— the highlighted item — is drawn by *panes*, so it is themed through a kit
primitive panes call:

```go
// inside a pane's Render:
if s.Focused && job.ID == p.cursor.Selected() {
    line = kit.Selected(line) // applies the theme's fg/bg
}
```

Panes that use the primitive get theming for free; panes that hand-roll
lipgloss keep full styling control (the escape hatch). Panes receive the
effective theme via optional injection at wire-up (`SetTheme(kit.Theme)`)
before the first render — the same courtesy pattern as `Notify`; no
package state. Because the roles are semantic rather than visual, a
theme's meaning survives a restyle, and because the fields *are* lipgloss
styles, color-profile degradation comes free from lipgloss instead of being
rebuilt here. `kittest` pins the default theme for golden frames (same
input + same theme → same frame), with a `WithTheme` escape for testing
custom themes.

### Type budget

`App, Screen, Layout, Slot, Pane, Action, Ctx, Cursor, Modal, Theme,
Receipt, Violation` — twelve, plus the `Bindings` alias (`[]Action`), and
it should stay there.

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

## 9. Scrolling — anchored, two modes, no hooks

Panes are scrollable by default; the kit owns the mechanism and the edge
cases. Scroll state is the cursor's philosophy applied once more — pointers,
never contents, synced by the Track handshake (§3):

```go
type Scroll struct {
    anchor string // item mode: ID of the top-visible row
    line   int    // line mode: first visible line (anchor == "" ⇒ line mode)
    pinned bool   // was clamped to the bottom before the last Track
}
```

Two modes, because operational TUIs scroll two kinds of content: **item
mode** (list panes, anchored by identity — `Track(ids)`) and **line mode**
(logs, detail views, anchored by line number — `TrackLines(total, viewH)`).

The transition table — the whole machine:

| Event | Transition |
|---|---|
| `Track` / `TrackLines` | pinned → chase the new bottom; else re-anchor/clamp in place |
| `SCROLL_DOWN` / `SCROLL_UP` | `line ±= step` (step = viewH; half-page variant for ctrl keys), clamp; `pinned` set/cleared at the boundaries |
| `SCROLL_TOP` / `SCROLL_BOTTOM` | `line = 0 / max`; `pinned = false / true` |
| `Offset(viewH)` | **pure** clamp — the only function render calls; side-effect-free, so re-renders and dry-runs cannot mutate state |

What this absorbs, so pane authors never reinvent it: boundary clamps and
idempotent no-ops; tail-follow only while pinned (growth chases the bottom
for a pinned viewport, preserves an unpinned user's place); re-anchoring on
removal; pure-clamp resize handling; empty and shorter-than-viewport
content. Initial position is explicit policy — default top,
`kit.ScrollTail()` opt-in for logs.

**Line mode is append-stable only.** Line anchors are positions, not
identities: head-trims (ring-buffer rotation) and mid-content rewrites
renumber lines and drift the viewport. Content that mutates in the middle
belongs in **item mode** — and a rotating log's lines have natural IDs
(sequence numbers). Ownership: cursor-having panes scroll through the
cursor (it owns the offset; its `Offset` clamps *and* follows the
selection); cursorless panes embed `kit.Scroll`; audit rule — at most one
scroll owner per pane. Horizontal scrolling is refused (width overflow is
the pane's truncate-or-wrap business), and `TrackLines` totals must be
*rendered* heights — post-wrap — not logical line counts.

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

Scope refusals: no widgets, no theming *system* (no dark/light variants, no
profile detection, no runtime switching, no theme files — theming is a
four-role struct of lipgloss styles, §7), no mouse, no state management, no
runtime minimum enforcement, no behavior on layout containers, no
per-item keys, no nested screens, no slot naming (the pane is the slot's
address), no mandatory group labels, no ID-based routing, no change hooks
anywhere (the Track handshake is the only sync contract), no horizontal
scrolling, no rebinding of reserved verbs.

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
| 15 | Minimal theme: four semantic lipgloss roles (`Border`/`Title`/`Chrome`/`Selected`) + `kit.Selected` primitive | hardcoded kit chrome styles | **adopt** |
| 16 | One `Action` type — `Key` dies, `Bindings = []Action`, `Name` empty/set carries the mechanic-vs-verb line, `NeedsCursor` audited | `Bind().Run().Help()` key chains; no static target contract | **adopt** |
| 17 | Kit action catalog + own-machinery guardrail; reserved (`GLOBAL_UNWIND` et al.) vs overridable classes | ad-hoc reserved keys | **adopt** |
| 18 | Anchored scroll — Track handshake, two modes, pure `Offset`, tail-follow | stateless `Scroll.Window(track, total, height)` helper | **adopt** |
| 19 | Palette = one resolver (table ∩ context; executable-entries invariant) | no formal scope narrowing | **adopt** |

## 14. Ship sequencing (pre-tag window)

Ordered so each step lands green before the next begins:

1. **Size-as-fact** — frames stop owning `w/h`; render takes root size;
   `resolve` memo keyed by (w,h). Regression gate: pty probe — switch every
   screen, assert no TooSmall.
2. **Vocabulary pass** — Region→Pane, Frame→Layout (internal), Cell→Slot;
   `Title` on layout nodes; `Render(Slot)`/`Notify(Slot)` merge Env;
   optional group labels with extrapolated IDs; the `Theme` struct and
   `kit.Selected` primitive (Render conventions are set here, where the
   convention is cheapest to establish); the verb layer collapses to one
   `Action` type (`Bindings = []Action`).
3. **Kit owns Run** — `kit.Run(app)`; embedded-Model escape hatch preserved.
4. **Cursor-anchor + Track handshake** — ID-anchored cursors, `Ctx` targets
   (`NeedsCursor`, `ViewH`), scroll anchoring; lazylite migrates as proof.
5. **Router-as-query + catalog** — exported `Route()` receipts;
   help/palette/audit/footer refactored onto it as projections under the
   one-resolver rule; the kit catalog (globals, goto, cursor, scroll,
   confirm).
6. **Pane-min fold + derived honest minimum** in the audit.
7. **`kittest`** — promote `testrender`; headless driver, golden frames, and
   the addressing API (`SlotOf`/`Region`/`Dump`).
8. **Debug palette + footer** — the two new projections.
9. **Tag v0.1.0** — once. Then vanity path / v0.2.0 per the publish plan.

## 15. Architecture — the cheat sheet

Twelve types and an alias, three diagrams. Everything else in this doc is
commentary.

### The composition chain

```
App ── owns: terminal size · universal verbs (esc ? :) · both stacks · Theme
 │
 ├── screen set (siblings — digits 1-9 switch, replaces stack top)
 │      └── Screen: Min promise + one Layout + its keys
 │             └── Layout: Row/Cols tree (Fill/Fixed/Ratio · Title · ID)
 │                    └── deals one Slot{W,H,Focused} per leaf, every frame
 │                           └── Pane.Render draws exactly that
 └── stacks (runtime): screen stack (drill-down) · modal stack (capture)
```

### Dispatch — one table, three scopes

```
key in
  │
  ├─ modal stack top? ── yes ──► it captures; universal verbs still answer
  │      no                      (esc pops innermost, ? help, : palette)
  ▼
focused pane Keys ──► screen Keys ──► global verbs     (first consumer wins)
  │
  └── every dispatch leaves a Receipt
```

### The frame function

```
frame = render(screen, data, wiring, env)

  data    yours — the kit never touches it
  wiring  the bindings table (help, palette, footer, audit are its projections)
  env     {size, focus} — App-owned facts; nothing below the root stores a w
```

### Static vs runtime

| Declared once | Pushed and popped |
|---|---|
| screen set, layouts, bindings, theme, minimums | screen stack, modal stack, cursor, scroll |

### The twelve types (plus the `Bindings` alias)

| Type | One line |
|---|---|
| `App` | root: owns size, verbs, both stacks, theme — never your data |
| `Screen` | a named place; `Min` promise + one layout; digits 1–9 default |
| `Layout` | `Row`/`Cols` split tree; `Fill`/`Fixed`/`Ratio`; optional `Title`/`ID` |
| `Slot` | per-frame rectangle `{W, H, Focused}` dealt to each leaf |
| `Pane` | leaf: `Render(Slot)`; optional `Keys`/`Notify`; scrollable by default |
| `Action` | one verb: key → handler, `Help`; optional `Name` (verb vs mechanic), `NeedsCursor`, `Danger` |
| `Ctx` | what a handler receives: `App`, viewport height, resolved `Target{ID, Index}` |
| `Cursor` | anchored to item IDs; Track in, derive out; never panics |
| `Modal` | capturing overlay on the modal stack; esc pops one level |
| `Theme` | four lipgloss roles: `Border`, `Title`, `Chrome`, `Selected` |
| `Receipt` | one record per dispatch — the stream debug mode and tests share |
| `Violation` | audit output: collisions, dead keys, help lies, unconfirmed danger, undersized `Min`, cursor claims off-scope |

`Bindings = []Action` is the alias that names the table. `esc`/`?`/`:` are
the pinned catalog entries `GLOBAL_UNWIND`/`GLOBAL_HELP`/`GLOBAL_PALETTE`.

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
- **D55 — The kit themes what it draws.** Theming is refused as a *system*
  (dark/light variants, profile detection, runtime switching, theme files)
  and adopted as a *config*: one `Theme` struct of four semantic roles
  (`Border`, `Title`, `Chrome`, `Selected`), each a lipgloss style —
  passing a custom theme is passing lipgloss, and color-profile degradation
  is inherited from lipgloss rather than rebuilt. `Selected` is honored by
  the `kit.Selected` primitive panes call for the highlighted item; a
  theme entry backed by nothing would be a lie, so hand-rolled lipgloss in
  panes remains the escape hatch. Themes inject at wire-up (`SetTheme`),
  never global; golden tests pin the default theme. The convention lands
  pre-tag because retrofitting it after panes hand-roll styles grows
  expensive with every consumer; ammaboard and lazylite migrating on four
  roles is the sufficiency proof.
- **D56 — One Action type.** The verb layer collapses into a single
  `Action` (`Key`, `Run`, `Help`, `Name`, `NeedsCursor`, `Danger`);
  `Bindings = []Action` is an alias; `Key` dies as a type. Three
  corrections kept the merge honest: scope is placement-based, never a
  field (self-verifying; preserves pane self-containment); the cursor's
  *dependency* merges as the audited `NeedsCursor` (pane scope only) while
  the cursor itself stays infrastructure; the mechanic/verb line survives
  as `Name` empty vs set. Type budget 13 → 12 plus alias.
- **D57 — The kit catalog.** The kit ships actions only for machinery it
  owns (cursor, scroll, stacks, palette, help, debug, confirm) — never data
  verbs. Two classes: **pinned** reserved verbs (`GLOBAL_HELP`,
  `GLOBAL_PALETTE`, `GLOBAL_UNWIND` — keys frozen; liveness by construction;
  the pinned exit forces push/pop mode modeling) and **overridable**
  defaults (`GLOBAL_QUIT`, `GLOBAL_DEBUG`, confirm grammar, cursor/scroll
  mechanics). Overrides are same-name+scope replacements; duplicate app
  names are violations. `goto <screen>` is generated per registered screen —
  digits become data.
- **D58 — The palette is one resolver.** The palette shows the routing
  table ∩ current context; every entry executes exactly as its keypress
  would; palette and dispatch resolve scope through the same function — a
  separate filter would be a lie vector. Strict v0 (focused pane only);
  reachable sibling entries with stated prerequisites are a future opt-in;
  search-everything is refused.
- **D59 — Anchored scroll; the Track handshake; no hooks.** Scroll state is
  `{anchor, line, pinned}` — pointers, never contents; item mode (IDs) and
  line mode (append-only text); `Track*` declares content per render,
  `Offset(viewH)` is a pure clamp, mutations happen only in Track and
  actions. Tail-follow is one rule; line mode drifts under head-trim and
  rewrite, so mutating content belongs in item mode (log rotation = seq
  IDs). The Track handshake is the kit's only data-sync contract (cursor
  and scroll are its instances); change hooks are refused everywhere — the
  retained-mode reflex declined.
