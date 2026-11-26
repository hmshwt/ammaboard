# tuikit — specification

> Status: **ready for the step-0 implementation plan — review closed.** This
> document is the single
> source of truth for the kit's contract. It consolidates the clean-slate
> design, the extracted-kernel contract, and the resolutions of four
> architecture reviews. There are no further review rounds: from here the
> contract is proven or revised by code, and every change is traced to a
> failing test. This revision is the one patch that closed review; each of
> its changes is logged as a named test (§17.1).
>
> **What is checked, and how.** The day-one program (§10.1–§10.2) is
> extracted from this document by script and compiled, vetted and
> gofmt-checked against the real bubbletea v2 (verified on v2.0.9), then
> **audited** — the audit rules that need no runtime are run on it, and its
> report must be clean. The default keys in the catalog (§6.4) are read from
> the same text and checked against real key spellings. Other code blocks
> are sketches, not programs. Sections marked **step 0** freeze only when
> their slice's tests (§17.1) pass; until then a gap is an open question,
> not a rule.
>
> Working name: **tuikit** (package `tuikit`; test helpers `tuikittest`).
> Placeholder pending the final name.

---

## 1. Overview

Building an operational TUI on bubbletea requires repeatedly solving
interaction correctness, navigation, discoverability, and testability — and
the underlying framework provides no coherent integrated solution. tuikit
extracts that layer once.

> **A reliable interaction layer for bubbletea applications.** Declare the
> app's screens, actions, and interaction rules once. The same declarations
> drive routing, contextual help, command discovery, validation, debugging,
> and tests. If any two of these disagree about anything the kit observes,
> it is a defect in the kit.

The central value is the reduction of **interaction drift**: one declared
interaction model produces execution, discoverability, validation,
debugging, and test assertions.

**Positioning:** a small framework with escape hatches at every layer. It
owns `Run`, routing, layout resolution, and reserved keys — framework-shaped
by honest admission — but has no global state, no service locator, never
touches domain data, and specifies a bypass for every ownership boundary
(§10.5). Raw bubbletea is always one delegation away.

**Trust anchor:** the escape hatch, designed rather than asserted (§10.5) —
`AsModel` for an app that owns its own program, the host hook for messages
and unbound keys, pane-owned rendering. One hatch (a per-pane raw-key
interface) is not built, and the claim is narrowed to match. The
unbypassable parts are promises to the app's *users* (reserved exits,
capture semantics, confirm flows), not restrictions on the developer.

Prior art (screen shells, single-table help generators, bindings-as-data
libraries) exists; on the evidence gathered so far none of it leads with
**verifiable wiring** — an audit that fails CI, dry-run routing assertions,
and machine-readable receipts. That is the differentiation, and the pitch
leads with it. The evidence is secondary and gets checked by a prior-art
pass before the tag (§17.1); if the claim does not survive, the pitch
changes, not the pass.

## 2. Scope

### 2.1 Ownership

| tuikit owns | tuikit never owns |
|---|---|
| Places: screens, panes, layouts, slots | Domain data — the store is the app's own Go struct; no kit type touches it |
| Verbs: bindings, actions, routing, capture | Styling — lipgloss is passed through, not rebuilt |
| Generated projections: help, palette, footer | The render loop — bubbletea |
| Verification: audit, receipts, dry-run | Async domain work — `tea.Cmd`/`tea.Msg` pass through untouched |
| Environment facts: size, focus, theme | State management, mouse, widgets (bubbles is hosted, never rebuilt) |

### 2.2 Focused use case

Tools that are mostly *views over data*, driven by keys: take actions on
what the cursor points at, drill into detail, switch between places.
Lazygit-shaped: operational dashboards, list/detail navigators, log
explorers, queue inspectors.

Priorities: **DX for designing such tools first; reliable testing, CI-based
audits, quick debugging second; runtime performance explicitly last.**

### 2.3 Non-goals

- **Scale.** 10,000-item panes are not a kit problem; the per-frame
  contract ("draw exactly your Slot") pushes panes toward windowed
  rendering by construction. Virtualization lives in the pane.
- **Complex cases.** Mouse-first apps, rich text editing, animation,
  widget dashboards — out of scope (§15).
- These are boundaries, not eternal truths. Concrete use cases can move
  them in future versions.

## 3. Principles (non-negotiables)

1. **One direction of trust.** The kit consumes `tea.Msg`/`tea.Cmd`, never
   hides them. Pinned keys and geometry are handled before any app code
   runs. Non-key messages reach the host hook and then every live pane,
   each instance once; keys travel only through the table and then the
   hook. The only code that can stop a delivery is the app's own host hook
   (§5.3). Pass-through is structural and observable, not promised.
2. **Everything is data; the rest is derivation.** Bindings are data →
   routing, help, palette, footer, audit, receipts, and README key tables
   are generated views of the same table. If any two of them **disagree
   about anything the kit observes**, the design is wrong. (The kit
   eliminates inconsistencies it can observe and structurally guarantee;
   it cannot prove semantic truth — that a handler does what its help says
   is the app's claim.)
3. **Facts have one owner.** Exactly one terminal, so exactly one size; it
   lives at the root. Screens hold *promises* (`Min`), never facts. Nothing
   below the root stores a `w`. The 0×0-resize bug class is made
   unrepresentable, not caught.
4. **Determinism first.** Same input → same frames. Receipts carry sequence
   numbers, never timestamps. Tests run headless.
5. **No change hooks.** The declare handshake (§4.5) is the only data-sync
   contract; staleness has a maximum lifetime of one sync phase.
6. **Structure recurses; behavior doesn't.** Any layout node may carry a
   title; only leaves carry keys, cursors, focus, and scroll.

## 4. Vocabulary

### 4.1 The five nouns

| Concept | Is | Carries | Never carries |
|---|---|---|---|
| **Screen** | a named place | `Min` promise, named layout variants + a selector, keys, `Reaches` | children screens, its own size |
| **Layout** (`Row`/`Cols`) | anonymous split tree — `Row` stacks top to bottom, `Cols` places side by side | constraints (`Fill`/`Fixed`/`Ratio`), optional `Title`, optional label | keys, focus, cursors |
| **Pane** | leaf | `Render(Slot)`; optional keys, declarations, `Metrics`, `Init`/`Update`, `Min` | children panes |
| **Action** | a verb | keys, handler, help; optional `Title`, `NeedsCursor`, `Danger` | geometry |
| **Modal** | capture | view, optional keys/update/init, text-capture flag | routing scope, slots, focus ring |

Screens switch, panes tile, cursors point, details stack, modals capture.
A pane enters a layout through `Leaf(name, pane)` (§10.1).

### 4.2 The nesting rule

> Any layout node can carry a title (chrome). Only leaves carry keys,
> cursors, focus, and scroll.

This makes the hard questions boring by construction:

- focus ring = flat list of leaves; containers never appear in it
- routing = four scopes, forever: modal → pane → screen → global (a modal
  on top routes first; no path precedence)
- capture/esc = stack depth, independent of tree depth
- audit walks leaves for behavior, any node for claims

Litmus: a box with a cursor is a pane; a box with a title can be anything;
a box with keys must be a pane. Chrome is exactly `Title` — not decoration,
but *data* (help names scopes by it).

### 4.3 Identity model

Two kinds of identity, explicitly separated:

| Identity | What it is | Used by | Lifetime |
|---|---|---|---|
| **Structural** | `pane name @ screen name` (layout groups: optional label + position-derived token) | routing scopes, help, receipts, diagnostics | the declared wiring |
| **State** | the pane *instance* value | cursor, scroll, view state | the instance |

Rules:

- The screen *registry* holds **named factories**; the screen **stack**
  holds instances, and the only way onto it is `Push(name, params)` (§7.2).
  Bare instances never reach the stack, so nothing the app can show is
  invisible to the audit. The audit constructs every factory with nil
  params, records what it declared, and discards the instance — it is never
  initialized, updated, or rendered. Two entries of the same screen type
  are two instances with independent state, by construction.
- **Roots persist; details do not.** A root is built once at wire-up and
  lives as long as the program; `Switch` shows that instance. A detail is
  built when pushed and discarded when popped (§8).
- The same pane *instance* mounted on two screens shares its cursor —
  legal and documented (drill-down preserves selection).
- Layout group IDs are extrapolated (`main` → `main__C1xR0`; unlabeled →
  `__C1xR0`) and unique by construction. They are **diagnostics only**:
  violations, tree dumps, receipts. Never routing scopes, never help or
  palette entries, never key or focus targets — that fence keeps
  behavior-on-containers from returning through the diagnostics door.
- Slots are never named: the pane's name is the slot's address.

### 4.4 Items carry identity; panes carry behavior

Panes contain **items** — rows of content with stable IDs. The cursor
anchors to an item ID; the index is derived against the live list each sync
phase (killing the last row can never panic or drift). Items never register
keys: if keys lived on items, the routing table would become data-dependent
and help, palette, and audit would collapse. The rule: **the pane declares
keys statically; every handler reads the cursor's item.** Different item
types needing different verbs dispatch by type *inside* the handler.

### 4.5 The declare handshake — the kit's only sync contract

Kit state that must stay coupled to app data (cursor selection, scroll
anchor) holds **pointers, never contents**. The handshake, one shape, used
exactly twice (cursor, scroll):

- **Declare** — the pane exposes declarations (`ItemIDs()`,
  `LineCount(width)`); the kit collects them in the **sync phase** (§5.2),
  which runs after geometry resolves and never during render.
- **Read** — `Index()`, `Selected()`, `Offset()` take no arguments and
  return what the last sync stored. No side effects; safe to call anywhere.

The sync phase is the only writer of cursor and scroll state, apart from
the mechanics that move them. State persists between sync phases but is
**re-validated every sync** — clamped, re-anchored, confirmed. Deliberate
writes (a search seek, a jump) go through the same guarded transitions as
kit actions, so every write gets the boundary safety. **No change hooks
anywhere.** "Notify me when data changes" is the retained-mode reflex; the
handshake replaces it.

## 5. Runtime contract

### 5.1 The frame pipeline *(step 0)*

```
1. App receives a message (key press, tea.Msg, resize)
2. The kit routes it — pinned keys and geometry first (§5.3, §5.5)
3. Queued effects apply: stack, focus, modals — after fan-out, in call order
4. Each screen's selector picks a layout variant (may depend on app state)
5. Geometry resolves; slots are dealt (content areas; chrome excluded)
6. Sync phase — with resolved slots: the kit collects declarations and
   writes normalized cursor and scroll state
7. Panes render — read-only
8. Chrome composes the frame
```

**Sync follows geometry because declarations need facts.** `LineCount(width)`
wraps against the slot's width; scroll tail-follow needs the slot's height.
A sync that ran first would normalize against stale dimensions.

**Render purity, stated precisely:** `Render(Slot)` is read-only. All kit
state mutation happens in the sync phase or in guarded action transitions.
Same (data, wiring, env) → same frame, regardless of render call count.
A dry-run runs steps 4–6 on copies and stops there (§13).

### 5.2 The sync phase

After fan-out and geometry — once slots are dealt — the kit collects
declarations from every visible pane and **writes** the normalized state:

| Declaration | Pane method | The sync phase writes |
|---|---|---|
| Item order | `ItemIDs() []string` | cursor anchor re-anchored to nearest live ID; duplicate IDs reported (`sync` event + violation); `NeedsCursor` actions refuse a duplicated ID |
| Rendered height | `LineCount(width int) int` | post-wrap height; scroll offset clamped; tail-follow only while pinned |
| Body geometry | `Metrics` — optional (§9.5) | which rows count as items: body rows, and visible items from row height |

- **Sync is the writer.** `Index`, `Selected`, and `Offset` return what the
  last sync stored. They are pure because they only read — nobody else
  stores a scroll position.
- **Visible panes only.** A hidden pane has no slot, so its `LineCount` has
  no width; it is skipped until it is visible again. That is safe for
  targets because a target is never read from stored state (§6.5).
- Removal recovery keeps the last known index for nearest-row anchoring
  instead of falling to 0. Pure panes implement nothing; the sync phase
  skips them.

### 5.3 Message order and async contract *(step 0)*

The kit owns `Init`/`Update`/`View`. Every message takes exactly one path:

```
message in
  │
  ├─ resize (tea.WindowSizeMsg) ──► the kit applies it as a fact, first.
  │                                 the host hook may observe; it cannot consume.
  │
  ├─ key press (tea.KeyPressMsg) ─► 1. esc / ctrl+c ......... the kit, always
  │        key releases are          2. a modal is on top? ... its Keys() — a match
  │        dropped, never               dispatches; no match: text capture?
  │        dispatched                              → its Update   else → dropped
  │                                 3. the table ............. pane → screen → global
  │                                 4. still unbound? ......... the host hook, then dropped
  │                                 keys never reach a pane's Update
  │
  └─ any other message ───────────► a paste, with a text-capture modal on top?
  │                                  → that modal alone
        otherwise: the host hook first — true means consumed; else every
                  live pane and open modal, each instance exactly once
  └─ after fan-out: queued effects apply → sync → render
```

**Why this order:**

- **Pinned exits first.** `esc` and `ctrl+c` are handled before any app
  code runs. The table, a modal, and the host hook never see them, so no
  wiring can break the promise that every state has an exit. A hook cannot
  consume a resize either: size is a fact (§3, principle 3).
- **The table before the hook.** A bound key always does what the table
  says. A hook that ran first could shadow the table, and help would then
  lie. The hook sees only unbound keys — an app-level raw-key path
  (§10.5) with no power over declared verbs.
- **No `handled` on panes.** A pane returns only a command; `nil` means
  "nothing to run", not "ignored". First-handler-wins among panes would
  starve later panes, and two instances of one screen type would swallow
  each other's replies.
- **Paste follows capture.** With a text-capture modal on top, a paste goes
  only to that modal's `Update` — not to the host, not to panes, not to
  other modals. Pasted text is input to the field that has focus;
  broadcasting it lets a background pane or the host consume what the user
  typed into a prompt.

```go
// Every app callback receives the same Ctx: one effects vocabulary everywhere.
type Handler    func(c Ctx) tea.Cmd
type Initializer interface{ Init(c Ctx) tea.Cmd }
type Updater    interface{ Update(c Ctx, msg tea.Msg) tea.Cmd }
type Host       interface{ Handle(c Ctx, msg tea.Msg) (tea.Cmd, bool) }
```

- **`Ctx` everywhere.** A pane reacting to a message can `Push`,
  `OpenModal`, `Notice`, or `Quit` exactly as a key handler can. Effects
  called during fan-out are **queued** and applied after the last pane has
  received the message, in call order — a pane cannot change the stack
  underneath a sibling mid-delivery. `Ctx.Target` is valid only inside a
  `NeedsCursor` handler.
- **The host hook** is declared with `tuikit.WithHost(h)` and is optional.
  For a non-key message it runs first and may consume; for a resize it only
  observes; for a key it sees only what the table left unbound.
- **Commands.** Callbacks return `tea.Cmd`; the kit wraps each one before
  forwarding (§5.4) and never schedules work itself or owns goroutines.
- **`Init`:** a root's panes at wire-up; a detail screen's panes when it is
  pushed; a modal when it is opened. That is the activation semantics —
  there is no notification callback.
- **Delivery is recorded, not judged.** Each non-key message produces a
  `delivered` event naming its type and its recipients — or that the hook
  consumed it. The kit cannot know whether a pane *acted*, so it does not
  claim to. `delivered` events are coalesced per message type and
  re-recorded only when the recipient set changes, so a spinner ticking ten
  times a second holds one slot in the buffer, not all of them (§11.2).

**Blocking rule:** handlers and `Update` must not block. Long work returns
a `tea.Cmd`. Not enforced at runtime; **durations go to the debug log**
keyed by receipt `seq` (§14) — a stall is a log line, and the receipt
stream stays timestamp-free.

**Mutation rule:** domain state is mutated only in `Update` or in
handlers, on the program goroutine. Command goroutines read and compute;
their result crosses back as a `tea.Msg`. A command that mutates the store
races the frame function and voids determinism.

### 5.4 Failure handling

- A panic in a handler, `Init`, `Update`, `Render`, or a command aborts the
  program. The kit **restores the terminal, flushes the receipts with
  `result: panic`, and exits non-zero.** It does not try to continue: app
  data may be half-mutated, and rendering from half-mutated state invents
  state rather than recovering it.
- **Commands are wrapped.** Each command the kit forwards is wrapped so a
  panic in its goroutine becomes an abort message on the program goroutine
  — the same path as a synchronous panic. An app that batches commands
  batches through `tuikit.Batch`, which wraps each one; a `tea.Batch` the
  app builds itself is a single opaque command to the kit, and only its
  outer call is wrapped.
- **`Run` returns an error:** `nil` on a normal quit, `tea.ErrInterrupted`
  when bubbletea ends the program on an interrupt, or an error wrapping the
  recovered panic and its stack.
- **`AsModel` is narrower.** The app owns the `tea.Program`, so the kit
  guarantees recovery, the receipt, and a clean quit, and exposes the cause
  as `app.Err()` for the app to read after its program returns. It does not
  guarantee the exit code or the terminal: those belong to whoever runs the
  program.

### 5.5 Input and capture *(step 0)*

```
key press in (tea.KeyPressMsg)
  │
  ├─ esc / ctrl+c ─────────────► the kit — always, before anything else
  │
  ├─ a modal is on top? ── yes ─► its Keys(): a match dispatches (modal scope)
  │        no                     no match: text capture? → its Update
  │                                      else          → dropped
  ▼
focused pane Keys ──► screen Keys ──► global verbs     (first consumer wins)
  │
  ├─ still unbound ──► the host hook, which may consume it
  │
  └── every key leaves a dispatch Event — `noop` if nothing consumed it
```

| Modal on top | Matched by the modal's `Keys()` | Unmatched key |
|---|---|---|
| Normal capture (confirm) | dispatched in modal scope | dropped |
| **Text capture** (`TextCapture() true`) | dispatched in modal scope | delivered to the modal's `Update` as typed text |

**Declared keys win; the rest is text.** A filter modal binds `enter`
(apply) in `Keys()`; every other key — `?` and `:` included — reaches its
`Update`. The two capture modes differ only in what happens to an unmatched
key.

Consequences:

- **Exactly two keys answer during capture, in both modes:** `esc` and
  `ctrl+c`.
- **Help and palette never open over a modal.** `?` and `:` are ordinary
  global verbs and do not fire while anything is capturing — a confirm
  cannot be covered by a palette.
- `esc` pops exactly one level — the top modal — and touches nothing below.
- `enter` confirms nothing by default (§6.4).
- Non-key messages bypass capture: the machine keeps stepping under any
  overlay, and every open modal and pane still receives them (§5.3). Paste
  is the one exception: with a text-capture modal on top it goes only to
  that modal (§5.3).

### 5.6 bubbletea v2 facts the contract depends on

The kit targets bubbletea **v2**. These behaviors were read from the v2
source (v2.0.9) and are load-bearing:

| Fact | Consequence for the kit |
|---|---|
| Both `tea.KeyPressMsg` and `tea.KeyReleaseMsg` satisfy the `tea.KeyMsg` interface, with **identical** strings | The kit dispatches presses only. A release is dropped by type before routing — filtering by string would fire every key twice on terminals that report releases |
| Key spelling is `Key.String()`: `ctrl+c`, `shift+tab`, `pgdown` | Catalog defaults are checked against it (§6.4); `tea.Cmd` is bubbletea's type and the kit defines no `Cmd` of its own |
| ctrl+c normally arrives as a key press; delivered as a signal instead, bubbletea v2 ends the program itself with `tea.ErrInterrupted`, before any model sees it | Either way the user exits, and nothing in the table or the hook can prevent it |
| `Model.View()` returns `tea.View`, a struct — not a string | Building it from the frame is the kit's concern, inside `Run` and `AsModel` |
| `tea.Msg` is `any` | App messages implement nothing |
| Environment messages (window size, background color, focus) are ordinary messages | They take the ordinary path (§5.3); the kit never swallows one |

## 6. Verbs

### 6.1 The Action type

```go
type Action struct {
    id          string   // identity: override matching, receipts, audit
    keys        []string // canonical key first; aliases follow ("r", "F5")
    run         Handler
    help        string   // every action has one; generated help/footer read it
    title       string   // palette/help display; empty = mechanic (hidden from palette)
    needsCursor bool     // static handler/target contract; audited
    danger      bool     // compiles to a confirm
    override    bool     // permits replacing a kit default or cross-scope shadowing
}

type Bindings = []Action

func NewAction(id string, keys []string, run Handler) Action
func (a Action) Help(text string) Action
func (a Action) Title(text string) Action
func (a Action) NeedsCursor() Action
func (a Action) Danger() Action
func (a Action) Override() Action
```

**Fields are unexported; the builder chain is the API.** Two reasons, one
of them the compiler's: Go rejects a field and a method sharing a name, so
`Help string` and `.Help(...)` cannot coexist. The other is the contract:
exported mutable fields would let an app change an action after `Keys()`
is taken, and the static table that every projection and the audit depend
on would no longer be static. Builders take a value and return a copy, so
an action is immutable the moment it is registered. Tooling reads actions
through accessors and through receipts and audit output.

- `id` and `title` are separate: mechanics carry kit IDs but no `title`;
  app actions carry both. Override matching is by `id`+scope.
- `keys` is a list because aliases are ordinary (`r` and `F5` for one
  refresh). The canonical (first) key is what help, palette, and footer
  display.
- `NeedsCursor` ⇒ pane scope, **and a pane that has a cursor.** A screen or
  global action claiming a cursor target is rejected — "kill the focused
  thing" as a global verb is exactly the anomaly this rule catches — and so
  is a cursor claim in a pane that embeds no `Cursor`: it could never fire.
- **Availability growth path (not in v0.1):** an optional pure
  `Enabled(c Ctx) (ok bool, reason string)` evaluated for footer/palette
  display and receipted on refusal. v0.1 behavior: a `NeedsCursor` action
  with no selection does not fire; the receipt records `target: none`;
  the handler never nil-checks.

### 6.2 Scope, instances, and shape *(step 0)*

An action's scope is where it is registered: a modal's `Keys()`, a pane's
`Keys()`, a screen's `.Keys(...)` builder, or `tuikit.Global(...)` passed to
`NewApp`. The registration site is self-verifying; a scope *field* would be
a claim needing its own audit rule, and centralized declaration would break
pane reuse: a pane must carry its own verbs.

**`Keys()` is called once per instance, at creation** — a root's panes at
wire-up, a detail's when pushed, a modal's when opened. Handlers are method
values bound to one instance: `p.kill` acts on *that* `p`. A table cached
from some other instance — the one the audit built — would act on the wrong
pane.

**Shape.** What the audit records from a factory is not a table but its
*shape*: per variant, per pane name, the ordered `(id, keys, NeedsCursor,
Danger)` of declared actions, which mechanics discovery grants each pane
(§6.4), and the screen's or modal's own keys. Every later instance must
declare the same shape. A difference is fatal at wire-up, and a refused
push or refused open at runtime. Help and title are display text and may
differ.

Why so strict: a screen's verbs are a function of its type, not of its
parameters. A detail screen whose keys depend on whether it shows a job or
a container is two screens — register two factories, and the audit sees
both.

Effects (`Push`, `OpenModal`, …) are statements and return nothing; a
handler's only return value is its `tea.Cmd`.

### 6.3 Reserved verbs — the pinned class

| ID | Key | Effect |
|---|---|---|
| `GLOBAL_UNWIND` | `esc` | pops the innermost pushable (modal stack first, then screen stack); receipt records which |
| `GLOBAL_HELP` | `?` | generated help |
| `GLOBAL_PALETTE` | `:` | generated palette |
| `GLOBAL_QUIT` | `ctrl+c` | quit — pinned (liveness: every state has a guaranteed exit *and* a guaranteed kill) |

**Why pinned, never rekeyable:** they are promises to the app's users, not
config for the author; capture liveness holds by construction ("you never
audit what cannot be misconfigured"); capture is implemented *through*
them, so the dispatcher knows statically what `esc` does; the pinned exit
forces honest mode modeling — nuance is expressed by pushing a level, never
by rekeying the exit. They are also handled before any app code runs
(§5.3). Enforced by one audit rule: **binding `esc`, `?`, `:`, or `ctrl+c`
anywhere is a violation.**

`esc` is a mechanism, not a binding: it appears in no catalog entry, so
"dismiss with `esc`" is never a key an app or even the kit's own confirm
grammar can claim.

Nothing else is pinned. Digits, `tab`, and the mechanic keys are kit
defaults — overridable, but shadowing them is checked (§6.4).

### 6.4 The kit catalog

Guardrail: the kit ships actions only for machinery it owns (cursor, scroll,
stacks, palette, help, confirm). It never ships data verbs — no `refresh`,
no `delete` — because it has no store.

| Scope | ID | Default key | Class |
|---|---|---|---|
| global | `GLOBAL_UNWIND` / `GLOBAL_HELP` / `GLOBAL_PALETTE` / `GLOBAL_QUIT` | `esc` / `?` / `:` / `ctrl+c` | **pinned** |
| screen | `GOTO_<name>` (one per root) | digits `1`–`9` in registration order | overridable; digits are data (palette lists, footer hints, audit sees) |
| pane with a cursor | `CURSOR_NEXT` / `CURSOR_PREV` / `CURSOR_FIRST` / `CURSOR_LAST` | `j` / `k` / `g` / `G` | mechanic |
| pane with a cursor | `CURSOR_PAGE_UP` / `CURSOR_PAGE_DOWN` | `pgup` / `pgdown` | mechanic |
| pane with a scroll | `SCROLL_UP` / `SCROLL_DOWN` | `ctrl+u` / `ctrl+d` (half-page) | mechanic |
| pane with a scroll | `SCROLL_TOP` / `SCROLL_BOTTOM` | `home` / `end` | mechanic |
| every pane | `FOCUS_NEXT` / `FOCUS_PREV` | `tab` / `shift+tab` | mechanic |
| the palette | `PALETTE_UP` / `PALETTE_DOWN` | `up` / `down` | **fixed in v0.1** |
| the palette | `PALETTE_SELECT` | `enter` | **fixed in v0.1** |
| help | `HELP_UP` / `HELP_DOWN` | `up` / `down` | **fixed in v0.1** |
| help | `HELP_PAGE_UP` / `HELP_PAGE_DOWN` | `pgup` / `pgdown` | **fixed in v0.1** |
| confirm | `CONFIRM_YES` / `CONFIRM_NO` | `y` / `n` | **fixed in v0.1** |

Every ID has exactly one default key. The `Default key` column is checked
against real v2 spellings (§5.6); the rest of this section says who
receives which mechanic.

**Mechanics are granted by discovery — what a pane embeds decides:**

| The pane… | Discovered through | Receives |
|---|---|---|
| embeds `Cursor` | the promoted `KitCursor() *Cursor` | `CURSOR_*` |
| embeds `Scroll` | the promoted `KitScroll() *Scroll` | `SCROLL_*` |
| embeds both | — | audit Error: two scroll owners (§9.5) |
| is any leaf | — | `FOCUS_*` |

Cursor panes scroll through the cursor, so they do not get `SCROLL_*`;
scroll panes have no selection, so they do not get `CURSOR_*`.

**Four rules govern rebinding and shadowing:**

1. **Rebinding a kit default's keys is explicit by construction.** Mechanics
   are active with their default keys without any registration — they are
   not opt-in. `Mech(id, keys...)` *rebinds* one in its own scope; writing
   `Mech(id, defaultKey)` is a no-op. The rebinding sits at the
   registration site, so the intent is in the code. Rebinding a mechanic the
   pane does not receive is an audit Error.
2. **Replacing a kit default's behavior needs `.Override()`.** An app
   action whose `id`+scope matches a catalog entry must carry it.
3. **Cross-scope shadowing needs `.Override()`.** A pane that binds `tab`,
   a digit, or any key another scope (screen, global, or a kit mechanic)
   already binds is an audit Error without it. First-consumer-wins remains
   the resolution rule; the audit only insists that every cross-scope
   duplicate is intentional. Within one scope, declaration order is
   deliberate precedence — legal, reported as a Warning.
4. **A duplicate `id`+scope among the app's own actions is a violation.**

**The kit's own modals have keys, and in v0.1 they are fixed.** An app has
no registration site at which it could rebind the palette, help, or
confirm, so the catalog does not advertise them as overridable:

- *Palette* — a text-capture modal. `up`/`down` move the highlight and the
  list follows; `enter` selects; every other key is filter text (§11.1).
- *Help* — a normal-capture modal. `up`/`down` scroll a line and
  `pgup`/`pgdown` a page; every other key is dropped.
- *Confirm* — `y` and `n`. Not `enter`: it is the drill-in key on most list
  screens, and a double-enter reflex on a destructive row would confirm it.
  Not `esc`: esc is unwind and cannot be bound, so the kit's own catalog
  obeys the rule it enforces on apps.

Drill-down (`enter` opens detail) is deliberately not kit-provided — what
"open" means is app semantics.

### 6.5 Target safety *(step 0)*

**Resolve fresh.** Immediately before any `NeedsCursor` dispatch — a
keypress, a palette selection, or `CONFIRM_YES` — the kit calls the
originating pane's `ItemIDs()` and computes the target from that, whether
or not the pane is visible. It never reads a declaration stored by an
earlier sync: a hidden pane stops syncing (§5.2), and a confirm can outlive
the frame in which it was opened.

1. At the trigger — the keypress, or the palette *selection*; one path per
   the one-resolver rule — the kit resolves the target fresh. The captured
   `Target` carries the **originating pane's** structural identity.
2. An ordinary action runs at once with that target.
3. A `Danger` action opens the confirm, which **displays the target**
   ("kill job-123?").
4. On `CONFIRM_YES` the kit resolves again, fresh, from the originating
   pane: it recomputes `Index`, refuses a duplicated ID, and aborts with
   `result: target-gone` if the ID is gone. Both outcomes carry the push's
   correlation ID (§11.2).
5. The kit guarantees the target existed when it was shown and exists when
   the handler runs. Domain truth at execution time remains the handler's
   re-check — a documented convention.

**Palette timing.** NeedsCursor entries show the focused pane's *current*
target. The hint is derived on every render and never stored, so it cannot
go stale while the palette is open; selection then resolves at that moment,
exactly as the keypress would. The kit never executes a snapshot taken
earlier: a selection must do what the identical keypress does, which is the
one-resolver invariant itself.

## 7. Stacks and navigation

### 7.1 Two pushable stacks

- **Screen stack** — drill-down instances; they replace the viewport and
  capture nothing extra; `esc` pops. Each entry is a fresh instance from
  `Push` (§4.3).
- **Modal stack** — capturing overlays; `esc` pops; the screen stays
  rendered underneath.

`esc` pops the innermost pushable (modal first, then screen). Inside a
derived confirm modal, popping *is* the dismissal; `CONFIRM_NO` remains the
explicit deny key. No depth limits are enforced: deep chains are a design
smell the receipts make visible, not a wiring error.

**Confirm is a derived modal** — `Danger` actions compile to opening the
kit's `confirm` modal; confirm flows are never hand-wired. The palette is
itself a text-capture modal: it grabs input while open, and can never be
opened over another modal (§5.5).

### 7.2 Effects API *(step 0)*
Handlers manipulate navigation through typed `Ctx` methods — every one is
receipt-recorded:

```go
c.Push(name, params)      // instantiate registered screen `name`, push it
c.Pop()                   // pop the caller's own screen off the stack
c.Switch(name)            // show root `name` as the only entry on the stack
c.Focus(paneName)         // move focus within the current screen
c.OpenModal(name, params) // instantiate registered modal `name`, push it
c.CloseModal(value)       // the calling modal closes itself
c.Quit()                  // exit the program
c.Notice(text)            // persistent one-line result area (a receipt projection)
```

**Effects belong to the caller.** A stack or focus effect is accepted only
from the component that owns input right now, and it acts on that
component's own context. A background pane, or a modal sitting under
another, cannot close, pop, or switch something it does not own: the effect
is refused and receipted, and nothing moves.

| Caller | Is current when… |
|---|---|
| a pane | its screen is the top of the screen stack and no modal is on top |
| a modal | it is the top of the modal stack |
| the host hook | always — it is app-level |

| Effect | Accepted from | Acts on |
|---|---|---|
| `Push`, `Switch` | a current pane, the top modal, the host | the screen stack — subject to `Reaches` (§7.3) |
| `Pop` | a current pane, the host | a pane pops its own screen; the host pops the top. A root is never popped |
| `Focus` | a current pane, the host | focus within the current screen |
| `OpenModal` | a current pane, the top modal, the host | the modal stack; the caller becomes the opener (§7.5) |
| `CloseModal` | the top modal only | the calling modal — nothing else |
| `Quit`, `Notice` | any live caller | the program, the notice line — they act on no stack, so ownership does not apply |

- A key handler is always current: keys reach only the component that owns
  input. Ownership therefore matters for `Init` and `Update`, which every
  live instance receives.
- Effects are statements: they return nothing, and are queued until fan-out
  ends (§5.3).
- `Push` and `OpenModal` take **names, never instances**. Params are opaque
  to the kit (`any`) and are the factory's to interpret; the audit
  constructs every factory with `nil`, so a factory must build a valid
  screen or modal without them.
- `Push` and `Switch` refuse targets that are not declared (§7.3).

### 7.3 Navigation is declared *(step 0)*
- A screen's `.Reaches(names...)` covers every action registered in that
  screen — its panes' and its own. `tuikit.Reaches(...)` passed to `NewApp`
  covers global actions and the host hook.
- **`Push` and `Switch` refuse, at runtime, any target that is unregistered
  or undeclared** for the caller: a receipted refusal, and nothing moves.
- **The rule follows the caller, whatever invoked it.** A handler, `Init`,
  and `Update` are all calls from a component, and each is checked against
  that component's declared set.
- **A modal inherits its opener's set**: opened by a pane, that pane's
  screen's `Reaches`; by another modal, that modal's inherited set; by the
  host, the app-level set. Modals declare nothing of their own.
- **The kit's digit jumps are exempt.** `goto <root>` is generated, not
  declared, and is not held to a `Reaches` it could never appear in.
- Handlers are opaque to the audit, so the declaration is the only way it
  can see navigation at all — it is what makes "unreachable screen"
  computable (§12) — and the runtime refusal is what keeps the declaration
  true.

Digits address **roots only** — screens registered with `Root`. Screens
registered with `Detail` are drill-down targets and get no digit. `goto`
(a digit) **resets the screen stack** — digits mean coarsest navigation;
landing inside a drill chain contradicts that. The receipt records the pop
count.

### 7.4 Focus

- The focus ring is the flat list of leaf panes of the current screen's
  active variant, re-derived every frame.
- If the focused pane's name is absent after re-derivation, focus resets to
  the first focusable — deterministic, receipt-recorded.
- `tab`/`shift+tab` (`FOCUS_NEXT`/`FOCUS_PREV`) move focus. There is no
  esc-blur: focus is reachable and leavable symmetrically via tab; `esc`
  with empty stacks is a receipted no-op.

### 7.5 Modal contract *(step 0)*

```go
type Modal interface {
    Render(w, h int) string
    TextCapture() bool // true ⇒ unmatched keys reach Update as text (§5.5)
}

type ModalFactory func(params any) Modal

// optional, shared with panes:
//   Keyer       Keys()  — the modal's keys; matched before any text is delivered
//   Updater     Update() — hosted input, and the results of modals it opened
//   Initializer Init()  — cursor blink
```

| Concern | Contract |
|---|---|
| Registration | Modals are registered as named factories — `tuikit.RegisterModal(name, factory)` — and opened by name, so the audit can instantiate every one and check its keys and its capture mode |
| Shape | `Keys()` is called once per instance at open and must match the audited shape (§6.2) |
| Open / close | `OpenModal(name, params)` pushes. `CloseModal(value)` is the **calling modal closing itself** — only the top modal can. Both are receipted with a **correlation ID** assigned at the push |
| Result | `ModalResult{Name, Value, Canceled}` goes **only to the opener** — the current component that called `OpenModal`: a pane, a modal, or the host, from a handler, `Init`, or `Update` alike. `Canceled` is true when `esc` closed it |
| Hosted input | A modal embedding a bubbles textinput returns `TextCapture() true` and forwards messages through `Update`; the kit never interprets the text |
| Capture | One level: `esc` pops the top modal only; `?` and `:` do not fire while any modal is up |
| Lifetime | Built and initialized at open; discarded at close |
| Non-key messages | Reach every open modal and every pane regardless of capture (§5.3) |

- **Result delivery is the one exception to always-receive.** A result is a
  reply to one caller. `Name` is the registration name, so one opener can
  host several kinds of modal.
- **Who the opener is.** Whoever called `OpenModal`, recorded at the push
  (§7.2). A result whose opener is no longer live is receipted and dropped.
- **Confirm** is a derived modal the kit registers under the name
  `confirm`. `Danger` compiles to `OpenModal("confirm", …)` carrying the
  action and its captured target (§6.5); `CONFIRM_YES` and `CONFIRM_NO` are
  its keys; the opener of a derived confirm is the kit itself.

## 8. Lifecycle — ratified rules

| Scenario | Ratified behavior |
|---|---|
| Screen removed while on the stack | Unrepresentable: the registry is static and the only route onto the stack is `Push(name, params)` |
| `Switch` to a root | Shows the persistent root instance — its panes keep their state — as the stack's only entry |
| Detail screen pushed | The factory runs with the params, the shape is checked, its panes run `Init`; then it is on the stack |
| Detail screen popped | Discarded: its panes leave fan-out and are never reused |
| Pane's backing data replaced | Declare handshake re-validates each sync; cursor re-anchors to nearest live ID |
| Confirm's target vanishes before `y` | `target-gone`; the action aborts. The recheck is a fresh `ItemIDs()` call on the originating pane, visible or not (§6.5) |
| Action switches screens, pushes, or opens a modal | Queued until fan-out ends, then applied; the receipt records the transition |
| Action changes which layout variant is selected | Selector re-evaluated next frame; focus re-derives; bindings unchanged — every variant was audited (§9.3) |
| A pane is hidden by the newly selected variant | Keeps its cursor and scroll (state identity is the instance); it stops syncing until visible again |
| Focus points at a nonexistent pane | Focus resets to first focusable, deterministically, receipted |
| Zero screens | Audit error at startup — fatal |
| Registration order changes | Digits shift with order; digits are derived data and should track the registry |
| Same pane instance on two screens | Shared cursor state — a documented feature; it receives each message once |
| Same screen type pushed twice | Two instances, two states; each instance's handlers act on it alone |
| A modal closes | Its opener — and only its opener — receives the `ModalResult`; `Canceled` is true if `esc` closed it |
| Palette or help opened over a modal | **Refused**: only `esc`/`ctrl+c` answer during capture, so `:` and `?` cannot fire while any modal is up (§5.5) |
| Data changes while the palette is open | The hint shows the current target on every frame; selection resolves at that moment (§6.5) |
| App resized while a modal is up | Geometry is applied first, capture notwithstanding; the TooSmall gate evaluates against the current screen's `Min` |
| Duplicate item IDs in a declaration | `sync` event + audit violation; `NeedsCursor` actions refuse the duplicated ID |
| Host hook returns `true` for `esc`, `ctrl+c`, or a resize | No effect: it never sees the first two and cannot consume the third (§5.3) |
| Non-key message arrives | Recorded as `delivered` — the recipients, or that the hook consumed it; re-recorded only when the recipients change |
| Key release arrives | Dropped before routing: no dispatch, no event |
| `Push` or `Switch` names an unregistered or undeclared target | Receipted runtime refusal; nothing moves. An unregistered name is also an audit Error (§7.3) |
| An instance declares a different shape than its audited factory | Fatal at wire-up; a refused push or open at runtime (§6.2) |
| The selector answers a variant that is not declared | An audit Error if it is the selector's first answer; at runtime a receipted event, and the first variant is shown |
| A background pane, or a modal under another, calls a stack or focus effect from `Update` | Refused and receipted; nothing moves (§7.2) |
| Anything but the top modal calls `CloseModal` | Refused and receipted (§7.2) |
| A modal calls `Push` or `Switch` | Accepted; checked against the opener's inherited `Reaches` (§7.3) |
| A paste arrives while a text-capture modal is on top | Delivered to that modal alone — not the host, not any pane (§5.3) |
| A palette entry is selected | The palette closes first; the entry dispatches with the original pane as caller. Refused if that pane is no longer current (§11.1) |
| Panic in a handler, `Init`, `Update`, `Render`, or a wrapped command | Terminal restored, receipts flushed, exit non-zero; `Run` returns the cause (§5.4) |

## 9. Geometry

### 9.1 Constraints
**Directions.** `Row` stacks its children top to bottom — heights add.
`Cols` places its children side by side — widths add. Either may nest.

**Sizes.** A constraint sizes a node's **outer box, border included**. A
pane's `.Min(w, h)` is the smallest **slot** (content rectangle) it
accepts, and a slot is the box minus its border (§9.2) — so `Fixed(42)`
shows 40 columns, and a pane promising 24 needs a box of 26.

| Constraint | Takes |
|---|---|
| `Fixed(n)` | exactly `n` cells along the parent's axis |
| `Ratio(a, b)` | `a/b` of the space left after `Fixed` |
| `Fill` | a share of whatever is left — **the default** for a node with no constraint |

Distribution along the parent's axis, in this order:

1. `Fixed` takes exactly its size.
2. Each `Ratio(a, b)` takes its floor share of what is left after `Fixed`;
   leftover whole cells go to the largest fractional part, the earlier
   declaration first (largest-remainder).
3. `Fill` nodes split the remainder **equally**; a cell that cannot be
   split goes to the earliest `Fill`.

The arithmetic is relative and deterministic; there is no shrink
negotiation and no overflow resolution, ever. `Fixed` sizes that exceed
the space are an audit Error. If the terminal is smaller than a screen's
promise, the TooSmall gate fires — the only runtime size gate.

### 9.2 Slot semantics and chrome costs
```go
type Slot struct {
    W, H    int    // the CONTENT area — chrome (borders, titles) is outside
    Focused bool
    Theme   Theme  // environment fact, not injected state
}
```

The layout deals slots; panes draw their slot; the kit draws chrome.
`Render` returns one string per slot row, each exactly `W` cells wide, and
exactly `H` rows. Output is **clipped and padded to the slot by cell
width** — kit-owned and deterministic, so an overflowing pane cannot shift
its neighbors. A pane that overflows habitually is recorded as an audit
Info (§12).

Chrome costs are constants, so geometry can be computed by hand and checked
by the audit:

| Chrome | Cost | When |
|---|---|---|
| Pane border | 2 rows + 2 columns | every leaf pane |
| Group border | 2 rows + 2 columns | a group that carries a `Title` |
| Title text | 0 | it is drawn in its node's top border |
| Footer | 1 row | every screen, always |
| Notice line | 1 row | every screen, always reserved — empty when there is no notice |
| Help, palette, confirm, any modal | 0 | overlays cover; they do not push |

A leaf's slot is its box minus the border. The layout tree gets the screen
minus the footer and the notice line.

### 9.3 Layouts may depend on state — through declared variants *(step 0)*

A screen declares a **finite set of named layout variants**; a per-frame
selector picks one as a function of application state (two-pane →
three-pane on a keypress is legal). What is refused is a tree built ad
hoc: a pane that exists only inside a hand-built tree is invisible to the
audit and to `Keys()`.

```go
return tuikit.NewScreen("jobs", tuikit.Min(100, 20),
    tuikit.Variant("plain",    plainLayout),
    tuikit.Variant("with-log", logLayout),
).Select(func() string { return ui.layout }) // closes over app state
```

Invariants that make this safe:

- the audit resolves **every variant** — geometry, pane-name uniqueness,
  pane minimums, and the screen's honest minimum (the smallest size at
  which *all* variants solve)
- `Keys()` is taken for the union of panes across variants; bindings are
  static — layout choice may be dynamic, behavior may not
- **one pane, one instance, in every variant.** Panes are constructed once
  per factory call and referenced from each variant that shows them
  (§10.1). A second `&QueuePane{}` is a second cursor, so switching
  variants would silently reset the selection; the audit rejects the same
  pane name bound to different instances within a screen
- **the selector's first answer is audited.** The audit calls it once and
  requires a declared name; at runtime an unknown name is receipted and
  the first variant is shown
- the focus ring re-derives per frame with deterministic reset (§7.4)

### 9.4 Minimums — checked by the real solver, at every size

Minimums are audit-time promises: **screens declare `Min(w, h)`, panes
declare `.Min(w, h)`**. Pane minimums are never runtime layout inputs.

The audit runs the actual constraint solver at **every size from the
smallest up to a bound** (480×200 by default, configurable) and checks
every leaf at each. A failure at or above the declared `Min` is an Error
naming it: "at 40×14 in variant `plain` the `queue` pane gets 18×10; it
promised 24×10." A failure *below* `Min` is not an error — those
failures are what the honest minimum is made of.

A sweep is needed because a check at `Min` alone is unsound twice over. A
fold over declared minimums misses `Ratio(1,10)` starving a leaf at width
100. And largest-remainder rounding makes solvability **non-monotonic**: a
size can fail above a size that passes, so passing at `Min` says nothing
about any larger terminal.

The **honest minimum** is the smallest size such that every size at or
above it, up to the bound, passes. The audit reports it next to the
promise: "your screen's honest minimum is 108×22; you promised 80×20."
`Min` stops being a hand-tuned guess and becomes a checked claim. Beyond
the bound nothing is claimed. `Fixed(n)` implies a minimum of `n` — no
double-declaring. Chrome costs (a title costs its rows) are kit-known
constants.

### 9.5 Scrolling — anchored, two modes, no hooks

Panes are scrollable by default; the kit owns the mechanism and the edge
cases. Scroll state is the cursor's philosophy applied once more —
pointers, never contents, synced by the declare handshake:

```go
type Scroll struct {
    Mode   Mode   // ScrollItem (the zero value) | ScrollLine — set at declaration
    anchor string // item mode: ID of the top-visible row
    line   int    // line mode: first visible line
    pinned bool   // follows the bottom; set only by tail intent
}

func (s *Scroll) ScrollTail() // opt-in: start pinned at the bottom
```

| Mode | Anchored by | Declared by | For |
|---|---|---|---|
| Item | item ID | `ItemIDs()` | list panes — identity survives inserts and removals |
| Line | line number | `LineCount(width)` | logs, detail views — append-only text |

The mode is an **explicit, exported field chosen where the pane is
constructed** and never changed afterward. The zero value is item mode, the
common case; a log pane writes `tuikit.Scroll{Mode: tuikit.ScrollLine}`.
The other fields stay unexported: an empty item list must not silently
turn an item-mode pane into line mode, and nothing outside the kit may
write the anchor or the pin.

Transition table — the whole machine:

| Event | Transition |
|---|---|
| sync (`ItemIDs` / `LineCount`) | pinned → chase the new bottom; else re-anchor/clamp in place. **The result is stored here** |
| `SCROLL_DOWN` / `SCROLL_UP` | line ± half-page, clamp; `pinned` set at the bottom, cleared elsewhere |
| `SCROLL_TOP` / `SCROLL_BOTTOM` | line = 0 / max; `pinned = false / true` |
| `Offset()` | pure read of the value sync stored — the only function render calls |

What this absorbs, so pane authors never reinvent it: boundary clamps and
idempotent no-ops; tail-follow only while `pinned` (growth chases the
bottom for a pinned viewport, preserves an unpinned user's place);
re-anchoring on removal by last known index; pure-clamp resize handling;
empty and shorter-than-viewport content.

Rules:

- **Pinning is intent, not inference.** `pinned` is set by `ScrollTail()`,
  `SCROLL_BOTTOM`, or scrolling to the bottom — never by content merely
  being shorter than the viewport.
- **Line mode is append-stable only.** Line anchors are positions, not
  identities: head-trims and mid-content rewrites renumber lines and drift
  the viewport. Mutating content belongs in item mode — a rotating log's
  lines have natural IDs (sequence numbers).
- **Ownership is discovery.** A pane that embeds `Cursor` scrolls through
  the cursor (it owns the offset, which follows the selection); a pane that
  embeds `Scroll` owns a `Scroll`; a pane that embeds both is an audit
  Error. Go surfaces the same rule late — the first `Offset` call is an
  *ambiguous selector* — but only if the pane makes it; the audit catches
  the embedding.
- **Follow needs the body, not the slot.** A pane that draws a header, a
  footer, or multi-line rows must say so, or the selection scrolls out of
  view while the kit believes it is on screen:

```go
// optional — defaults: Reserved() = 0, RowHeight() = 1
type Metrics interface {
    Reserved() int  // rows that are not item rows: header, footer, gaps
    RowHeight() int // cells one item occupies
}
```

  The kit computes body rows as `Slot.H − Reserved()` and visible items as
  body rows over `RowHeight()`; cursor follow and scroll clamping use
  those, and `Ctx.ViewH` is the visible item count. The kit cannot see
  rendering, so an undeclared header is a lie it cannot check: a pane draws
  a header or multi-line rows **only** by declaring `Metrics`, or not at
  all.
- `LineCount` totals must be *rendered* heights — post-wrap — which is why
  the declaration takes the slot's width (§5.2).
- Horizontal scrolling is refused (§15.3): width overflow is the pane's
  truncate-or-wrap business.

## 10. API surface

### 10.1 Day one — one screen, two variants, four verbs *(step 0)*
```go
var ui = struct{ layout string }{layout: "plain"} // its first answer must name a declared variant

// A registry factory. The audit calls it with nil params and records the
// declared shape; every later instance must declare the same shape.
func newJobs(p any) *tuikit.Screen {
    // Once per call, and the same instance in every variant — a second
    // &QueuePane{} would be a second cursor.
    queue := &QueuePane{store: store}
    detail := &DetailPane{}
    logPane := &LogPane{Scroll: tuikit.Scroll{Mode: tuikit.ScrollLine}}

    screenKeys := tuikit.Bind( // screen scope — no cursor claims
        tuikit.NewAction("layout", []string{"l"}, toggleLog).Help("toggle log pane"),
    )
    return tuikit.NewScreen("jobs", tuikit.Min(51, 14), // the audit's honest minimum
        // Cols places its children side by side; Row stacks them top to bottom.
        tuikit.Variant("plain", tuikit.Cols(
            tuikit.Leaf("queue", queue).Fill().Min(24, 10),
            tuikit.Leaf("detail", detail).Fill(),
        )),
        tuikit.Variant("with-log", tuikit.Cols(
            tuikit.Leaf("queue", queue).Fill().Min(24, 10),
            tuikit.Row( // the log appears under the detail
                tuikit.Leaf("detail", detail).Fill(),
                tuikit.Leaf("log", logPane).Fill().Min(20, 4),
            ).Fill(),
        )),
    ).Keys(screenKeys).Reaches("job-detail").Select(func() string { return ui.layout })
}

func toggleLog(c tuikit.Ctx) tea.Cmd { // switches the variant; that is all
    if ui.layout == "with-log" {
        ui.layout = "plain"
    } else {
        ui.layout = "with-log"
    }
    return nil
}

func doRefresh(c tuikit.Ctx) tea.Cmd { // long work is a command, never a block
    return store.Reload
}

// A host hook is optional; this one only observes.
type hook struct{}

func (h *hook) Handle(c tuikit.Ctx, msg tea.Msg) (tea.Cmd, bool) {
    return nil, false // false ⇒ every pane still receives the message
}

func main() {
    receipts, err := os.Create("receipts.log") // never the terminal: it would corrupt the screen
    if err != nil {
        log.Fatal(err)
    }
    defer receipts.Close()

    app := tuikit.NewApp(
        tuikit.Root("jobs", newJobs),              // digit 1
        tuikit.Detail("job-detail", newJobDetail), // drill-down target, no digit
        tuikit.Global(
            tuikit.NewAction("refresh", []string{"r", "F5"}, doRefresh).
                Help("reload from the API"), // claims no cursor
        ),
        tuikit.WithHost(&hook{}),
        tuikit.ReceiptsTo(receipts), // optional: stream events to a file
    )
    if err := tuikit.Run(app); err != nil {
        log.Fatal(err) // an audit Error, a recovered panic, or an interrupt
    }
}
```

Four verbs, each in the only scope its contract allows: `refresh` is
global because it claims no cursor; `layout` is screen scope; `drill` and
`kill` read the cursor and therefore live in the queue pane's `Keys()`
(§10.2). The app's own code it refers to but does not show — `Store`,
`DetailPane`, `LogPane`, `newJobDetail` — is ordinary app code.

**The example passes its own audit, and that is checked rather than
asserted.** The CI job extracts this program, compiles it, and runs the
audit rules that need no runtime on it; the report must carry no Error and
no Warning. That is what fixes `Min(51, 14)`: it is the honest minimum the
audit reports for these two variants, so the promise is neither broken nor
over-stated (§9.4).

Three details of the example are each a rule in action:

- `Row` stacks and `Cols` places side by side (§9.1), so `plain` is a list
  beside a detail, and `with-log` adds a log *under* the detail without
  moving either.
- The `Fill`s share width and height equally (§9.1), and the queue's
  `.Min(24, 10)` is what sets the honest minimum.
- Receipts go to a file. Written to the terminal, they would corrupt the
  screen being drawn (§14).

**Naming rules, each forced by the compiler:**

- A package function and a type never share a name. The types are the
  vocabulary (`Screen`, `Pane`, `Action`, `Layout`, `Slot`, `Cursor`,
  `Modal`, `Theme`, `Event`, `Violation`, `Ctx`, `App`).
- **`Leaf` places a `Pane`; `Pane` is what the app implements.** One name
  cannot be both the interface a pane struct satisfies and the constructor
  that puts it in the tree, so `Leaf(name, pane)` is the constructor.
- `Leaf`, `Variant`, `Root`, `Detail`, `Global`, and the other options
  return placement values apps never name: they are chained and passed on,
  so their types are unexported.
- Options are typed, not variadic `any`: `NewScreen(name, min, variants...)`
  with `.Select`/`.Keys`/`.Reaches` builders; `NewApp` takes options only.
- `Variant` takes a tree — a single `Leaf` or a `Layout` — so a one-pane
  screen needs no wrapping group.

### 10.2 Pane wiring
```go
type QueuePane struct {
    tuikit.Cursor // ① embed: the pane's selection — and its discovery
    store         *Store
}

// ② declarations — read by the sync phase, never by Render:
func (p *QueuePane) ItemIDs() []string { return p.store.Open() }

func (p *QueuePane) Render(s tuikit.Slot) string { // ③ read-only
    ids := p.store.Open()
    rows := []string{}
    for i := p.Cursor.Offset(); i < len(ids) && len(rows) < s.H; i++ { // the window sync stored
        line := "  " + ids[i]
        if s.Focused && ids[i] == p.Cursor.Selected() {
            line = s.Selected("> " + ids[i])
        }
        rows = append(rows, line)
    }
    return strings.Join(rows, "\n") // the kit clips and pads to the slot
}

func (p *QueuePane) Keys() tuikit.Bindings { // called once per instance
    return tuikit.Bind(
        tuikit.NewAction("drill", []string{"enter"}, p.drill).
            Help("open job detail").NeedsCursor(),
        tuikit.NewAction("kill", []string{"K"}, p.kill).
            Help("kill job").Danger().NeedsCursor(),
    )
}

func (p *QueuePane) drill(c tuikit.Ctx) tea.Cmd {
    c.Push("job-detail", c.Target.ID) // Target is resolved at dispatch
    return nil
}

func (p *QueuePane) kill(c tuikit.Ctx) tea.Cmd { // ④ identity, not index
    p.store.Kill(c.Target.ID)
    return nil
}
```

- **Embed is the wiring.** `Cursor` carries the promoted `KitCursor()
  *Cursor` accessor, so the kit discovers this pane as a cursor pane and
  grants it `CURSOR_*`, with zero methods written by the app. A method
  named `Cursor` could not be discovered through an embedded `Cursor`
  field — Go field shadowing — hence the distinct name.
- `j`/`k`, `g`/`G`, and `tab` need no registration: mechanics are active by
  default. `kill` is `Danger`, so pressing `K` opens the kit's confirm
  showing the target; `y` runs the handler (§6.5).
- **`Render` windows from `p.Cursor.Offset()`** — the first visible row,
  stored by sync (§9.5) — so a list longer than the slot keeps the
  selection in view instead of clipping it off.
- Handlers receive one context:

```go
type Ctx struct {
    App    *App
    ViewH  int      // focused pane's visible item rows (§9.5)
    Target Target   // {ID string; Index int; Pane string} — valid iff NeedsCursor
}
```

- Edge rules: `NeedsCursor` with no selection does not fire — the receipt
  records `target: none`. `Target.Index` is computed at dispatch, never
  carried over from an earlier frame.

### 10.3 Theming — four roles, made of lipgloss

```go
type Theme struct {
    Border   lipgloss.Style // pane outline + modal boxes
    Title    lipgloss.Style // title text
    Chrome   lipgloss.Style // footer, help, palette surfaces
    Selected lipgloss.Style // highlighted item fg/bg
}
```

The kit themes exactly what it draws; there is no theming *system* (no
dark/light variants, no profile detection, no theme files). The theme is
an **environment fact delivered in `Slot`** — never package state, never
late injection. The highlighted item is drawn by panes through the slot
primitive `s.Selected(line)`; panes that hand-roll lipgloss keep full
control (the escape hatch). Color-profile degradation comes free from
lipgloss. Golden tests pin the default theme.

### 10.4 Types

`App, Screen, Layout, Slot, Pane, Action, Ctx, Cursor, Modal, Theme,
Event, Violation` — plus the `Bindings` alias, `Target`, `Size`, `Mode`,
`ModalResult`, and the optional interfaces (`Host`, `Keyer`, `Initializer`,
`Updater`, `ItemSource`, `LineSource`, `Metrics` — shared by panes and
modals). The **budget is concepts, not types**: a day-one user learns ~10
concepts to build the day-one app (Screen, Pane, Row/Cols, Fill/Fixed,
Action, Min, keys). The full public type list is larger and that is fine;
concept load at first use is the metric. (`Event` is the record type; "the
receipt stream" is what the stream of them is called.)

### 10.5 Escape hatches — designed, not asserted

| Layer | Hatch | What it bypasses |
|---|---|---|
| Program | `tuikit.AsModel(app) tea.Model` — the composed kit as a bubbletea model the app runs inside its own `tea.Program` | `tuikit.Run` — with a narrower failure guarantee (§5.4) |
| Messages | the host hook — sees every non-key message first and may consume it | kit delivery, for a message the app wants to own |
| Input | unbound keys reach the host hook after the table | the binding table, for keys nobody declared |
| Routing | any action replaces its handler; `Mech` rebinds a kit mechanic's keys; `.Override()` permits replacing a kit default or a cross-scope shadow | kit default behavior |
| Rendering | a pane draws its slot however it likes — lipgloss, plain strings, hosted bubbles components | the kit theme, `s.Selected` |
| Layout | any `Row`/`Cols` tree over the three constraints | presets (there are none) |
| Input, per pane | a raw-key pane interface — **not built** (§15.3) | the binding table, for keys a table *has* declared |

"Raw bubbletea is one delegation away" is made true by the first two rows.
No hatch reaches the pinned keys or geometry: they are handled first, on
purpose (§5.3). The last row is the one hatch that does not exist, and the
positioning claim (§1) is narrowed to match rather than extended past the
design.

## 11. Projections — one table, many views

The routing table is a static, enumerable data structure. Everything below
is a pure function of it (plus, for footer/debug, the live event stream):

| Projection | What it is | Guarantee |
|---|---|---|
| **Routing** | the dispatch, with a dispatch event recorded per key | the table *is* the router |
| **Dry-run** | `app.Route(key, opts...)` → an Event, without executing | runs on copies: nothing runs, moves, is stored, or recorded (§13) |
| **Help** (`?`) | generated columns per scope | derived from bindings |
| **Palette** (`:`) | type-to-filter list, keys shown inline | derived from bindings |
| **Footer** | current scope's verbs (focused pane + globals) | derived — hand-maintained footer text is the drift the kit kills |
| **Audit** | `Validate()` → `[]Violation` | walks the declared graph (§12) |
| **Receipts** | one event stream: dispatch, lifecycle, sync, delivered, debug, audit | recorded, bounded, sequence-numbered (§11.2) |
| **Debug** | the receipt stream, human-visible | same stream tests assert on (§14) |

### 11.1 The palette — one resolver, never two

The palette shows the routing table **intersected with the current context**
(focused pane + current screen + global). The invariant: **every palette
entry, when selected, executes exactly as if its key had been pressed right
now** — the receipt cannot tell the two triggers apart (except the trigger
field). This holds only structurally: palette and dispatch resolve scope
through the same function. A palette with its own narrowing logic is a lie
vector.

The palette is itself a text-capture modal (§5.5): `up`/`down` move the
highlight, `enter` selects, and every other key is filter text.
**Selection closes the palette before dispatching.** The highlighted
entry is then dispatched with the original pane — the one that was
focused when the palette opened — as caller, so it resolves `NeedsCursor`
targets exactly as the keypress would (§6.5). If that pane is no longer
current, the selection is refused.

Consequences: the palette *shrinks as you drill* — context-relevant verbs,
not a command dump. Unfocused siblings' actions do not appear (v0.1 is
strict). A future opt-in **reachable mode** may list sibling actions with
stated prerequisites ("kill container · switches to containers"),
receipting the focus move — the invariant survives because the entry states
what will happen. **Search-everything is refused** (§15): cross-screen
discovery requires a transition model that does not exist yet; revisit
after real apps demonstrate operators losing time to the strict palette.

### 11.2 Events — the receipt stream, a public contract

One stream, several event kinds; `seq` orders the timed kinds:

```
Event {
  seq     uint64    // monotonic across all kinds; never a timestamp
  kind    dispatch | lifecycle | sync | delivered | debug | audit
  screen  string    // current screen name (all kinds)
  pane    string    // focused pane name (all kinds)

  // kind = dispatch
  trigger   key | palette
  key       string                    // canonical key, if key-triggered
  action    string                    // action ID; empty for an unbound key
  scope     pane | screen | global | modal
  target    {ID, Index, Pane} | none  // when a NeedsCursor action runs
  result    executed | confirm-pushed | confirm-yes | confirm-no
          | target-none | target-gone | refused | noop | panic
  dryRun    bool                      // true only on an event Route() returns
  corr      uint64                    // links a modal push to its close

  // kind = lifecycle — stack, focus, modal open/close, resize facts
  // kind = sync     — duplicate-ID reports, re-anchors
  // kind = delivered — message type, recipients (or "consumed by host"), count
  // kind = debug    — app Debugf lines
  // kind = audit    — Validate() results at startup
}
```

The schema is a **public contract from v0.1**, on four terms:

- **Additive-only — and open.** New kinds, results, and fields may be
  added; none are repurposed or removed. The result list above is the v0.1
  set, **not a closed enum**: readers must tolerate unknown kinds and
  unknown values. (A closed enum and additive-only evolution contradict
  each other; openness is what makes the addition possible.)
- **Everything is addressable.** Every event names the current screen and
  pane; a trace without them cannot answer "which pane did that".
- **Flows are reconstructable.** `corr` links a modal push to its close;
  a confirm and its resolution share one. `dryRun` is true only on events
  `Route()` returns — the recorded stream never contains one, so a test
  can never pass for a recorded session.
- **Two timing classes.** `dispatch`, `lifecycle`, `sync`, and `audit`
  events happen on the program goroutine in message order; tests assert
  their order. `debug` and `delivered` events are produced from arbitrary
  goroutines and timers, so tests match them but never order them.

**Coalescing.** A `delivered` event is recorded when a message type is
first seen and re-recorded only when its recipient set changes — a spinner
ticking ten times a second holds one buffer slot. `debug` lines are never
coalesced; they are the app's own words. The ring buffer is bounded (~500
events) and mutex-guarded.

Durations are **not** events: stalls go to the debug log keyed by `seq`
(§14), and the stream stays timestamp-free and deterministic. Debug view,
tests, and future tooling (replay, interaction coverage, regression
generation) build on this one stream; its stability is the cheap commitment
now, the tooling the expensive one later.

## 12. Audit

`Validate()` runs at dev startup (fatal on Error) **and** in CI (red
build). Three severities — a validator that only says "here are violations"
becomes noise; Info gives the audit somewhere to put taste:

| Severity | Meaning | CI behavior |
|---|---|---|
| **Error** | structurally invalid | fail |
| **Warning** | suspicious but legal | report |
| **Info** | recommendation | optional |

Catalog (seeded; the red-team pass extends it):

| Severity | Violations |
|---|---|
| Error | reserved-key rebinding (`esc` `?` `:` `ctrl+c`) · **cross-scope shadowing without `.Override()`** (a pane taking `tab`, a digit, or another scope's key) · a kit default replaced without `.Override()` · `Mech` rebinding a mechanic the pane does not receive · `NeedsCursor` outside pane scope, or in a pane without a cursor · a `Cursor` pane that does not declare `ItemIDs()` · a `Scroll` pane that does not declare the method its mode needs (`ItemIDs()` in item mode, `LineCount(width)` in line mode) · a selector whose first answer names no declared variant · duplicate action `id`+scope · pane with two scroll owners · duplicate screen names · duplicate pane names within a screen (checked per variant) · the same pane name bound to different instances across a screen's variants · zero screens · `Reaches` naming an unregistered screen · a screen or modal factory that fails to construct with nil params · an instance whose shape differs from its audited factory (§6.2) · any failing size between `Min` and the bound (§9.4) · `Fixed` sum overflow |
| Warning | key bound twice within one scope (deliberate precedence — must be conscious) · screen unreachable (no `Reaches` path from any root) · duplicate action help text · declared `Min` above the honest minimum (over-promising) · pane minimum unrealistic against its slot |
| Info | missing `Help` text · missing palette `Title` on a named action · potentially confusing key choice · habitual render overflow (§9.2) |

**What makes the static rules computable:**

- *Shadowing* is a set comparison: the pane, screen, and global tables
  against each other and the kit catalog, so a pane quietly taking `tab` or
  a digit cannot pass; and each modal's table against the reserved keys and
  the kit's modal defaults, so no modal can trap the user or silently
  replace the confirm grammar.
- *Reachability* is a graph walk over the `Reaches` declarations. A target
  reached only inside a handler is invisible statically; the runtime
  refusal (§7.3) covers it.
- *Shape* is equality against what the audit recorded when it instantiated
  the factory (§6.2).
- *Discovery* is interface satisfaction, so the declarations a pane needs
  are checked against what it embeds (§6.4).

**What the audit does not claim.** Delivery is not audited: the kit
cannot see whether a pane acted on a message, so it records `delivered`
and stops. And the audit is a **static check of the declared graph — not
a simulator**: it makes no claims about handler semantics. Every rule above
names a state that can actually be constructed; a rule for an impossible
state is noise and is not written down.

## 13. Testing story

Headless by construction: the frame is a pure function, so tests drive keys
and assert frames with no pty.

**Semantic assertions first; golden frames second.** Structural assertions
are the primary interface:

```go
tuikittest.Validate(t, app)                     // the audit as a test gate
tuikittest.Route(app, "K")                      // dry-run → Event{dryRun: true}
tuikittest.Route(app, "K", tuikittest.At(120, 40)) // as if resized
tuikittest.Route(app, "K", tuikittest.On("jobs"), tuikittest.Focused("queue"))
tuikittest.Route(app, "enter", tuikittest.WithModal("filter"))
tuikittest.SlotOf(app, "jobs", "detail")        // resolved geometry assertion
tuikittest.Region(frame, app, "jobs", "queue")  // crop to one pane's slot
tuikittest.Receipts(app)                        // the ordered kinds, with corr IDs
tuikittest.Dump(app)                            // annotated tree, resolved rects
tuikittest.Expect(keys...).Frame(golden)        // golden — deliberate visual regressions only
```

**A dry-run mutates nothing.** It runs layout → geometry → sync on
*copies* and stops: no handler runs, no cursor or scroll state is stored,
nothing is pushed, and the event it returns is not recorded. Its options
describe the state to evaluate — size, screen, focus, modal stack — and
default to the app's current state. Headless, the default size is the
smallest that satisfies every root's `Min`, derived rather than guessed.
So `Route` can assert the exact `Target` a real keypress would resolve.

**Ordered assertions cover `dispatch`, `lifecycle`, `sync`, and `audit`
events only.** `debug` and `delivered` events come from arbitrary
goroutines and timers; tests match them, never order them (§11.2).

Golden frames are brittle (width, styling, Unicode width, lipgloss version,
color profile) — used for intentional visual regressions, never as the
default workflow. The pty test remains for the true end-to-end smoke
(exit codes, real terminal path); it is the exception, not the workflow.

## 14. Debug

- **Recording is always on**: bounded ring buffer, sequence-numbered,
  mutex-guarded.
- **v0.1 ships the stream, not the overlay.** `tuikit.ReceiptsTo(w
  io.Writer)`, passed to `NewApp`, writes every event as it is recorded.
  It is a library option: the app owns the flag and the file. Stream to a
  file or a pipe — never to the terminal the program is drawing on, which
  the events would corrupt. Because the
  stream starts at construction, startup events survive to the end — a
  dump taken at exit would only ever hold the last ~500.
- The palette-style overlay (capture, auto-follow) is deferred (§15.3). It
  brings its own key when it lands; no key is reserved for it now.
- App events join the same timeline: `app.Debugf(...)` writes a `debug`
  event to the same stream tests read. **The stream developers watch is the
  stream CI asserts on.**
- **Durations live here, not in the stream.** Dispatch and sync timings are
  logged against the `seq` they belong to — a stalled handler is a log
  line, and events stay timestamp-free.

## 15. Tiers: invariants, defaults, limitations, refusals

Naming each refusal with its one-line justification is a documentation
requirement, not decoration.

### 15.1 Invariants (hard, structural)

Single owner of terminal geometry · deterministic frames · static,
enumerable actions · one routing resolver · capture cannot leak (pinned
exits always answer) · **pinned keys and geometry are handled before any
app code** · cursors anchor to identity · domain data stays outside ·
testing and production share routing semantics · no change hooks · facts
have one owner · structure recurses, behavior doesn't.

### 15.2 Opinionated defaults (changeable by evidence)
Pinned key *choices* are invariants, and the kit's own modal keys are fixed
in v0.1 (§6.4). The rest of the default grammar is changeable: digits →
roots in registration order · theme defaults · the mechanic keys of §6.4
(`j`/`k`, `g`/`G`, `pgup`/`pgdown`, `ctrl+u`/`ctrl+d`, `home`/`end`,
`tab`/`shift+tab`) · initial scroll position (top; `ScrollTail()` opt-in).

### 15.3 Deferred limitations (revisit triggers attached)

| Limitation | Revisit when |
|---|---|
| Mouse | a concrete use case; wheel-scroll first candidate |
| Horizontal scrolling | width overflow is the pane's truncate-or-wrap business; only on demonstrated need |
| Rich editing, per-item verbs | per-item verbs break the static table; dispatch-by-type inside handlers covers the pattern |
| Palette reachable mode | operators demonstrably lose time to the strict palette |
| `Enabled(c)` availability | first migrated app hand-rolls "no selection → do nothing" more than twice |
| Raw-key pane interface | first pane that must own keys the table also declares; unbound keys already reach the host hook |
| Action `Meta` extension point | operational apps demand availability/permission/async metadata |
| Debug overlay | when the streamed events prove insufficient; it brings its own key |

### 15.4 Explicit refusals

| Word/feature | Why refused |
|---|---|
| window | OS-window collision; implies overlap |
| view | overloaded to meaninglessness |
| widget | that's bubbles' job; the kit hosts, never rebuilds |
| tab (structure) | means the tab key; implies persistence |
| region | means nothing specific to anyone |
| grid | promises CSS-grid semantics (tracks, spans) the kit doesn't have — it's a split tree |
| cell | in every TUI library means one character cell — an active trap |
| search-everything palette | no transition model exists; a filter without disclosed prerequisites is a lie vector |
| multi-select | breaks the one-cursor target contract; a second selection is an identity the table cannot see |
| key chords | complicate capture and receipts — one key, one receipt, forever |
| pane zoom/collapse | a layout-mutation flavor; variants cover the need without a new mechanism |
| change hooks | the retained-mode reflex; the declare handshake replaces them |
| dynamic routing tables | data-dependent tables collapse help, palette, and audit |
| theming system | theming is a four-role config of lipgloss styles, not an engine |
| `enter` as confirm | it is the drill-in key; the key that navigates must not also destroy |
| a `handled` return on panes | it starves later panes and cannot mean what it says — `nil` is ambiguous by nature |

## 16. Adoption ladder

Every rung **adds** value; you can stop anywhere (rungs 4–5 build on 2–3 —
that is a ladder, not a claim of independence):

1. **Kill the Model boilerplate** — `tuikit.Run(app)` wires
   `Init`/`Update`/`View`; `tuikit.AsModel(app)` gives the same wiring to
   an app that owns its own `tea.Program`. Plumbing, not the value.
2. **Add screens** — places, switching, `Min` promises.
3. **Add verbs** — actions, danger→confirm, universal verbs, **generated
   help**. This is where the visible benefit lands first.
4. **Adopt the audit** — `Validate()` at startup and in CI.
5. **Adopt headless tests** — dry-run routing, receipts, geometry
   assertions, golden frames.

Design gate for the publish examples: a developer receives visible benefit
within 10 minutes — the day-one example must demonstrate generated help and
a dry-run assertion, not just wiring.

## 17. Release plan

### 17.1 Step 0 — five thin slices, then the long road
Review is closed. Step 0 is planned as slices thin enough that each builds
exactly what its tests need; no test waits on a later step.

| Slice | Builds | Tests it unlocks |
|---|---|---|
| 0a | Message loop, pinned keys, the table, receipts, the CI extraction-and-audit job | Pinned exits, Key release |
| 0b | Layout solver, variants, sync, cursor and scroll | Resize sweep, Declarations match discovery, Unknown variant |
| 0c | Factories, shape check, effects, navigation | Two detail instances, Shape refusal, Navigation refusal, Effect ownership |
| 0d | Modals, confirm, text capture | Hidden-pane confirm, A filter with `enter`, Paste routing, Palette selection |
| 0e | `Route`, command wrapping, `Run` errors | Dry-run purity, Command panic |

| # | Step | Gate |
|---|---|---|
| 0 | **Step 0**, in the five slices above | every named test passes; **the freeze happens here** |
| 1 | Size-as-fact; vocabulary; `Run`/`AsModel`; cursor identity; sync phase — completed across real apps | existing behavior preserved |
| 2 | Red-team pass: 20–30 hard runtime scenarios (§8 is the seed); every scenario ends in a ratified rule, an audit rule, or a documented refusal | zero "undefined" outcomes |
| 3 | Router-as-query, receipts, the full audit catalog, `tuikittest` — **completed**, not built from nothing | CI audit gate live for every app |
| 4 | Footer, palette, theme — all projections of the one resolver | one-resolver rule holds |
| 5 | Receipts stream (`ReceiptsTo`) — completed | debug answerable without the overlay |
| 6 | Migrate two real consumer applications; record the metric sheet | thresholds met (§17.2) |
| 7 | Prior-art pass: clone and run the known single-table / shell libraries | positioning confirmed or revised in writing |
| 8 | One new app built by someone unfamiliar with the kit, unassisted; observe, do not guide | concept budget holds; every struggle becomes a doc or API fix |
| 9 | Documentation, examples, starter template, stable package API | **v0.1.0 tag, once** |
| 10 | Public announcement | after the tag |

Steps 7 and 8 precede the tag on purpose: positioning that has not survived
a comparison, and an API that has not survived a stranger, are hypotheses.

**Step 0 builds four programs:** the day-one example (§10.1–§10.2) as it
runs for real; an async refresh (a `tea.Cmd` round-trip reaching a pane's
`Update`); a filter input (a text-capture modal); and a drill-down (a
factory push with a target-safe danger flow).

**Step 0 passes these named tests.** Each one is a contract that was
ratified on paper and is unproven until it runs:

| Test | Slice | Proves |
|---|---|---|
| **Pinned exits** | 0a | a host hook that returns `true` for everything cannot block `esc`, `ctrl+c`, or a resize |
| **Key release** | 0a | a `tea.KeyReleaseMsg` dispatches nothing and records nothing |
| **Resize sweep** | 0b | every size from each screen's `Min` to the bound: no pane under its minimum, no 0×0 frame, cursor and scroll stay normalized |
| **Declarations match discovery** | 0b | a `Cursor` pane without `ItemIDs()`, and a `Scroll` pane without the method its mode needs, are audit Errors |
| **Unknown variant** | 0b | a selector whose first answer names no variant is an audit Error; at runtime an unknown name is receipted and the first variant is shown |
| **Two detail instances** | 0c | the same detail pushed twice: each instance's handlers and replies act on it alone |
| **Shape refusal** | 0c | a factory whose keys depend on its params: the push is refused, and the receipt says why |
| **Navigation refusal** | 0c | `Push` and `Switch` of an unregistered and of an undeclared target are refused — from a handler, from `Update`, and from a modal against its opener's inherited `Reaches`; kit digit jumps are not |
| **Effect ownership** | 0c | a background pane and a modal under another cannot `Pop`, `Focus`, `Push`, or `Switch` from `Update`; only the top modal can `CloseModal`, and it closes only itself; `Quit` and `Notice` work from anywhere |
| **Hidden-pane confirm** | 0d | a danger confirm whose originating pane is hidden by a variant switch before `y` resolves freshly: the right item, or `target-gone` — never a neighbor |
| **A filter with `enter`** | 0d | `enter` applies; `?` and `:` are typed as text; `esc` yields `ModalResult{Canceled: true}` to the opener alone |
| **Paste routing** | 0d | with a text-capture modal on top, a paste reaches that modal alone — not the host, not any pane |
| **Palette selection** | 0d | selecting an entry closes the palette first, then dispatches it with the original pane as caller; refused if that pane is no longer current |
| **Dry-run purity** | 0e | `Route` leaves cursors, scroll, stacks, and the recorded stream byte-identical |
| **Command panic** | 0e | a panic inside a batched command restores the terminal, flushes events, and surfaces from `Run` |

**The CI job** extracts §10.1–§10.2 and the `Default key` column of §6.4
from this file, compiles the program against bubbletea v2, checks the keys
against real spellings, and **runs the audit on the program**. Compiling
alone would not have caught a layout that asks for 42 rows on a 14-row
screen.

**The final patch, logged.** The contract changes only through a failing
test (D47), with one exception: the patch that closed review was applied
whole, and each of its changes is a named test above.

| Change in the final patch | Test |
|---|---|
| `Row`/`Cols` directions, constraint sizes, equal `Fill`s, chrome costs | Resize sweep |
| A selector's first answer must name a variant | Unknown variant |
| `Cursor` and `Scroll` panes must declare what their mode needs | Declarations match discovery |
| Effects belong to the caller | Effect ownership |
| `Reaches` follows the caller; modals inherit it; digit jumps are exempt | Navigation refusal |
| Paste follows capture | Paste routing |
| The palette and help have keys; the palette closes before dispatching | Palette selection |

**Open questions step 0 answers — deliberately not ratified here:**

- *Teardown.* A popped detail is discarded, but a command it started keeps
  running until it finishes or stops itself. Is "commands check their own
  liveness" enough, or does a discarded instance need a stop signal? The
  two-detail-instances test is where this shows up.
- *Delivery granularity.* `delivered` is coalesced by recipient set. If
  real debugging needs to know about individual deliveries, the test driver
  can record them without touching the stream — or the coalescing rule
  changes.

### 17.2 Tag gate — migration metrics with thresholds

The tag waits for evidence, not completeness. Per migrated application:

| Metric | Threshold |
|---|---|
| UI-infrastructure lines of code | **≥ 30% reduction**, in both applications |
| Key handling outside the table | none for navigation or focus — no hand-written router |
| Focus / navigation bugs | zero regressions; the pre-existing class is not reproducible |
| Time to add an action | one declaration — no routing, help, or footer edits |
| Time to add a screen | one file + one registration |
| Interaction test coverage | **≥ 80% of registered actions** exercised by a dry-run or receipt assertion |
| Test setup | no pty for any interaction test |
| Concept count at day one | within the ~10-concept budget (§10.4) |

**If any threshold fails, the tag slips** — the failure is fixed, or the
threshold is re-argued in writing with the measurement attached. Five
reference layouts (single pane, list/detail, three-pane, nested
sidebar + bottom, adaptive narrow/wide) are implemented in both tuikit and
plain bubbletea/lipgloss as the composition benchmark: no state lost on a
layout change, deterministic resize, and a recorded LOC / concept count /
time-to-change comparison.

### 17.3 Versioning policy

- Event stream: additive-only and open from v0.1 (§11.2).
- Public API: **v0.x carries no compatibility promise.** After the v0.1.0
  tag, breaking changes require a minor bump and a migration note; v1.0.0
  is earned only after a release cycle in which real migrations needed no
  breaking change.
- Every contract change (new reserved key, capture change, slot semantics)
  must land with its audit-rule counterpart — no silent contract changes.
- `Min` promises: a default `Min` is refused (an unaudited promise is a
  lie waiting to happen); screens always declare.

## Appendix — decision log

| # | Decision |
|---|---|
| D1 | Size is root-owned; render is a pure frame function; screens hold `Min` promises, never facts — the 0×0 bug class is made unrepresentable |
| D2 | The kit owns `Run` — a small framework with escape hatches at every layer, honestly named; no global state; data never enters the kit; `AsModel` and the host hook are the designed hatches |
| D3 | Cursors anchor to item IDs; the index derives against the live list; never panics, never drifts |
| D4 | The router is a dry-runnable query — dispatch decisions assertable without mutation |
| D5 | The audit runs at dev startup (fatal on Error) and in CI; three severities with a seeded catalog |
| D6 | All tests headless; semantic assertions first, golden frames second; pty only for end-to-end smoke |
| D7 | Vocabulary verdicts: Screen/Pane/Action/Modal/Slot; `grid` and `cell` refused; `Slot{W,H,Focused,Theme}` is the per-frame handshake |
| D8 | `Title` is legal on any layout node; chrome is exactly the title; help names scopes by it |
| D9 | Minimums are audit-time promises — screens declare `Min`, panes declare `.Min`; the real solver runs at **every** size from `Min` to a bound, because rounding makes solvability non-monotonic; the honest minimum is the smallest size above which everything passes |
| D10 | Debug is a projection of the event stream; the stream (`ReceiptsTo`) ships before the overlay |
| D11 | Layout group IDs are extrapolated from optional labels plus position; diagnostics-only, unique by construction |
| D12 | The theme is four lipgloss roles delivered in `Slot`; no package state, no late injection, no theming system; `s.Selected(line)` is the pane primitive |
| D13 | One `Action` type: a key list (aliases are ordinary), `id` separate from display `title`, `NeedsCursor` audited to pane scope and to cursor panes; fields unexported, the builder chain is the API |
| D14 | The kit catalog: pinned class (`esc` `?` `:` `ctrl+c` — liveness by construction) vs overridable defaults; mechanics are granted by discovery — cursor panes get `CURSOR_*`, scroll panes get `SCROLL_*`; `Mech` rebinds; cross-scope shadowing is an Error without `.Override()`; the kit's own modals have keys — palette, help, confirm `y`/`n` — fixed in v0.1; no key is reserved for an unbuilt overlay |
| D15 | The palette is one resolver — table ∩ context; entries execute exactly as their keypress would; it shows the current target on every frame; strict contextual v0.1; reachable mode is a future opt-in; search-everything refused |
| D16 | Anchored scroll (item mode and line mode) with an explicit, exported `Mode` chosen at declaration; pin only after tail intent; sync is the only writer and `Offset()` only reads; `Metrics` declares reserved rows and row height; discovery demands its declarations — a `Cursor` pane declares `ItemIDs()`, a `Scroll` pane the method its mode needs |
| D17 | Two identity kinds: structural (`name@screen`) vs state (the instance); the registry holds named factories and the stack holds instances created only by `Push(name, params)`; roots persist, details are built at push and discarded at pop |
| D18 | Text capture: only `esc` and `ctrl+c` are intercepted; the modal's declared keys win and every other key is delivered as text; paste follows capture — it goes only to the text-capture modal on top; bubbles inputs hosted |
| D19 | Danger-target safety: `Target` carries the originating pane; every `NeedsCursor` dispatch resolves freshly from `ItemIDs()` — key, palette selection, and `CONFIRM_YES` alike — recomputing `Index`, refusing duplicates, aborting with `target-gone` |
| D20 | Message order: pinned keys and geometry before any app code; the table before the host hook; unbound keys then the hook; non-key messages to the hook then every live instance once; every app callback receives `Ctx`; effects queue until fan-out ends; panes have no `handled`; delivery is recorded (`delivered`), never judged |
| D21 | The event stream is a public contract from v0.1 — additive-only **and open**: `kind`, screen and pane names, correlation IDs, a dry-run marker; ordered and unordered kinds; dry-run events are returned, never recorded |
| D22 | Scope is placement, never a field; `Keys()` is called once per instance at creation, and every instance of a factory must declare the audited shape; effects are statements |
| D23 | Focus re-derives every frame with deterministic reset; `tab`/`shift+tab` are the focus mechanics; esc-blur is dropped |
| D24 | Digits address roots only; `goto` resets the screen stack; details are undigit-ed |
| D25 | `ctrl+c` joins the pinned class — every state has a guaranteed exit and a guaranteed kill |
| D26 | `Slot` is the content area; overflow is clipped/padded by the kit; habitual overflow is recorded as Info |
| D27 | The type budget is replaced by a day-one concept budget (~10 concepts), measured against the compiled example |
| D28 | Tiers over ideology: invariants, opinionated defaults, deferred limitations with triggers, explicit refusals |
| D29 | Positioning: "small framework with escape hatches"; the pitch leads with verifiable wiring (audit, dry-run, receipts) |
| D30 | Versioning policy: additive-only events; contract changes require audit-rule counterparts; handlers must not block (durations are logged, never recorded as events) |
| D31 | `Action` fields are unexported and the builder chain is the API — Go rejects a field and a method sharing a name, and mutable exported fields would break the static table |
| D32 | Screens register as named factories (`Root` gets a digit, `Detail` does not); `Push(name, params)` is the only route onto the stack; the audit constructs every factory with nil params and discards the instance uninitialized |
| D33 | Layouts vary only through a finite set of named variants; a per-frame selector picks one; every variant is audited; one pane name is one pane instance across all of them; the selector's first answer must name a declared variant |
| D34 | Sync runs after geometry with resolved slots and is the writer; `LineCount` takes the slot width; a dry-run normalizes copies and stores nothing |
| D35 | Modals are registered as named factories and opened by name; `ModalResult{Name, Value, Canceled}` goes only to the opener; built at open, discarded at close; modals share the panes' optional interfaces |
| D36 | Only `esc` and `ctrl+c` answer during capture, in every mode — help and palette never open over a modal |
| D37 | A panic restores the terminal, flushes events, and exits non-zero; commands are wrapped; `Run` returns the cause; `AsModel` guarantees recovery and the receipt, not the exit code |
| D38 | Domain mutation happens only in `Update` or handlers; command goroutines are read-only and return results as messages |
| D39 | `Target.Index` is computed at dispatch, never carried over from an earlier frame |
| D40 | Navigation is declared: `Push` and `Switch` refuse unregistered and undeclared targets; screens declare `Reaches`, the app declares it for global scope; delivery is recorded, not audited; rules for unconstructible states are not written |
| D41 | Escape hatches are designed — `AsModel`, the host hook, pane-owned rendering; unbound keys reach the hook, an app-level raw-key path; the per-pane raw-key interface is not built |
| D42 | Tag **v0.1.0**, only after the migrations meet written thresholds, the prior-art pass is done, and someone unfamiliar with the kit builds an app unassisted |
| D43 | `Leaf(name, pane)` is the constructor and `Pane` stays the interface apps implement — one name cannot be both; `Leaf` and `Variant` return unexported placement types; construction takes typed options, never variadic `any` |
| D44 | The day-one program and the catalog's default keys are extracted from this document by script and compiled and checked against the real bubbletea v2 before the document changes — review missed defects this caught, and a check that ran before the last edit missed one more, so the check runs on the final text |
| D45 | Only `tea.KeyPressMsg` is dispatched; key releases are dropped by type, never by string — they print identically |
| D46 | Pinned keys and geometry precede all app code; the host hook cannot consume or intercept either |
| D47 | Review is closed; the contract changes only through a failing step-0 test — the one exception is the final patch before the implementation plan, whose every change is logged as a named test in its slice |
| D48 | `Row` stacks top to bottom and `Cols` places side by side; a constraint sizes a node's outer box, border included, and a pane's `Min` is its smallest slot; chrome costs are constants in a table; `Fill`s share equally and an indivisible cell goes to the earliest |
| D49 | Effects belong to the caller: stack and focus effects are accepted only from the component that owns input and act on its own context; `CloseModal` is the top modal closing itself; `Quit` and `Notice` act on no stack and are anyone's to call |
| D50 | `Reaches` follows the caller wherever it calls from — handlers, `Init`, and `Update`; a modal inherits its opener's set; the kit's digit jumps are exempt |
| D51 | The palette closes before it dispatches, with the original pane as caller |
| D52 | Step 0 is five thin slices, each building exactly what its tests need; the CI job audits the day-one program as well as compiling it |
