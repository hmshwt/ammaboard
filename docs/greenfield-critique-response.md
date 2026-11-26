# greenfield-critique-response.md — verdicts on the greenfield critiques

> Status: **adopted as working input** (2026-11-26). This responds to
> [greenfield-critique.md](greenfield-critique.md) (§1–§7),
> [greenfield-critique-2.md](greenfield-critique-2.md) (§8), and the three
> later reviews of the consolidated spec (§9, §10) point by point.
> Verdicts are one of **adopt** (goes into greenfield.md or the roadmap),
> **partially adopt** (the insight is right, the proposal is adjusted),
> **defer** (correct but not now, with a trigger for revisit), or **reject**
> (with the reason). Where a critique misreads the project's context, that is
> said plainly — several criticisms are aimed at goals the project does not
> currently hold. Where critique-2 supersedes a verdict from critique-1, the
> supersession is marked explicitly.

---

## 0. The context correction — read this first

The critique evaluates the kit as a startup product: it scores
"product-market fit 4/10" and "monetization 3/10", worries about adoption
ladders and business angles, and asks "what is the product after the
library?". Those are good questions for a company. They are the wrong
questions for this project, today.

**The kit's first and, for now, only customer is this repo.** It was extracted
from real usage in ammaboard's TUI (D46), it serves lazylite as its second
proof, and the publish decision is already explicitly gated:
"the publish decision waits on the kit surviving several real additions
without breaking change — no community-driven API compromises until then"
([tuikit.md](tuikit.md), Rollout). There is no product thesis to be weaker
than the architecture thesis, because there is no product thesis yet —
deliberately. AmmaBoard's beta (Dec 1 2026) is the actual product with the
hard date; the kit is infrastructure underneath it.

This reframes — but does not dismiss — several of the critique's central
concerns:

| Critique concern | Reframed for this project |
|---|---|
| "Adoption is the biggest threat" | Adoption pressure starts at the authors themselves. The adoption ladder's real test is *migration ergonomics*: does lazylite + ammaboard's TUI get simpler? That test is already scheduled (§14, step 4). |
| "PMF unproven" | Correct and unactionable pre-publication. The proxy we can act on is the migration evidence the critique also asks for. |
| "Business angles" | Noted, not acted on. Angle 3 (reliability tooling over the receipt/audit stream) is the one we agree is strategically interesting; it costs nothing to keep the receipt schema stable and everything to build early. |
| "Reposition the pitch" | Accept the *wording*; defer the *positioning* until there is something to position. |

The engineering critiques — lifecycle gaps, render purity, stack semantics,
audit severity, testing emphasis — stand on their own merit and are largely
**adopted**. That is where the value of this critique is, and it is
substantial.

---

## 1. Verdict summary

| # | Critique point | Verdict | Lands in |
|---|---|---|---|
| 1 | Sharper raison d'être ("reliable interaction layer", interaction drift) | **adopt** (wording) | greenfield.md §0/§1 |
| 2 | Three-tier taxonomy: invariants / defaults / limitations | **adopt** | greenfield.md §11 |
| 3 | Lifecycle & identity section before the tag | **adopt** — the single most valuable engineering point | greenfield.md new §3.5 |
| 4 | "Cannot lie" overclaims semantic truth | **adopt** (wording) | greenfield.md §2.3, §5 |
| 5 | Action metadata extension point | **partially adopt** | greenfield.md §7 note |
| 6 | Palette global-discovery mode | **defer** (already future opt-in in spec) | — |
| 7 | Semantic testing first, goldens second | **adopt** | greenfield.md §10, kittest docs |
| 8 | Receipt stream as strategic asset | **adopt** (schema stability now, tooling later) | decision log |
| 9 | Adoption friction / 10-minute test | **partially adopt** | roadmap |
| 10a | App as façade over private subsystems | **adopt** | implementation note |
| 10b | Render purity / Track-in-Render contract | **adopt** (formalize; no redesign) | greenfield.md §3 |
| 10c | Notify semantics | **partially adopt** (document precisely, no rename) | greenfield.md §7 |
| 10d | Stack state-machine spec + red-team pass | **adopt** — pre-tag gate | new work item |
| 10e | Audit severity model (Error/Warn/Info) | **partially adopt** (Error/Warn exist; add Info + catalog) | greenfield.md §5 |
| 11 | Missing capabilities triage | **mixed** — see §4 below | roadmap |
| 12 | Business angles | **defer** (all; Angle 3 flagged) | — |
| 13 | Competitive positioning | **defer** (parked for the publish README) | — |
| 14 | Migrate real apps before more design | **already agreed** — extend the metric set | roadmap |
| 15 | Roadmap re-phasing | **partially adopt** (migration evidence becomes a tag gate) | greenfield.md §14 |
| App. | Composition deep-dive (presets, dynamic layout, defaults) | **mixed** — see §5 below | greenfield.md §3/§6 |

---

## 2. Point-by-point responses

### 2.1 — Raison d'être (critique §1) — **adopt the wording**

The sharper statement is better than ours and costs one paragraph:

> Building a reliable operational terminal application requires repeatedly
> solving interaction correctness, navigation, discoverability, and
> testability — and the underlying framework doesn't provide a coherent
> integrated solution.

And the reposition — *"a reliable interaction layer for Bubble Tea apps; the
same declarations drive routing, contextual help, discovery, validation,
debugging, and tests; if any two disagree it is a defect in the kit"* — will
be the opening of greenfield.md §1. The phrase "the central value is the
reduction of **interaction drift**" names exactly what §5's one-table rule
was reaching for; we will use it.

The pain-hierarchy table is accepted as-is. It confirms the priorities the
spec already holds (interaction correctness > composition > rendering), and
its honest row — "basic screen composition: pain medium, solution good" — is
answered in §5 below.

### 2.2 — Over-constrained architecture (critique §2) — **adopt the taxonomy**

The critique is right that a list of individually defensible refusals can
collectively narrow the product. The fix is editorial, not architectural:
§11 currently mixes three kinds of statements under one heading of
"refused". It will be split:

1. **Invariants** — hard, structural, non-negotiable: one owner of size,
   deterministic frames, static enumerable actions, one resolver, capture
   cannot leak, identity-anchored cursors, data stays outside, testing and
   production share routing semantics. These are the design.
2. **Opinionated defaults** — changeable by evidence: pinned key *choices*
   (note: pinned *keys* stay pinned — see below), confirm grammar, scroll
   behavior, debug key, theme defaults. Most of these are already classified
   as "overridable defaults" in D57; the spec text will make the class
   explicit in §11 too.
3. **Current limitations** — mouse, horizontal scroll, rich editing,
   per-item verbs, dynamic layouts. Renamed from "refused" to "deferred, not
   refused", each with its revisit trigger.

One pushback on this section: the critique lists "`esc`, `?`, and `:` are
permanently pinned" under the refusals-to-reconsider. We keep the pin, and
the spec's reasons (§4) are stronger than the critique acknowledges: the pin
is what makes capture liveness *unauditable-by-construction* ("you never
audit what cannot be misconfigured"), and esc-as-unconditional-exit is a
promise to the app's *users*, which is exactly the class of thing the
critique elsewhere says should not be rekeyable. This is an invariant, not a
dogma — but it sits in tier 1, and the three-tier split will make that
classification itself auditable.

### 2.3 — Lifecycle and identity (critique §3) — **adopt; the single biggest accepted item**

Agreed in full. The spec defines ownership but not *lifetime*, and several
of the listed scenarios are genuinely undefined. A new **§3.5 "Identity and
lifecycle"** lands before the tag. Draft answers, to be ratified by the
red-team pass (§2.10):

- **Two kinds of identity, explicitly named.** *Structural identity* =
  `pane name @ screen name` — what routing, help, receipts, and diagnostics
  address. *State identity* = the pane instance value — where cursor, scroll,
  and view state live. The spec conflated these; the conflation is the source
  of most of the critique's question list.
- **Pane reuse across screens** is legal and *stateful*: the same pane
  instance mounted on two screens shares its cursor. Documented feature
  (drill-down preserves selection), not an accident.
- **Focus pointing at a nonexistent pane** cannot persist: the focus ring is
  re-derived per frame from the layout; if the focused pane's name is absent,
  focus resets to the first focusable. Deterministic, receipt-recorded.
- **Backing data replaced / modal target vanishes:** already answered by the
  Track handshake for cursors (re-anchor to nearest live ID, `target: none`
  in the receipt). The modal case gets one added rule: `Target` is guaranteed
  valid *at dispatch time only*; if data vanishes while a confirm is up, the
  handler owns its own domain re-check — the kit cannot prove domain truth
  (see §2.4).
- **Screen removed while on the stack** is unrepresentable by construction:
  the screen set is static ("declared once", §15); the stack holds references
  into it; there is no API to deregister a live screen. This will be stated,
  because the critique is right that unstated is indistinguishable from
  undefined.
- **Action switching screens / mutating the tree:** dispatch completes, then
  stack and layout effects apply; the receipt records the transition. Layout
  is re-derived next frame (see §5, case D).
- **Zero screens:** already an audit error (tuikit `Verify`: "no screens").
  Carried into greenfield.md's violation catalog.
- **Registration order changes:** digits 1–9 are derived data and *shift*
  with order — documented consequence, matching the D54 stance that
  position-derived identifiers should track the structure they describe.

Screen stack entry semantics (instance vs constructor, state while hidden,
same screen pushed twice) are exactly the kind of question the state-machine
table below must answer, and we will not answer them ad hoc here.

### 2.4 — "Everything is data" isn't fully true (critique §4) — **adopt the wording fix**

Accepted without reservation, and it is the cheapest high-value edit in the
whole critique. "Cannot lie" becomes: **"cannot disagree about anything they
observe."** The kit eliminates inconsistencies it can *observe and structurally
guarantee* — collisions, dead keys, scope claims, unconfirmed danger, help
drift. It cannot prove semantic truth: that a handler does what its help
says, that `Danger` matches actual destructiveness, that an action is safe to
repeat. §2.3 and §5 of greenfield.md get the corrected sentence. This protects
credibility precisely where the spec's credibility is the product.

### 2.5 — Action model (critique §5) — **partially adopt**

The critique asks: is `Action` a low-level dispatch primitive or the
declarative representation of application commands? **Resolved: a dispatch
primitive.** That is what the evidence supports — the six fields it carries
(`Key`, `Run`, `Help`, `Name`, `NeedsCursor`, `Danger`) are exactly the ones
the kit can observe and audit, and D56 collapsed three types into one only
weeks ago. The `Meta` extension-point sketch is accepted as a *documented
future direction* (one sentence in §7, no code): if operational apps demand
availability/permission/async metadata, the growth path is an optional
`Meta` struct or marker interfaces — additive, non-breaking — rather than
field accretion on the base type. The refusal to add the full metadata table
to v0.1 is explicitly agreed; the critique's own recommendation and the
spec's type-budget rule say the same thing.

### 2.6 — Palette global discovery (critique §6) — **defer (already deferred)**

The spec already contains this exact feature as a bounded future: "a future
opt-in 'reachable' mode may list sibling actions with stated prerequisites…
the invariant survives because the entry states what will happen" (§5), and
D58 records it as a decision, not a refusal. What remains refused is
*search-everything*, and that refusal gets sharper wording after this
critique: cross-screen discovery requires a transition model ("switch screen
→ select → execute") that does not exist yet; shipping it before lifecycle
semantics are pinned would be a lie vector of exactly the kind §5 exists to
prevent. Trigger to revisit: a migrated real app where operators demonstrably
lose time to the strict palette. Two modes, when it comes, will follow the
critique's split — contextual default, global opt-in with disclosed
prerequisites.

### 2.7 — Testing emphasis (critique §7) — **adopt**

Agreed, and the spec is closer to this than the critique credits — but the
*ordering in the docs* is wrong and that matters for how users learn the
tool. kittest's primary interface will be documented as the structural
assertions (`SlotOf`, `Region`, `Dump`, dry-run, receipts); golden frames
become the secondary tool for deliberate visual regressions, with the
documented caveats (width, color profile, lipgloss version). "Semantic
testing first, golden rendering second" goes into §10 verbatim.

### 2.8 — Receipts as a strategic asset (critique §8) — **adopt the cheap half, defer the expensive half**

The machine-readable behavioral trace the critique sketches is, structurally,
what the receipt stream already is — sequence-numbered, timestamp-free,
uniform `target`/`result` fields, shared by debug view and tests. The cheap
and durable commitment adopted now: **treat the receipt schema as a public
contract from v0.1** — additive-only evolution, no field repurposing — so
that replay, coverage, and regression-generation tooling can be built *on*
it later without a migration wall. The expensive half (recording/replay
tooling, interaction coverage reports, regression generation) is deferred
with the trigger: a real debugging session that receipts could have answered
but didn't. The critique's observation that this stream is more defensible
than the UI abstraction itself is correct and is why the schema — not the
tooling — gets the guarantee now.

### 2.9 — Adoption friction (critique §9) — **partially adopt**

Two honest concessions. First, rung 1 (`kit.Run` kills Model boilerplate) is
the weakest rung — Bubble Tea boilerplate is small, and the critique is right
that it is not a compelling door. The ladder stays (its rungs are about
*independence*, not marketing), but the documented pitch order changes: the
visible-benefit rung is **verbs + generated help** (rung 3), where the
one-table payoff first appears, and rung 1 is described as plumbing, not
value. Second, the **10-minute test** — "a developer receives visible benefit
within 10 minutes" — is adopted as an explicit design gate for the publish
examples: the day-one example must demonstrate generated help and a dry-run
assertion, not just wiring.

Pushback: the conceptual-load list ("Screen, Pane, Layout, Slot, Action, Ctx,
Cursor, Modal, Track, Theme, Min, Scope, Receipt") counts nouns, not load.
The day-one example requires seven names to *write* and roughly three to
*understand* (screens hold panes; actions are verbs; everything derives from
the table). The genuinely load-bearing concepts — the Track handshake and
scope-as-placement — are conventions, not types, and each replaces something
worse (change hooks and scope fields). That said, this critique is only
falsifiable empirically, which is what the blind-test proposal (§3.14)
is for. Accepted there.

### 2.10 — Architecture improvements (critique §10)

**A. App as façade — adopt.** Agreed without qualification: a small public
API is good, a monolithic implementation is not. `App` stays the single
user-facing type; internally the implementation splits into private
subsystems (runtime, registry, router, stack manager, receipt recorder,
layout engine, test driver) with no public surface growth. This is an
implementation note in ARCHITECTURE.md, not a spec change.

**B. Render purity — adopt the formalization, refuse the redesign.** The
critique read the code correctly: `Render` mutates kit state via `Track`, so
"render is pure" is imprecise. We adopt the explicit contract rather than
moving `Track` to a separate phase, because the handshake's power *is* its
placement — declare-where-you-draw is what makes staleness impossible to
accumulate and keeps the per-pane cost one line. The contract, formalized in
§3:

> `Track`/`TrackLines` are **idempotent normalization**: they may mutate kit
> sync state only (cursor anchor, scroll pin), never domain data, never
> produce observable side effects, and are safe to call any number of times
> per frame. `Offset()`/`Index()` are pure. Rendering is deterministic in the
> strong sense: same (data, wiring, env) → same frame, regardless of call
> count.

The critique's seven-step lifecycle (event → transition → declare →
reconcile → geometry → render) is adopted as documentation of the frame
pipeline — it is, in fact, what already happens; now it is written down.

**C. Notify — partially adopt; document, don't rename.** The critique's
diagnosis is right: `Notify` needs a precise story or it looks like the
change-hook reflex sneaking back. But renaming to `Lifecycle{OnVisible,
OnResize}` costs vocabulary for zero semantic gain — the interface already
has exactly the two methods the rename would name. Adopted instead: §7 gains
the explicit semantics — fires only on (a) resize while the pane's screen is
current, (b) screen becoming current; must be idempotent; must not launch
domain work (`tea.Cmd` never originates here); it is a **geometry/lifecycle
courtesy, not a data-sync mechanism** — the one-sentence contrast with Track
that makes the distinction unfoggy. The word "courtesy, not truth" (already
in §7) is doing exactly the work the critique asks for; it just needs the
guarantees spelled out.

**D. Stack semantics — adopt as a pre-tag gate.** Fully accepted, including
the conclusion: a state-machine specification for the two stacks before
feature work. The question list (same screen twice on the stack, state while
hidden, footer during modal, palette over a confirm modal, screen entry
parameters, activation callbacks, reversibility) is adopted verbatim as the
seed of that table. Note one answer that falls out of existing rules: the
palette-over-confirm case is currently *defined* (universal verbs answer
during capture, §4) but arguably *wrong* — opening the palette from inside a
danger-confirmation feels like a capture leak with extra steps. The state
machine will decide it deliberately; the current behavior is provisional,
not ratified.

**E. Audit severity — partially adopt.** The severity model already half
exists: tuikit's `Verify` returns `Issue{Severity: Error | Warn}` with
exactly the Error/Warn split and CI behavior the critique proposes. Adopted:
greenfield.md's `Violation` gets the explicit three-level model (Error /
Warning / Info) with the critique's catalog as the seed — unreachable
screens, duplicate descriptions, missing quit action as Warnings;
palette-label and help-suggestion nits as Info. The closing argument — "a
validator that only says 'here are violations' will eventually become noisy"
— is correct and is the reason Info exists: it gives the audit somewhere to
put taste that isn't a red build.

### 2.11 — Missing capabilities (critique §11) — triage

| Critique item | Verdict | Note |
|---|---|---|
| Lifecycle semantics | **adopt** | §3.5, pre-tag (see 2.3) |
| Focus transitions | **adopt** | part of the state-machine table — transition rules for gain/loss/reset |
| Disabled/unavailable actions | **defer** | the first real post-v0.1 candidate; requires an availability model that doesn't pretend to be data-derived. Trigger: first migrated app that hand-rolls "no selection → do nothing" more than twice |
| Action outcome model / error presentation | **defer** | the kit never owns data, so "the action failed" is the app's truth to present; a *convention* (receipt `result: failed` + Debugf) may be enough. Revisit with migrations |
| Screen/pane identity model | **adopt** | §3.5 |
| Accessibility / Unicode width | **adopt (awareness), defer (work)** | Unicode width correctness is inherited from lipgloss/bubbletea; the kit's job is to not *break* it (equal-height slots, truncate-or-wrap is pane business). Screen-reader interaction is a genuinely open question for TUIs generally — noted, not claimed |
| Palette disabled-reasons, search, categories | **defer** | behind §2.6's trigger |
| Recording/replay | **defer** | schema-stable receipts are the enabling move (2.8) |
| Configurable keymaps | **partially adopt** | overridable defaults already exist (D57); *user-facing* keymap config is a product decision deferred to publish |
| State persistence, error boundaries, async feedback | **defer** | Priority-2 consensus; no trigger yet |
| Mouse, horizontal scroll, Priority 3 | **defer** | agreed — not until real demand |

### 2.12 — Business angles (critique §12) — **defer, all of it**

Already covered in §0. The one-line responses: Angle 1 is the default path
and requires no decision now; Angle 2 (starter kits) is attractive but is a
*content* business competing with ammaboard itself for the same hours —
declined while the beta date stands; Angle 3 is the genuinely interesting one
and is preserved by the receipt-schema decision (2.8); Angle 4 (optional
framework package) is compatible with "library, not framework" and is the
natural shape *if* publish ever happens; Angle 5 (ecosystem standardization)
is a long shot the spec's vocabulary is accidentally well-shaped for — noted
in the decision log, built for by nothing.

### 2.13 — Competitive positioning (critique §13) — **defer**

"The missing interaction layer for Bubble Tea" is a better one-liner than
anything in greenfield.md §1 and will be parked at the top of the publish
README draft. The positioning table is accepted as the contrast structure for
that README. Nothing to do until the publish gate opens.

### 2.14 — Validate before investing (critique §14) — **already agreed; extend the metrics**

The spec's ship sequencing already ends in migration-before-tag (§14 step 4:
"lazylite migrates as proof"), and tuikit.md's acceptance criteria are a
cruder version of the critique's metric table ("new pane → one file + one
registration"). Adopted extensions:

- **Broaden the migration set** beyond lazylite: ammaboard's own TUI (screens,
  modals, full-bleed fault screen — good modal coverage) and one
  list/detail-with-danger app cover the critique's profiles 1, 2, 4, and 5;
  the tiny two-pane case is the day-one example itself (profile 1).
- **Adopt the metric table** (LOC, custom handlers, time-to-add-screen,
  interaction test coverage) as the measurement sheet for those migrations.
- **Adopt the blind test** — hand a Go developer a small Bubble Tea app and
  the kit, watch where they struggle — as the *post-v0.1* validation step.
  Pre-tag, the authors' own migrations are the honest available evidence;
  the critique is right that they are weaker evidence than a stranger's
  confusion, and wrong that another architecture exercise is what's being
  defended against — the red-team pass (2.10d) replaces it.

### 2.15 — Roadmap changes (critique §15) — **partially adopt**

The critique's five phases and the spec's §14 sequencing are already ~85%
isomorphic: core contract first, developer feedback loop second, UX
projections third, migration proof fourth, tag last. Two adopted deltas:

1. **Migration evidence becomes an explicit tag gate.** §14 step 9 changes
   from "once steps 1–8 are green" to "once steps 1–8 are green **and** the
   migration metrics (2.14) show net simplification on two real apps." If
   the migrations show the kit makes apps *harder*, the tag slips — that is
   the critique's most important process correction.
2. **The red-team pass (2.10d) is inserted as step 4.5** — between cursor/
   Track landing and router-as-query, i.e., before the audit becomes the
   load-bearing gate, while semantics are still cheap to change.

The critique's "do not let debug palette or theme sophistication block phase
1" is already the spec's own ordering (steps 8 vs 1–3) — convergence noted.

---

## 3. The five challenges, answered directly

1. **"Are you solving the most painful problem, or the most satisfying
   one?"** The pain table (critique §1) is the strongest evidence available
   pre-migration, and it supports the kit's actual center of mass:
   collisions, discoverability, capture correctness, cursor drift, testing —
   all rated high-pain by the critique itself, all rated excellent by the
   same author. The satisfying-but-maybe-unimportant parts (composition
   elegance) are precisely the ones the spec refuses to elaborate. Verdict:
   the evidence says the thesis is aimed right; the migrations are what will
   actually settle it.
2. **"Is the API simpler than the code it replaces?"** Unknown until
   migration day, by design. The 10-minute gate (2.9) and the metric sheet
   (2.14) are the instruments; the tag gate (2.15) is the enforcement.
3. **"Are constraints becoming sacred?"** Partially true, now fixed by the
   three-tier taxonomy (2.2). The pinned verbs survive scrutiny as tier-1
   invariants; everything else is explicitly demoted to default or
   limitation.
4. **"Is the testing/receipt system more valuable than the UI
   abstraction?"** Yes — and the spec half-knows this ("the stream you watch
   is the stream tests assert on") without drawing the conclusion. Drawn
   now: the receipt schema is the durable asset (2.8), and the publish
   positioning leads with reliability, not components (2.13).
5. **"What is the product after the library?"** Nothing, presently, by
   decision. The kit's product is ammaboard. If the publish gate opens,
   Angle 3 (reliability tooling over receipts) is the designated direction —
   chosen because it builds on the asset that is hardest to copy: a schema
   plus a testing culture, not a layout engine.

---

## 4. The red-team pass — the accepted next engineering step

The critique's closing recommendation is adopted as the immediate pre-tag
work item, before the state-machine table is written:

**Walk 20–30 hard runtime scenarios and find contradictions.** Seed list
(from critique §3, §10d, plus the stack questions):

1. Screen pushed twice; esc pops which state?
2. Screen state while hidden; resize delivered late or never?
3. Modal open over a confirm modal (nested confirm of a danger action's confirm?)
4. Palette open over a modal; over a screen-stack entry.
5. Danger action whose handler pushes a screen.
6. Target item deleted between confirm push and confirm yes.
7. Pane instance shared by two screens, focused on one, Track'd by both.
8. Focus on a pane whose layout position changed this frame.
9. Screen whose layout function returns a different tree per frame.
10. `NeedsCursor` action fired from palette when focus moved since palette opened.
11. Resize below `Min` while a modal is up (TooSmall gate vs capture).
12. Zero items + `NeedsCursor` + palette entry.
13. `GLOBAL_UNWIND` with empty stacks and no focus.
14. Same key registered at pane scope on two sibling panes — legal; palette shows which?
15. Scroll `TrackLines` with a total that shrank below the viewport mid-frame.
16. Confirm grammar rebound to `esc` (collision with the pinned unwind?)
17. Action handler panics — what does the frame function guarantee?
18. Stack depth grows unboundedly via an action that pushes itself.

Rule of the exercise: every scenario ends in either a ratified rule in
§3.5/the stack table, or a new audit rule, or a documented refusal. Anything
that ends in "undefined" is a spec bug.

---

## 5. The composition deep-dive — responses

The critique's appended composition analysis gets its own section because it
arrived with its own method (cases A–D, the five-layout benchmark).

### 5.1 — The "Good not Excellent" rating — accepted, with a reframe

Accepted: composition is table stakes here, deliberately. The reframe: the
rating is against a dimension the kit is *not trying to win*. The spec's
§0 is explicit that layout is the commodity half of the pair; the critique's
own conclusion ("composition is good enough that users barely think about
it, while routing… is what they cannot get elsewhere") is the spec's thesis
restated. Moving Good→Excellent via the *experience around* the primitives
— exactly the critique's recommendation — is adopted as the approach.

### 5.2 — Case A–D, answered

- **A (single-pane screen):** the day-one example is the current answer, and
  the ceremony count (7 concepts) is real. One adopted mitigation: the
  single-pane case gets a documented two-line recipe in the cookbook
  (`kit.Screen("x", kit.Min(...), kit.Pane("main", p).Fill())`), no new
  constructor. A `kit.Single()` preset is deferred behind the migration
  benchmark — if migrations show single-pane screens are common *and* the
  recipe is copy-pasted, the preset earns itself.
- **B (list/detail) and C (temporary third pane):** fully expressible today
  with `Row`/`Cols`/`Fill`/`Fixed`; no new composition type needed. Presets
  (`ListDetail`, `Sidebar`) deferred per the critique's own guardrail — 2–3
  only, and only if migrations repeat the declarations.
- **D (state-dependent layout):** the critique assumed this is refused. It
  is not — and this is the composition section's most substantive outcome.
  tuikit's screens already derive layout per frame (`Layout func() []Region`),
  and the greenfield "layout expression" is deliberately compatible with a
  function of state. What is refused is *mutating the tree in place* and
  *dynamic routing tables*. Adopted clarification for §3/§6:

  > A screen's layout may be a function of application state, recomputed
  > every frame. The invariants that make this safe: pane instances are
  > stable across recomputation (state identity is the instance, §3.5),
  > structural identity tracks position (D54), the focus ring re-derives
  > per frame with deterministic reset, and bindings stay static. Layout
  > choice may be dynamic; behavior may not.

  This makes the critique's showLogs example *already legal*, pending only
  the §3.5 identity guarantees — which is a nice demonstration that the
  lifecycle section was the real blocker all along.
- **Five-layout benchmark:** adopted as the measurement instrument for the
  migration phase (alongside the §2.14 metric sheet): implement the five
  layouts in kit and in plain Bubble Tea/Lipgloss, record LOC / concepts /
  time-to-change / resize bugs / state loss. This is the falsifiable form of
  "Good vs Excellent".

### 5.3 — Intention-oriented API, defaults, visual language — mostly rejected, cheaply

- **Sidebar/Main/FlexiblePane intent wrappers:** rejected for now. They add
  a second vocabulary over the constraint primitives, and the critique
  itself flags the synonym-trap (`Sidebar`/`Main`/`Stack`/`Group` as layout
  constructors is how DSL rot starts). `Fixed(30)` + `Fill()` already says
  "narrow sidebar, main takes the rest" in the vocabulary the docs teach.
  Revisit only if the benchmark shows constraint arithmetic is a real
  time sink, not a familiar one.
- **Safe ergonomic defaults:** partially adopted. Default focus to the first
  focusable (already the behavior), default constraint `Fill` (already),
  default screen chrome. Rejected: inferring `Min` — minimums are audited
  *promises* (D51); a default promise is an unaudited lie waiting to happen.
- **Dangerous implicit behavior list** (auto-focus changes, silent resize of
  fixed panes, auto-hiding panes, implicit scope changes): adopted verbatim
  into the §11 invariants tier as named non-behaviors.
- **"Composition as visual language":** agreed as the *quality bar for the
  cookbook*, not as new API. The existing `Row`/`Cols`/`Pane`/`Title` text
  already reads close to its wireframe; the cookbook's job is to make that
  the *taught* reading, with one example per pattern.

### 5.4 — Pane reuse (critique §4E) — adopt the guidance, refuse the generic

`ListPane[T]` as a kit type is refused — the kit ships machinery, never
widgets, and a generic list pane is a widget with a type parameter. But the
underlying need (four list panes, one interaction pattern) is real and is
solved at the application layer: the pane interface is small enough
(`Render` + optional `Keys`) that an app-defined generic helper over its own
data costs nothing the kit must own. Cookbook entry: "reusable app panes —
one pattern, four stores." The kit makes composition easy; it does not own
the components (§0's oldest fence, unchanged).

---

## 6. Concrete amendment list (what actually changes)

| # | Amendment | Where |
|---|---|---|
| A1 | Repositioned pitch: "reliable interaction layer"; interaction-drift framing; corrected "cannot lie" → "cannot disagree about what it observes" | §0, §1, §2.3 |
| A2 | Three-tier classification of invariants / defaults / limitations; "deferred, not refused" | §11 |
| A3 | New §3.5 "Identity and lifecycle": structural vs state identity, focus re-derivation rule, dispatch→effects ordering, static-screen-set guarantee, target-validity scope | new |
| A4 | Track/Track/Offset purity contract formalized (idempotent normalization; frame pipeline documented as the 7-step lifecycle) | §3 |
| A5 | Notify semantics pinned (when it fires, idempotency, no domain work, courtesy-not-truth contrast with Track) | §7 |
| A6 | `Violation` severity model: Error / Warning / Info + seeded catalog | §5 |
| A7 | Semantic-first testing order; goldens demoted with documented caveats | §10 |
| A8 | Receipt schema declared a public contract: additive-only from v0.1 | §5, decision log |
| A9 | Action resolved as dispatch primitive; `Meta` recorded as the growth path | §7 |
| A10 | Layout-may-be-function-of-state clarification with the four safety invariants | §3, §6 |
| A11 | Dangerous implicit behaviors named as non-behaviors | §11 |
| A12 | Ship sequencing: migration-metrics tag gate; red-team pass inserted at step 4.5 | §14 |
| A13 | Stack state-machine table (ratifies the §2.10d question list) | new, pre-tag |
| A14 | Composition cookbook: five-layout benchmark, single-pane recipe, app-level pane reuse pattern | publish prep |

A1–A6, A10–A12 are pre-tag. A7–A9 are documentation-level and land with the
kittest/catalog work. A13 is the next work item (§4). A14 lands with the
publish prep, after the migrations.

> **Superseded by critique-2** (see §8): A4's formalization-only Track
> contract is replaced by the explicit sync phase (B6); A3's static
> screen-set and target-validity answers are revised by the screen-factory
> model and kit-owned target safety (B4, B10); A12's sequencing is amended
> by the compile-first step 0 (§8.7). The A-list stands otherwise; the
> B-list in §8.8 is the authoritative pre-tag blocker set.

---

## 7. Position after critique-1

The critique's engineering core is sound and mostly adopted: the lifecycle
gap, the render-purity ambiguity, the stack semantics, the severity model,
and the semantic-testing emphasis make the spec materially better, and the
red-team pass is the right next hour of work. Its product core evaluates the
kit against a thesis the project has deliberately deferred — publish is
gated on migration evidence, and the only PMF that matters pre-tag is
whether lazylite and ammaboard's TUI get simpler, which the roadmap already
measures. Its strategic core (receipts as the durable asset, reliability as
the positioning) is accepted as direction and preserved as decisions, not
built as code.

Where the critique and the spec disagree after this response, the
disagreements are now explicit and recorded: pinned verbs stay pinned,
`Min` stays a promise, the palette stays contextual, widgets stay outside,
and the DSL stays unbuilt — each for a stated reason that the red-team pass
can attack, rather than a preference it can't.

---

## 8. Response to greenfield-critique-2

> Critique-2 is a different kind of review than critique-1: it stopped
> designing and started **compiling**. It found two real Go bugs in the
> spec's sample APIs, several genuine runtime holes (async, text capture,
> the danger/confirm race), and an unsound audit fold. Both Go claims were
> re-verified here (go1.25.8) before this response was written — see §8.2.
> Its headline verdict — *don't freeze the draft; the static wiring is
> specified well, the runtime is not* — is **accepted as the pre-tag
> organizing principle**. Where critique-1 asked the spec to explain itself,
> critique-2 found the places where it cannot yet run. That is the more
> valuable review at this stage, and it revises several critique-1 verdicts
> (marked **superseded** below).

### 8.1 Verdict summary

| # | Critique-2 point | Verdict | Supersedes |
|---|---|---|---|
| 1.1 | Prior art exists (tuilib, ninebox, ochazuke); lead with verifiable wiring | **partially adopt** | — |
| 1.2 | Clean-slate convergence ≠ independent validation; outside app before tag | **partially adopt** | critique-1 §2.14 |
| 1.3 | "Library, not framework" is not honest — it owns Run/routing/keys | **adopt** | — |
| 1.4 | Adoption rungs not independent; standalone table package | **partially adopt** | — |
| 2.1 | Embedded `Cursor` shadows its own method — **verified** | **adopt** | — |
| 2.2 | Constructors collide with type names — **verified** | **adopt** | — |
| 2.3 | No async path | **adopt** | — |
| 2.4 | Danger target resolves after confirm — the race | **adopt** | critique-1 §2.3 (partial) |
| 2.5 | Text input impossible; `?`/`:` reserved during capture | **adopt** | — |
| 2.6 | Render purity: sync phase, not formalization | **adopt** | **supersedes** critique-1 A4 |
| 2.7 | Minimum fold unsound under Ratio | **adopt** | critique-1 §2.3 (partial) |
| 2.8 | `kit.Selected` contradicts "no package state" | **adopt** | — |
| 3 | Architecture gaps (17 items) | **mixed** — §8.4 | critique-1 §2.3 (revised) |
| 4.1 | Type budget broken; count concepts | **adopt** | — |
| 4.2 | Cut positional IDs, debug overlay, 12-type target | **partially adopt** | — |
| 4.3 | Refuse multi-select/chords/zoom explicitly | **adopt** | — |
| 4.4 | Separate documents | **partially adopt** | — |
| 4.5 | Panic recovery, blocking-handler rules, versioning policy | **adopt** | — |
| order | Compile-first suggested order | **adopt** | critique-1 §2.15 |

### 8.2 The Go claims — independently verified

Both blockers were re-tested on go1.25.8 before this response; they are real
and the critique was entitled to them.

**Claim 1 — cursor discovery fails (confirmed).** An embedded `kit.Cursor`
field is itself named `Cursor` and sits at depth 0, so it shadows the
promoted `Cursor()` method at depth 1. The interface check returns `false`
and the direct call does not even compile:

```
claim1: interface satisfied = false
./main.go:26:6: invalid operation: cannot call p.Cursor
  (variable of struct type Cursor): Cursor is not a function
```

The spec's wiring story ("embedding satisfies the kit's discovery interface
via Go promotion", §7) is wrong as written. **Adopted fix:** the discovery
method is renamed `KitCursor()` (critique-2's suggestion); the embedded
field is the `kit.Cursor` value itself and the method is a plain accessor on
the type, so the embedding still does the wiring and the zero-method claim
survives — but now it survives compilation.

**Claim 2 — constructors collide with types (confirmed).**

```
./main.go:5:6: Action redeclared in this block
```

`kit.Action("refresh", ...)` next to `type Action struct` cannot exist in
one package; same for `Screen` and `Pane`. **Adopted fix:** keep the twelve
type names (they are the vocabulary); constructors become `kit.NewScreen`,
`kit.NewPane`, `kit.NewAction` (critique-2's `Leaf`/`Do` are cuter but worse
for grep and docs). All spec samples updated accordingly.

### 8.3 Blockers 3–8

**3. No async path — adopt.** This is the single largest hole critique-2
found, and it is also the hole critique-1 gestured at ("async/background
orchestration: deliberately out of scope") without noticing that *out of
scope* is not *possible to avoid*. The spec's own non-goal says async domain
work arrives through bubbletea's contract and "passes through untouched" —
but if `kit.Run` owns the Model and panes have no `Update`, there is no
untouched channel: the kit is a dead end for any app with a background
refresh, which is the spec's own target audience (§0: "operational
dashboards… log explorers"). The out-of-scope clause claimed to leave the
developer's domain alone while removing the pipe it flows through.
**Adopted:** handlers return `(tea.Cmd)`; panes get optional `Init()` /
`Update(msg) (cmd, handled)` with the documented fan-out order — app hooks
first, then every pane, including background (non-visible) ones, because a
log pane tail-following while hidden is the normal case, not an edge case.
Unconsumed messages keep falling through to the host `Update` per §2.2.

**4. Danger/confirm race — adopt.** The spec says `Target` resolves "the
moment an action actually runs" — and for a `Danger` action, that moment is
*after* the confirm modal, potentially several refreshes later. The cursor
clamps to a neighbor in between; the operator confirms killing job-123 and
the kit delivers job-124. This is the worst possible failure mode for the
one action class the kit explicitly guards. **Adopted:** the target ID is
captured at trigger time (key or palette — identical path per the one-resolver
rule), displayed in the confirm modal ("kill job-123?"), re-checked against
the live list on `CONFIRM_YES`, and the action aborts with a receipt-recorded
`result: target-gone` if it vanished. This also supersedes the partial answer
given to critique-1 §2.3 ("the handler owns its own domain re-check"): target
safety is now kit-owned up to the dispatch boundary, not delegated.
Note what this does *not* adopt: the kit still cannot re-validate domain
truth at execution time (the store may have moved the item) — the boundary
remains "the kit guarantees the target existed and was shown at confirm";
the handler's re-check is a convention, documented.

**5. Text input impossible — adopt, and it is a spec self-contradiction.**
§4 says the top modal "captures all keys" while "universal verbs still
answer during capture"; §4 also says the palette "grabs input
(type-to-filter)". Both cannot hold: typing `?` into the palette filter opens
help. The same trap kills confirm-with-comment flows and any commit message.
The pinned-verb rationale (§2.2 above) is untouched — liveness needs esc
during capture — but *three* universal verbs during capture is one more than
liveness requires. **Adopted:** an explicit **text-capture mode** on modals:
only `esc` (unwind) and `ctrl+c` (quit) intercept; every other key,
including `?` and `:`, flows to the captured input; paste supported; hosting
a bubbles textinput is the documented path (the kit hosts, never rebuilds —
§11's oldest fence). `?`/`:` being unreachable *inside* a text field is the
correct trade: the user is typing prose, not commands, and esc is always
there.

**6. Render purity — adopt, superseding critique-1 A4.** Critique-1 offered
two fixes for `Track`-in-`Render`: formalize the contract (chosen) or move
the sync to an explicit phase (declined). Critique-2 closes the debate with
a correctness argument the formalization cannot answer: **dry-run routing
must resolve `Ctx.Target` without rendering**, so with Track-in-Render a
dry-run receipt shows a stale or `target: none` — which breaks the
"palette entry executes exactly as the keypress would" invariant, the
spec's own §5 centerpiece. The formalized guarantee ("safe to call any
number of times per frame") does not fix staleness between dispatches.
**Adopted, superseding A4:** panes expose declarations (item IDs, content
heights) via a method the kit calls in an explicit **sync phase** after each
message dispatch and each resize; cursor and scroll normalize there;
`Render` is genuinely read-only. The Track handshake survives — pointers
never contents, declare-then-derive — but declare happens in sync, not
draw. Sticky scroll (`pinned`) also moves into Track/sync, answering the
"pure Offset can't remember position" point. Cost: one method on stateful
panes; benefit: purity becomes true instead of documented-as-approximately.

**7. Minimum fold unsound — adopt.** The critique's counterexample is
correct: the D51 fold compares *declared* minimums against the screen's
`Min`, but runtime distribution under `Ratio` can starve a leaf below its
declared minimum while the fold passes (width 100, `Ratio(1,10)` gives a
pane needing 30 exactly 10). The "honest minimum" the audit reports would
be dishonest precisely when the layout's arithmetic makes it so — the audit
would certify the 0×0 bug class the whole design exists to kill.
**Adopted:** the audit runs the actual constraint solver at the screen's
declared `Min`, checks every leaf against its declared minimum, and searches
upward for the true honest minimum. Rounding behavior and `Fixed`-overflow
(the declared sum exceeding the screen) get specified values — overflow is
already an audit error in tuikit `Verify`; rounding is defined as
largest-remainder, documented. This supersedes the fold-only answer implied
by critique-1's acceptance of D51: the fold survives as a fast pre-check,
the solver run is the truth.

**8. `kit.Selected` package state — adopt.** Confirmed contradiction: a
free function cannot see the wire-up-injected theme without package state,
and §7/D55 say "never global". **Adopted:** the effective theme rides in
`Slot` (which already carries the per-frame env: `{W, H, Focused}` becomes
`{W, H, Focused, Theme}`), panes call `s.Selected(line)`, and `SetTheme`
injection dies — the theme is an environment fact, and §2.4's principle
("facts have one owner") says environment facts live in the env, not in
late injections. Slot growing a field is the right cost; it keeps
"deterministic frame = f(data, wiring, env)" honest, since the theme was
always part of env.

### 8.4 Architecture gaps — triage of the 17-row table

| Gap | Verdict | Response |
|---|---|---|
| No API for modals/details/focus/quit/errors | **adopt** | `Ctx` gains typed methods (`Push`, `Pop`, `Switch`, `Focus`, `Quit`, `Notice`). Today the spec implies handlers reach into `Ctx.App` with no defined surface — the guarantee-carrying operations must be kit methods so every one is receipt-recorded. A status-notice convention (persistent one-line result area) is adopted as projection-of-receipts, not new state |
| Drill-down has no parameters; repeated screen shares cursors | **adopt (revises critique-1 §2.3)** | Screen entries in the registry become factories; a push instantiates. This *revises* the earlier "screen set is static, removal unrepresentable" answer: the set of screen *types* stays static, stack entries are instances, and the "same screen twice" question resolves itself — two instances, two states. Detail screens get no digit (digits address registry roots, `goto` targets) |
| Digit mid-drill | **adopt** | `goto` resets the screen stack. Digits mean "coarsest navigation" (§4's own words); landing inside a drill chain contradicts it. Receipt records the pop count |
| Shadowing and `ctrl+c` trapping | **adopt** | `ctrl+c` joins the pinned class (liveness: every state has a guaranteed exit *and* a guaranteed kill). Pane keys shadowing kit defaults (digits, quit) require explicit `.Override()` — silent replacement (current D57 rule) is downgraded to a Warning-plus-requirement, because silent scope wins are exactly the drift the audit exists to catch |
| Tab focus missing from catalog | **adopt** | Spec bug: §3 says "slot-jumping is tab's job" and the catalog has no such actions. `FOCUS_NEXT`/`FOCUS_PREV` added as kit mechanics. "Blur" as an esc side effect is dropped — esc with empty stacks is a no-op, receipted; focus should be reachable *and leavable* symmetrically via tab, not by accident via esc |
| One key per action vs `enter`/`y` | **adopt** | `Key string` → `Keys []string`; first is canonical (help/palette/footer show it), the rest are aliases. The confirm grammar was already unrepresentable — two defaults for `CONFIRM_YES` — so the type was wrong, not the grammar |
| Mechanics unnamed but overrides match by name | **adopt** | Separate internal `ID` (override matching, receipts, audit) from optional `Title` (palette/help display). Fixes D56/D57's collision: "Name empty = mechanic" and "override by same name" cannot both hold for mechanics |
| Availability state | **partially adopt** | The pure `Enabled(ctx) (ok bool, reason string)` shape is right and is now the documented growth path in §7 (alongside `Meta`). Still deferred from v0.1: it needs a context the table can evaluate cheaply for all actions, and the v0 escape (action doesn't fire; receipt `target: none`) is honest. Footer/palette disabled-with-reason is the first post-tag feature candidate, ahead of critique-1's queue position |
| `Keys()` cached at wire-up | **adopt** | "Static by convention" becomes static by construction — one call at wire-up, cached; a pane returning different bindings per frame breaks the one-table invariant and should be unrepresentable, not discouraged |
| Uniqueness checks | **adopt** | Spec bug: D54's "unique by construction, no uniqueness checks" is true for *group IDs* only, but reads as if it covers pane names. Two panes named "queue" on one screen collide in routing, help, and `kittest.SlotOf`. Audit adds: screen names unique, pane names unique per screen |
| Slot vs chrome semantics | **adopt** | `Slot` is the content area; kit draws borders/titles outside it; panes declare body height per row. Currently ambiguous enough to produce off-by-chrome bugs in every pane |
| Overflow handling | **adopt** | Render output is clipped/padded to the slot by cell width — kit-owned, deterministic; a debug-only violation counter records panes that overflow habitually. "Draw exactly your Slot" was a contract without an enforcement story |
| Scroll mode edge cases | **adopt** | Empty `Track(ids)` flipping to line mode is a real bug in the spec's own encoding (`anchor == "" ⇒ line mode`). `Scroll` gains an explicit mode field; `pinned` only set after explicit tail intent (`ScrollTail()`), never inferred from short content |
| Duplicate/removed item IDs | **adopt** | `Track` reports duplicate IDs (receipt/violation — duplicates make identity anchoring meaningless); removal keeps the last known index for nearest-row recovery instead of falling to 0 |
| Receipt schema, `Debugf` concurrency | **adopt** | Already adopted from critique-1 (A8) — fixed fields + outcome enum confirmed; the ring buffer is mutex-guarded since `Debugf` is callable from `tea.Cmd` goroutines, which the async path (§8.3.3) now makes a normal case |
| Escape hatch designed, not asserted | **partially adopt** | The host-shaped hatch (`Init/Update/View` exposed, raw-key pane interface) is specified, but the raw-key pane is deferred: it exists to host mouse-first or canvas panes, which are §11 deferreds. The spec sentence "every layer can be bypassed" is downgraded to list the bypasses that actually ship |
| Bubble Tea v2 | **partially adopt** | The repo already targets `charm.land/bubbletea/v2` v2.0.9 (tuikit.md dependency law) — so this is a spec-text bug, not a migration: greenfield's key-string examples (`ctrl-d`, `ctrl-u`) predate v2 spelling. Normalized to `ctrl+d`/`ctrl+u` throughout. Background-color messages pass through per §2.2's fall-through, now documented |

### 8.5 Scope and document

- **Type budget broken — adopt.** Confirmed by counting: `Scroll`, `Item`,
  `Handler`, `Keyer`/`Notifier` interfaces, and the constraint builders are
  public surface outside the twelve. The 12-type budget was a good
  discipline that became a false advertisement. **Adopted replacement:**
  the budget is restated as *the day-one user learns ~10 concepts to build
  the day-one app* (Screen, Pane, Row/Cols, Fill/Fixed, Action, Min, keys) —
  measured against the compiled example, not the full type list, which is
  the metric critique-2 proposes and the only one that maps to load.
- **Cut positional group IDs — partially adopt.** They stay, but demoted:
  they are diagnostics-internal (Dump, violations), cost ~30 lines, and
  `kittest.SlotOf` addresses by pane name, not group ID. What is adopted
  from the critique: group labels + tree paths appear in human-readable
  diagnostics *first*, the `main__C1xR0` token becomes an implementation
  detail of Dump output rather than spec'd API. Deferral to post-tag would
  also have been fine; the cost is too low to churn the spec over.
- **Debug overlay deferred — adopt.** Ship sequence amended: receipts go to
  a log file (`--receipts=path`) before any overlay exists. The overlay is
  palette-shaped and palette-adjacent anyway; it can borrow the palette
  implementation when it lands. This also decouples step 8 of §14 from the
  release.
- **12-type target dropped — adopt** (see above).
- **Refuse explicitly: multi-select, key chords, zoom/collapse — adopt.**
  All three added to §11 with one-line reasons: multi-select breaks the
  one-cursor target contract (a second selection is a second identity the
  table can't see); chords complicate capture and receipts (one key, one
  receipt, forever); zoom/collapse is a layout-mutation flavor and waits for
  the layout-as-function clarification (A10) to be battle-tested.
- **Separate documents — partially adopt.** greenfield.md keeps the spec +
  delta (they are one review artifact by design, §13's table *is* the
  diff), but the decision log seeds (Appendix, D46–D59) move to
  `decisions.md` where D1–D45 already live. Modest, keeps the greenfield
  doc readable as a spec.
- **Panic recovery, blocking rules, versioning — adopt.** `kit.Run` installs
  panic recovery with terminal restore (bubbletea handles the raw-mode
  teardown; the kit guarantees the receipt tail is flushed to the log first)
  and documents that a panicking handler aborts the dispatch, records
  `result: panic`, and the frame function's next output is the restored
  prior frame. Blocking-handler rule adopted: handlers must not block (the
  async path from §8.3.3 is the escape); v0 does not enforce it, receipts
  expose it (gap between sequence numbers makes a stall visible).
  Versioning policy adopted: receipt schema additive-only (already A8),
  public API pre-1.0 per Go convention, `Min`/behavior changes require an
  audit-rule counterpart — no silent contract changes.

### 8.6 Raison d'être items

- **Prior art — partially adopt.** The critique is honest that its
  comparison "rests on search excerpts, not a code review" — but the burden
  flips back: if tuilib, ninebox, and ochazuke feed help from action tables,
  the spec's §0 claim ("everyone reinvents this badly") needs evidence or
  softer wording. **Adopted:** §0 keeps the pain claim (it is the authors'
  own observed pain in lazylite and ammaboard — the D46 extraction story is
  the evidence) but drops "everyone"; §1's lead becomes exactly the
  critique's suggestion — what the prior art visibly lacks is **verifiable
  wiring**: an audit that fails CI, dry-run assertions, machine-readable
  receipts. A one-hour prior-art pass (clone all three, run their examples)
  is added to the publish prep; if one of them already has a CI-failing
  wiring audit, the positioning needs another look.
- **Convergence ≠ validation — partially adopt.** Already agreed in
  critique-1 §2.14 (blind test adopted, migration metrics adopted).
  What is not adopted: "one new app built by someone else **before the
  tag**". Pre-tag, the honest evidence is the migration metrics + red-team
  pass; the beta date (Dec 1) makes recruiting an outside builder now
  unrealistic, and a rushed outside app is worse evidence than none. The
  tag gate (critique-1 §2.15) stays migration-based; the outside-app build
  is the first post-tag gate before public announcement.
- **"Library, not framework" — adopt.** The critique is right and this is a
  genuine credibility fix. The kit owns Run, routing, layout resolution, and
  reserved keys — that is framework-shaped. What §2.1 meant (and failed to
  say) is: no global state, no service locator, data never enters the kit,
  and every ownership boundary has a documented bypass. **Adopted
  wording:** "a small framework with escape hatches at every layer" — and
  the escape hatches are now *specified* (§8.4, host-shaped hatch) rather
  than asserted.
- **Rung independence — partially adopt.** The claim "every rung
  independently valuable" was overreaching: rungs 4–5 (audit, tests)
  require rungs 2–3. What is true and kept: every rung *adds* value and you
  can stop anywhere. The standalone-package proposal (table + resolver +
  audit without screens) is deferred — the audit's geometry half needs
  layouts, and splitting the module pre-tag trades real simplicity for
  a packaging claim. Revisit if a consumer wants routing-audit without
  screens.

### 8.7 The suggested order — adopt

Critique-2's five steps and the amended §14 (critique-1 §2.15) merge into
this pre-tag sequence:

1. **Compile the runtime contract first** — the day-one example *plus* the
   three things critique-2 names that the spec cannot currently express:
   an async refresh (`tea.Cmd` round-trip), a filter input (text-capture
   mode), a drill-down (screen factory + target-safe danger flow). This
   example is the new step 0; a spec that cannot compile its own pitch
   ships nothing. (Critique-2's order is adopted over critique-1's
   sequencing here — compile-first beats any amount of document polish.)
2. Red-team pass (§4 of this document) — now with the §8.3/8.4 answers as
   candidate ratifications to attack, and the two verified Go fixes
   already in.
3. Router, receipts, audit, `kittest` — with the solver-based minimum
   check (§8.3.7) and the severity catalog.
4. Footer, palette, theme — as projections of the one resolver.
5. Migration metrics on lazylite + ammaboard's TUI → **tag gate** (§6, A12).
6. Outside-app build (§8.6) → first post-tag gate.

### 8.8 Updated amendment list (critique-2 additions)

| # | Amendment | Where |
|---|---|---|
| B1 | Cursor discovery: `KitCursor()` accessor; spec samples corrected | §7 |
| B2 | Constructors: `NewScreen`/`NewAction`; all samples updated | §7 — *the pane constructor was later renamed `Leaf`; see §9.5, row 3* |
| B3 | Async contract: handlers return `tea.Cmd`; optional pane `Init`/`Update`; fan-out order documented; fall-through preserved | §3, §7 |
| B4 | Target safety: capture at trigger, show at confirm, recheck on yes, `target-gone` receipt | §4 |
| B5 | Text-capture mode: only `esc`/`ctrl+c` intercept; paste; bubbles-hosting documented | §4 |
| B6 | **Supersedes A4:** explicit sync phase; panes declare (IDs, heights); `Render` read-only; dry-run correctness restored | §3 |
| B7 | Minimum check: solver run at declared `Min`, per-leaf verification, upward search for honest minimum; rounding defined | §6 |
| B8 | Theme in `Slot`; `s.Selected(line)`; `SetTheme` deleted | §7 |
| B9 | `Ctx` methods (`Push`/`Pop`/`Switch`/`Focus`/`Quit`/`Notice`), all receipt-recorded | §7 |
| B10 | Screen registry = factories; stack entries = instances; detail screens undigit-ed; `goto` resets stack | §4 |
| B11 | `ctrl+c` pinned; explicit `.Override()` for shadowing kit defaults | §4 |
| B12 | `FOCUS_NEXT`/`FOCUS_PREV` in catalog; esc-blur dropped | §4, §7 |
| B13 | `Keys []string` (canonical + aliases); internal `ID` vs palette `Title` | §7 |
| B14 | `Enabled(ctx)` documented as the availability growth path | §7 |
| B15 | `Keys()` cached at wire-up; name-uniqueness audit rules (screens, panes-per-screen) | §5, §7 |
| B16 | Slot = content area; overflow clip/pad + violation counter | §3 |
| B17 | Scroll: explicit mode field; pin only after tail intent; duplicate-ID reporting; removal keeps last index | §9 |
| B18 | Receipts to log file before debug overlay; overlay decoupled from release | §8, §14 |
| B19 | Type budget → day-one concept budget (~10), measured against the compiled example | §2.6 |
| B20 | Explicit refusals: multi-select, chords, zoom/collapse | §11 |
| B21 | Panic recovery + terminal restore; blocking-handler rule; versioning policy | new |
| B22 | Prior-art pass on tuilib/ninebox/ochazuke; §0 "everyone" softened; §1 leads with verifiable wiring | §0, §1, publish prep |
| B23 | "Library, not framework" → "small framework with escape hatches", hatches specified | §2.1 |
| B24 | Rung claim reworded ("each rung adds value"); standalone package deferred | §12 |
| B25 | Key strings normalized to bubbletea v2 spelling (`ctrl+d`, `ctrl+u`) | throughout |

B1–B17, B19–B21, B25 are **pre-tag blockers** — they change the contract.
B18, B22–B24 are scheduling/positioning and land with the sequencing in
§8.7.

### 8.9 Final position (after both critiques)

Critique-1 evaluated the spec as a product thesis and was right about
lifecycle, severity, and testing emphasis. Critique-2 evaluated it as a
compiler would and was right about everything it tested: two API samples do
not compile, the audit's honesty claim has a counterexample, the
not-lying palette cannot resolve targets without rendering, and the target
audience's defining workload (async data) had no path through the kit. The
response to critique-1 adopted its *conclusions*; the response to
critique-2 adopts its *method* — **nothing in the spec is trusted that has
not been compiled or red-teamed**. The draft is not frozen (critique-2's
headline demand, accepted): it goes through §8.7's sequence, and the tag
waits for migrations that prove net simplification, an outside app, and a
red-team pass with no undefined outcomes.

The combined position, one line each: the architecture thesis survives both
reviews — the vocabulary, the one-table rule, the capture semantics, and
the ownership principles were attacked five ways and stand. The runtime
contract does not exist yet and is now the work. The product question
remains deliberately deferred behind migration evidence. The next hour of
work is step 1 of §8.7: make the pitch compile.

---

## 9. Response to the second review of tuikit-spec

> The third review compiled the consolidated spec and found the design
> direction sound but the document not freeze-ready: one new compile bug, a
> day-one example that fails its own audit, two internal contradictions in
> capture rules, and a set of runtime holes (modal contract, sync ordering,
> fan-out starvation, factory/instance split). Verdict: **accepted in
> almost full** — 27 of 28 items adopted outright, one (R7) partially adopted
> with
> the one-resolver invariant as the reason. This is the review that closes
> the "contradictions" class: every finding was either compile-verified or
> a direct citation of two spec sections disagreeing. The spec updates in
> §9.4 have been applied to [tuikit-spec.md](tuikit-spec.md).

### 9.1 Verdict summary

| # | Finding | Verdict |
|---|---|---|
| B1 | `Action` fields and methods share names — **verified** (`field and method with the same name Help`) | **adopt** |
| B2 | Day-one example registers global `kill` with `NeedsCursor` — fails its own audit | **adopt** |
| B3 | Sync runs before geometry; scroll/cursor need resolved dimensions | **adopt** |
| B4 | Modals have no contract | **adopt** (ratified in spec; validated in step 0) |
| B5 | Capture rules contradict: §5.5 vs §8; `CONFIRM_NO` binds forbidden `esc` | **adopt** |
| B6 | Screens both factories and instances; `Push` bypasses the registry | **adopt** |
| B7 | Dynamic layouts break static audit/caching guarantees | **adopt** |
| B8 | Scrolling lost its section in consolidation | **adopt** |
| R1 | `handled` starves panes; shared instances consume each other's replies | **adopt** |
| R2 | Host hook both first in line and the fall-through | **adopt** |
| R3 | `Init` on push unspecified; `Notifier` undefined | **adopt** (Notifier deleted) |
| R4 | Panic recovery "restores prior frame" while data may be half-mutated | **adopt** |
| R5 | Stalls shown as sequence gaps, but sequences carry no time | **adopt** |
| R6 | Store mutation from command goroutines not forbidden | **adopt** |
| R7 | Palette captures target at selection, after data may have moved | **partially adopt** |
| R8 | Confirm recheck doesn't name the pane; `Index` goes stale | **adopt** |
| RC1 | Receipt schema: not publishable as a closed enum; missing kinds/names | **adopt** |
| RC2 | Cross-scope key shadowing unchecked (`tab`, digits) | **adopt** |
| RC3 | `Mech` semantics ambiguous; page-step keys unnamed | **adopt** |
| RC4 | "Screen unreachable" unimplementable without navigation declarations | **adopt** |
| RC5 | "Missing quit path" impossible with pinned `ctrl+c` | **adopt** (dropped) |
| RC6 | Pass-through "audited" but has no rule | **adopt** |
| RC7 | Pane minimums lack a declaration API; honest-minimum search should be linear | **adopt** |
| RD1 | Escape-hatch promise outruns design; no embedding API | **adopt** |
| RD2 | "Pre-1.0" conflicts with tagging v1.0.0 | **adopt** (tag v0.1.0) |
| RD3 | Outside-user test + prior-art pass before the tag; metric thresholds | **adopt** (supersedes critique-1/2 deferrals) |
| RD4 | D7 shows pre-theme `Slot` fields | **adopt** |
| RD5 | "Anything not covered is a spec bug" premature | **adopt** |

### 9.2 The verified and the contradictory

**B1 — verified.** Recompiled on go1.25.8: fields `Help`, `NeedsCursor`,
`Danger` plus builder methods of the same names produce
`field and method with the same name Help` (×3). The critique is right that
this was a *new* clash introduced by the consolidation — the greenfield
draft's `Help` was only ever a method, and merging the field list and the
builder chain into one type sketch created the collision. **Adopted fix:**
`Action` fields are unexported; the builder chain is the API. This also
settles the field-vs-method question in the right direction — a mutable
public struct would let apps mutate actions after registration, breaking
the cached-static-table guarantee (§6.2) that the audit depends on.

**B2 — verified by reading.** §10.1 registers `kill` in the global block
with `.NeedsCursor()`, and §6.1/§12 make exactly that an audit Error.
The spec's own pitch example violated its own contract — the strongest
argument yet for the compile-first sequencing. **Adopted fix:** global
block keeps only `refresh`; `kill` lives solely in the queue pane's
`Keys()` (which §10.2 already showed correctly).

**B5 — verified by reading; the kit violated its own audit.** Three
inconsistencies:
1. §5.5 says capture intercepts only `esc`/`ctrl+c`; §8's lifecycle table
   said "`:` opens the palette over a confirm — provisional." Ratified:
   **§5.5 wins.** Only `esc` and `ctrl+c` answer during capture; the
   palette cannot open over a capturing modal. This also resolves the
   critique-1-era flag ("palette over confirm feels like a capture leak")
   in the conservative direction.
2. `CONFIRM_NO` defaulted to `esc`/`n` — but §6.3's audit rule forbids
   binding `esc` anywhere. The kit's own catalog was a standing violation.
   Ratified: **`CONFIRM_NO` is `n`**; `esc` inside a confirm is unwind
   (dismissal), which is a mechanism, not a binding — it never appears in
   the catalog and the audit never sees it.

**B3 — verified by reasoning.** Scroll tail-follow needs the viewport
height; `LineCount` of wrapped content needs the width — both exist only
after geometry resolves. The §5.1 pipeline (sync at step 3, geometry at
step 4–5) made the sync phase unable to do its job. **Adopted fix:**
transition → layout resolve → geometry solve → **sync with resolved
slots** → render → chrome; the declaration is `LineCount(width)`.

### 9.3 Design decisions ratified

**B4 — modal contract.** Adopted in full: `Modal` is now a specified
interface (`Render(w, h)`, `TextCapture()`; optional `Keys()`/`Update()`
for hosted inputs), `Ctx` gains `OpenModal(m)`/`CloseModal(result)`, and
modal results are delivered as messages carrying a correlation ID to the
opening screen's panes — which is what makes the confirm flow's
`confirm-yes`/`confirm-no` receipts correlatable. Confirm remains a
derived modal built by the kit. This is ratified in the spec now and is
one of the four contracts step 0 must exercise in real code.

**B6 — factories vs instances.** Adopted: the registry holds **named
factories**; navigation is `c.Push(name, params)` — instances never bypass
the registry, so the audit can enumerate every screen the app can ever
show by instantiating each factory with zero-value params. This makes
"unrepresentable by construction" (critique-1 §2.3) true again, with a
concrete mechanism. Day-one sample updated to register factory functions.

**B7 — dynamic layouts.** Adopted, and it tightens §9.3 rather than
reversing it: a screen declares a **finite set of named layout variants**;
the per-frame layout function only *selects* one. The audit resolves and
audits every variant; `Keys()` caching covers the union of panes across
variants. Layout choice stays dynamic; the audited surface stays static.

**B8 — scrolling.** Adopted as an omission, not a design change: the
consolidation dropped the scroll section, leaving D16 referencing a
contract with no body. Restored as §9.5 (struct, item/line modes with the
explicit mode field, `ScrollTail()` intent, ownership rules, the transition
table, and the edge rules — empty lists, shrinking content, duplicate IDs,
append-stability of line mode).

**R1/R2 — fan-out.** Adopted as a single ratified rule set: panes **always
receive** every message (each distinct instance exactly once — shared
instances deduplicated, which kills the two-detail-instances-swallowing-
each-other's-replies bug); the `handled` return is removed from pane
`Update`; the **host hook** (`Handle(msg) (cmd, handled)`) is the only
consumer that can stop descent, called first, exactly once. Fall-through
and first-in-line stop being contradictory because there is one hook, and
its `handled` means "consumed — do not fan out."

**R3 — `Init` on push; `Notifier` deleted.** Adopted both. Pushed screens
run `Init` when pushed (activation semantics without a notification
callback), and `Notifier` is deleted rather than defined: with per-frame
layout re-derivation and a sync phase that runs with resolved slots every
frame, a background screen misses nothing that matters — its panes
re-declare on the next frame after it becomes current. A callback whose
job the pipeline already does is drift surface, not courtesy.

**R4 — panic.** Adopted: the earlier "next frame is the restored prior
frame" was unsound — app data may be half-mutated, and rendering from it
invents state. Ratified: **restore the terminal, flush the receipt tail,
exit non-zero.** Covers panics in handlers, `Update`, `Render`, and
commands.

**R5 — stalls.** Adopted: receipts stay timestamp-free (their determinism
is the asset); **durations go to the debug log**, keyed by receipt seq.

**R6 — mutation rule.** Adopted as a documented rule: domain mutation only
in `Update` or handlers; command goroutines compute and read, never mutate.
Enforcement is by convention plus the receipts (§5.3), not a runtime
guard — the kit cannot see the store, and pretending otherwise would break
principle 2.

**R8 — recheck identity.** Adopted: `Target` carries the originating
pane's structural identity; the confirm recheck runs against *that
instance's* latest declaration, recomputes `Index` at execution, and
refuses when the ID is duplicated — the critique's "recheck in the
originating instance" is what makes `target-gone` mean one thing instead
of "gone from whichever pane looked."

**RC1 — receipt schema.** Adopted, superseding the §11.2 sketch:
- an **event `kind`** field (`dispatch`, `lifecycle`, `sync`, `debug`,
  `audit`) so non-dispatch events (duplicate-ID reports, `Debugf` lines,
  stack changes) live in the same stream — which the debug view always
  claimed;
- **pane and screen names** on dispatch events (a trace without them
  cannot answer "which pane did that");
- a **correlation ID** linking confirm pushes to their resolutions;
- a **dry-run marker** so live and simulated streams are distinguishable;
- the result enum is **extensible with unknown-value tolerance**, not
  closed — "closed enum" and "additive-only" were mutually contradictory
  as written.

**RC2 — shadowing.** Adopted and strengthened from the critique's ask:
cross-scope shadowing of kit defaults (digits, `tab`, mechanics) or of
another scope's binding is an **Error** unless the registering action
carries `.Override()`. Within one scope, declaration-order precedence
remains deliberate and legal — reported as a Warning so it stays conscious.
Writing the rule out exposed that `.Override()` had two inconsistent
stories in the spec; it is now four explicit rules (rebinding keys via
`Mech`; replacing a kit default's behavior; cross-scope shadowing; duplicate
own `id`+scope).

**RC3 — `Mech` semantics.** Ratified: kit mechanics are active by default;
`Mech(id, keys...)` is a **rebinding**, which is an explicit override by
construction (the call *is* the intent — no second `.Override()` marker
needed). Page-step keys are named (`CURSOR_PAGE_UP/DOWN`) and distinct IDs
from `SCROLL_*`, so no collision exists. The catalog now gives every
mechanic exactly one default key — which exposed that `SCROLL_TOP/BOTTOM`
had none; they take `home`/`end` (a tier-2 default, rebinding is allowed).

**RC4/RC5 — audit rules that couldn't work.** Adopted both fixes:
screens declare **`Reaches []string`** (the screens they can navigate to),
making "unreachable" computable as a Warning; the "missing quit path"
warning is **dropped** — pinned `ctrl+c` makes it unconstructible, and a
rule for an impossible state is noise.

**RC6 — pass-through.** Adopted with a real rule, per §3.1's "audited
invariant, not a promise": the audit checks that the wiring declares a host
hook or explicitly opts out, and the test helper asserts that every
unconsumed message class is receipted — pass-through is verified, not
asserted.

**RC7 — pane minimums.** Adopted: panes declare `.Min(w, h)` (audit-time
only, as with screens), and the honest-minimum search is **linear upward**
— rounding makes the minimum non-monotonic in the declared size, so a
binary search can miss.

**RD1 — escape hatch.** Adopted: the hatch is now designed, not asserted —
(a) the host hook, (b) **`tuikit.AsModel(app) tea.Model`**: the composed
kit state as a bubbletea model the app runs inside its own `tea.Program`,
which makes "raw bubbletea is one delegation away" true without a raw-key
pane (that stays deferred, §15.3).

**RD2 — v0.1.0.** Adopted. "Pre-1.0 per Go convention" alongside "tag
v1.0.0, once" was incoherent; the tag is **v0.1.0**.

**RD3 — gates before the tag.** Adopted, superseding the earlier deferrals
— the context changed: this spec moves to a dedicated repo, so the
"consumer beta has the calendar" argument no longer applies. The
outside-user app and the prior-art pass both move **before** the tag, and
the migration metrics get thresholds (§17.2): ≥30% reduction in
UI-infrastructure LOC, zero focus/navigation regressions, ≥80% of actions
covered by interaction tests, add-action = one declaration, add-screen =
one file + one registration, concept count within the day-one budget.

**R7 — palette target timing — partially adopted, with the invariant as
the reason.** The critique proposes snapshotting the target when the
palette opens. That would break §11.1's centerpiece: a keypress at the
selection moment resolves *then*, so a snapshot-at-open palette and the
key path diverge — exactly the lie vector the one-resolver rule exists to
kill. Adopted instead: resolution stays at **selection** (keypress parity),
NeedsCursor palette entries **display the open-time target as a hint**,
execution re-resolves, and the danger confirm displays the execution-time
target (§6.5) — so the operator is never shown one ID and acted on
another. What the kit refuses is executing a stale snapshot.

### 9.4 Spec updates applied

All ratified items are now in `tuikit-spec.md`: corrected status note; §5.1
pipeline reorder with `LineCount(width)`; §5.3 fan-out rules (host hook,
always-receive panes, `Init` on push, Notifier deleted, mutation rule);
§5.4 panic contract; §5.5/§8 capture reconciliation (`CONFIRM_NO` = `n`,
palette never over a capturing modal); §6.1 unexported-field builders; §6.5
pane-scoped recheck; §7 factories + `Push(name, params)`; new modal
contract (§7.5) and scroll section (§9.5); §9.3 layout variants; §9.4 pane
`.Min` + linear search; §10.1 audit-clean day-one example; §10.5 designed
escape hatches; §11.2 revised event schema; §12 audit-rule fixes; §17
v0.1.0 tag, pre-tag gates, thresholds; decision log D7/D9/D13/D14/D16/D17/
D19–D22/D30 corrected in place and D31–D44 appended.

### 9.5 Self-review: what compiling the spec found

Applying the fixes introduced new API surface that had never been through a
compiler, so the spec's code samples were compiled against a type-only stub
of the API (go1.25, `go vet` clean). **It found eleven problems, one of
them a reversal of a position taken earlier in this document.** They are
recorded here because they are the same class the reviewer keeps finding,
and because the fix for that class is procedural, not editorial.

| # | Found | Fix |
|---|---|---|
| 1 | The replacement day-one example was itself audit-invalid: `drill` sat at screen scope but reads the cursor | `drill` moved to pane scope; the screen-scope verb is now `layout` (a variant toggle), which also exercises §9.3 |
| 2 | `Variant` was both a type and a constructor — the exact clash reported as blocker 1, reintroduced | placement types are unexported; apps chain and pass them |
| 3 | `NewPane` had no type it could return: `Pane` is the interface app structs implement | **the reviewer's `Leaf` was right** — adopted as the constructor; the earlier rejection of it (§8.2, "worse for grep") was wrong for a concrete reason and is withdrawn |
| 4 | `Scroll.mode` was unexported, so a pane embedding `Scroll` by value could not choose its mode — the "explicit mode" requirement was half-undone | `Mode` is the one exported field, set at declaration |
| 5 | Effects returned `tea.Cmd` in one sample and nothing in the API list | effects are statements; a handler's only return is its `tea.Cmd` |
| 6 | `NewScreen(name, min, variant, keys)` and `NewApp(screens…, actions…)` were not type-expressible as written | typed options and builders: `Root`/`Detail`/`Global`, `.Select`/`.Keys`/`.Reaches` |
| 7 | `SCROLL_TOP/BOTTOM` had no default keys | `home`/`end` |
| 8 | Within-scope duplicate keys were an Error in §12 and "legal precedence" in §6.4 | Warning |
| 9 | The "focus-ring count ≠ focusable panes" rule was unconstructible once focus re-derives per frame | dropped — the same logic as RC5, applied to our own rule |
| 10 | The record type was `Receipt` in the type list and `Event` in the schema | type is `Event`; "the receipt stream" names the stream |
| 11 | The prior-art differentiation was stated as fact on secondary evidence | hedged, and made a gate: step 7 confirms or revises it before the tag |

What the compile check verified positively: every `tuikit.*` name the spec
uses exists in the stub; `KitCursor()` is discoverable through an embedded
`Cursor` and a method named `Cursor()` is not (the negative control from
the first review); all 31 section cross-references resolve; the decision
log is continuous D1–D44.

**The procedural fix is now in the spec, not just in this response:** the
status block requires that every code sample compile against the stub
before the document changes (D44). Step 0 replaces the stub with a real
implementation. Review caught the first two rounds of this class; the
compiler should catch the third.

### 9.6 Next

Per the review's own sequencing, adopted as the work order. **Blockers 1,
2, 5 are closed** — small edits, and their samples now compile. **Blockers
3, 4, 6, 7 are ratified in the spec and compile against the stub, but must
be settled in real code by step 0** — the day-one example with async
refresh, filter input, and drill-down exercises the sync ordering, modal
contract, factory registry, and layout variants together; any contract step
0 cannot express goes back through review before the freeze.

---

## 10. Response to the third review of tuikit-spec

> The reviewer's headline: "the best draft yet, nearly ready for step 0 —
> fix items 1–3 on paper, then stop reviewing and build step 0." That is
> adopted as written, with one deliberate deviation: **all** items were
> applied on paper, not only 1–3, because a document that knowingly
> contradicts itself cannot be the source of truth that step 0 is built
> against. What the reviewer asked to be settled by tests is marked that
> way — each is ratified text plus a named test in §17.1 of the spec, and
> none is claimed as proven.
>
> **Verdict: all 22 findings adopted** (25 counting the four modal
> sub-parts separately), four of them extended beyond the letter. No
> pushback this round — every finding reproduced on inspection.

### 10.1 The claim that was false, and why

The reviewer was right that "every code sample in this document compiles"
was false as printed, and the truth is worse than they stated:

| Defect as printed | Found by |
|---|---|
| §10.2 `Render`: `declared and not used: i`, `missing return` | reviewer (confirmed) |
| §10.2 mixes `tuikit.Cmd` and `tea.Cmd` | reviewer (confirmed) — **introduced by my own final edit, after my compile check had run** |
| §10.1 is not a compilable unit: bare statements at package level | extraction |
| §10.2 calls `idsOf`, which the document never defines | extraction |

The root cause is not a typo. My previous check compiled a **retyped,
patched** version of the samples against a stub whose `tea.Model` was also
wrong. It verified that my transcription compiled, not that the document
did. The fix is procedural and now stands in the spec (D44): the day-one
program and the catalog's default keys are **extracted from the document by
script** and checked against the **real** bubbletea v2, and the check runs
on the final text.

### 10.2 What checking against the real library found

Checking against the real library, instead of against the stand-in types
my earlier harness used, surfaced four v2 behaviors the contract depends
on. All were read from the v2.0.9 source and are now in §5.6:

| v2 fact | Why it is load-bearing |
|---|---|
| The page-down key spells **`pgdown`**, not `pgdn` | the catalog bound `pgdn` — a default that could never match a real press. Caught by checking every default key against `Key.String()` |
| `tea.KeyReleaseMsg` also satisfies `tea.KeyMsg`, with the **same string** as the press | on terminals that report releases, every bound key would have fired twice. Releases are now dropped by type before routing |
| `Model.View()` returns `tea.View`, a struct | `AsModel` and the kit's `View` were specified against a string |
| An interrupt delivered as a signal ends the program inside bubbletea, before any model sees it | the pinned `ctrl+c` has a second, unblockable exit path — good news for liveness, but it belonged in the contract |

### 10.3 Two defects in the check itself

A check that passes means nothing until it is shown to fail. I mutated
copies of the document and ran the check on each. It caught the `pgdn`,
`tuikit.Cmd`, and `Render` defects — and it exposed two ways the check
itself could report success wrongly:

1. **A compile failure printed `OK`.** Under `set -e`, a failure in the
   first element of `a && b && c` does not abort the script. The steps are
   now separate statements.
2. **A stale `ok` for a changed document.** Go's test cache keys on the
   *value* of the environment variable naming the spec, not on the file's
   contents — so a changed spec at the same path was reported with the
   previous spec's result. In CI, where the path is constant, that is
   exactly how a drifted spec keeps a green check. The spec checks now run
   with `-count=1`.

The second is the more serious: it is the failure mode the check exists to
prevent. Both are fixed, and the mutation run now reports fail for every
known defect and pass only for the clean document.

(One further anomaly was my own doing, not the harness's: I issued a file
write and the run depending on it in the same parallel batch, so the first
run executed a half-written script.)

### 10.4 The six "fix before step 0" items

**1. The host hook can break the pinned exits — adopt.** The hook saw every
message first and could consume `ctrl+c`, `esc`, or a resize. New order
(§5.3): **pinned keys and geometry are handled before any app code runs**;
the hook may observe a resize but cannot consume it. Where other keys meet
the hook was left open by the reviewer; decided: **after the table**, so a
bound key always does what the table says and help cannot lie. The hook
sees only unbound keys — which makes it an app-level raw-key path without
power over declared verbs.

**2. Cached keys don't work with factories — adopt.** `p.kill` is bound to
one instance; a table cached from the audit's instance would act on the
wrong pane. `Keys()` is now called **once per instance, at creation**, and
every instance must declare the **audited shape** (per pane: ordered
`(id, keys, NeedsCursor, Danger)`, plus which mechanics discovery grants).
A difference is fatal at wire-up and a refused push at runtime. Help and
title may differ; they are display text.

**3. `unhandled` can't be detected — adopt, with one rule of mine.** A pane
returns only a command, so `nil` is ambiguous. `unhandled`, the pass-through
audit, and `WithoutHost()` are gone; delivery is **recorded, not judged**.
The reviewer asked that spinner ticks not flood the buffer and left the
mechanism open; decided: a `delivered` event is re-recorded **only when the
recipient set changes**, so a 10 Hz tick holds one slot. This is flagged in
the spec as an open question for step 0, not a settled design.

**4. Confirm can recheck stale data — adopt, generalized.** Hidden panes
stop syncing, so "latest declaration" could be old. Rather than special-case
confirm, a single rule (§6.5): **every** `NeedsCursor` dispatch — key,
palette selection, or `CONFIRM_YES` — calls `ItemIDs()` on the originating
pane immediately before resolving, visible or not. One rule instead of a
special case; the named hidden-pane-confirm test proves it.

**5. Modals — adopt all four parts.** (a) Declared modal keys win, then the
rest is text — a filter binds `enter` and still receives `?` as typed text.
(b) Modals are registered as **named factories** and opened by name;
`ModalResult{Name, Value, Canceled}` goes **only to the opener** — the one
exception to always-receive — so the opener needs no correlation ID. (c) The
audit can now instantiate every modal and check its keys and capture mode.
(d) `Init` runs at open, for cursor blink. Extended: a modal opened by
another modal receives that modal's result; a result with no opener is
receipted and dropped.

**6. Instance lifetime — adopt.** Roots persist and `Switch` shows the
existing instance; details are built and initialized at push and discarded
at pop; the instances the audit builds are never initialized, updated, or
rendered.

### 10.5 The correctness table — all adopted

| Gap | Resolution |
|---|---|
| Solver checked only at `Min` | every size from `Min` to a bound is solved; honest minimum = smallest size above which all pass (§9.4) |
| Dry-run mutates | layout → geometry → sync run on **copies**; nothing stored, moved, or recorded (§13) |
| `Route` gives no state | options `At`/`On`/`Focused`/`WithModal`, defaulting to the app's current state; headless size derived from the roots' `Min` |
| Headers hide the selection | optional `Metrics{Reserved, RowHeight}`; a pane draws a header or multi-line rows only by declaring it (§9.5) |
| Who stores scroll position | sync is the only writer; `Offset()` takes no argument and only reads |
| Which panes get which mechanics | by discovery: `KitCursor()` → `CURSOR_*`, `KitScroll()` → `SCROLL_*`, both → Error, every pane → `FOCUS_*` |
| Undeclared `Push` | `Push` and `Switch` refuse **unregistered and undeclared** targets; screens declare `Reaches`, the app declares it for global scope (§7.3) |
| Palette hint fixed at open | the hint is derived on every render, never stored |
| `enter` drills and confirms | confirm grammar is `y`/`n` |
| Panic under `AsModel`, batched commands | commands are wrapped (`tuikit.Batch`); `Run` returns an error; `AsModel` guarantees recovery and the receipt, not the exit code |
| `Debugf` order | ordered assertions cover `dispatch`/`lifecycle`/`sync`/`audit` only; `debug` and `delivered` are matched, never ordered |

The reviewer's own "linear search" advice was withdrawn by the reviewer as
incomplete; the sweep replaces it.

### 10.6 Document defects — all adopted

Day-one example now has two variants (`layout` really toggles one),
declares a host with `tuikit.WithHost`, and shows the async path
(`doRefresh` returns a command). `GLOBAL_DEBUG` is removed — no key is
reserved for an unbuilt overlay. "Danger action without confirm compile" is
dropped as unconstructible. `--receipts` is replaced by
`tuikit.ReceiptsTo(w io.Writer)`: a library option, so the app owns the flag
and startup events survive the ring buffer's bound.

### 10.7 What applying the review turned up

Three holes the review did not list, found while making its fixes fit
together:

1. **No effects from message handlers.** Only key handlers received a
   `Ctx`, so a kit-owned app could not navigate or open a modal *in response
   to a message* — a basic need, and a gap since `kit.Run` took ownership of
   the model. Every app callback now receives the same `Ctx`
   (`Init(c)`, `Update(c, msg)`, `Handle(c, msg)`); effects called during
   fan-out are queued until it ends.
2. **A fourth scope that was routed but never compared.** Modals dispatch in
   modal scope and receipts record `scope: modal`, yet the spec said "three
   scopes, forever" and compared shadowing across three tables. A modal's
   keys were checked by nothing.
3. **Unconstructible values in the event schema.** `trigger: program` could
   not be produced by anything. Removed — the reviewer's own test, applied
   to my schema.

One question deliberately left open rather than invented (§17.1): a popped
detail is discarded, but a command it started keeps running. Whether a
teardown signal is needed is for the two-detail-instances test to show.

### 10.8 Decisions that go beyond the letter of the review

| Decision | Why it was needed |
|---|---|
| `Ctx` passed to every callback; effects queue until fan-out ends | the hole in 10.7.1 — a contract-shape question, which is the reviewer's own criterion for settling on paper |
| Unbound keys reach the host hook after the table | the reviewer left "where other keys meet the hook" open |
| `delivered` re-recorded only when recipients change | the reviewer left the anti-flood mechanism open |
| Size-sweep bound defaults to 480×200 | a number was required; none was given |
| `Metrics`, `At`/`On`/`Focused`/`WithModal`, `KitScroll` | names for interfaces the review asked for |
| New audit rules: `Mech` for a mechanic the pane does not receive; `NeedsCursor` in a pane without a cursor; one pane name bound to two instances across variants | consequences of discovery and variants that were otherwise unchecked |

### 10.9 Where this leaves the document

The spec's status is now **"ready for step 0 — review closed."** Its own
claim about itself is narrower and true: the day-one program and the
catalog's default keys are extracted from the text and checked against
bubbletea v2.0.9; everything else is ratified text, and the sections the
reviewer named are marked as freezing only when their tests pass.

The document grew by about 270 lines this round. Every addition traces to a
finding above, but it is longer than "concise" asked for; a pass that
removes restatement without removing rules is available on request.

The harness lives outside the repo at
`/tmp/opencode/speccompile/` (`run.sh <spec>`); it is the CI job that
§17.1 asks step 0 to add, and it should move to the new repo with the spec.

---

## 11. Response to the final review of tuikit-spec

> Verdict: **all items adopted and applied as one final patch.** The spec is
> now ready for the step-0 implementation plan. Each change is logged as a
> named test in its slice (§17.1 of the spec), which is what D47 asks of a
> patch applied without a failing test first.

### 11.1 Day-one example — fixed

| Defect | Fix in the spec |
|---|---|
| `Row`/`Cols` directions undefined; one variant fails under either reading | §9.1 now defines them: `Row` stacks top to bottom, `Cols` places side by side. `plain` uses `Cols` |
| Chrome costs had no values | §9.2 is a table: pane border 2 rows + 2 columns, titled group the same, title text 0, footer 1 row, notice 1 row, overlays 0 |
| `ui.layout` started empty | Initialized to `"plain"`; the selector's first answer is audited, and an unknown name at runtime falls back to the first variant |
| `ReceiptsTo(os.Stderr)` corrupts the screen | The sample writes to a file; §14 says why |
| `Render` ignored `Offset()` | The sample windows from `p.Cursor.Offset()` |

Also added in §9.1, since geometry cannot be computed without it: a
constraint sizes a node's outer box (border included), a pane's `Min` is its
smallest slot, and several `Fill`s share equally with the leftover cell
going to the earliest.

### 11.2 The five gaps — decided

1. **Effects belong to the caller (§7.2).** Stack and focus effects are
   accepted only from the component that owns input, and act on its own
   context. `CloseModal` is the top modal closing itself. A table lists
   which caller may call which effect.
2. **Callers other than actions (§7.3, §7.5).** The opener is whichever
   current component called, from a handler, `Init`, or `Update`. A modal
   inherits its opener's `Reaches`; kit digit jumps are exempt.
3. **Paste follows capture (§5.3).** With a text-capture modal on top, a
   paste goes only to that modal.
4. **Palette and help have keys (§6.4, §11.1).** Palette: `up`/`down` move,
   `enter` selects, the rest is filter text. Help: `up`/`down`,
   `pgup`/`pgdown`. The palette closes before dispatching, with the original
   pane as caller.
5. **Confirm keys are fixed (§6.4).** `y`/`n`, and no longer called
   overridable.

Two audit rules added in §12: a `Cursor` pane must declare `ItemIDs()`, and
a `Scroll` pane must declare the method its mode needs.

### 11.3 Plan ordering

§17.1 is now the five slices proposed (0a–0e), each building what its tests
need, with Effect ownership in 0c and Paste routing in 0d. Step 3 is
restated as *completing* the router, audit, and `tuikittest`.

### 11.4 Choices I made that were not in the review

| Choice | Why |
|---|---|
| `with-log` keeps queue and detail where `plain` has them, with the log under the detail | The review only asked for `plain` to use `Cols`; the alternative moved the queue from a top band to a left column on toggle |
| `Min` is `51×14`, not `100×20` | With borders and an equal `Fill` split, 100×20 over-promises. 51×14 is the honest minimum |
| `Quit` and `Notice` are exempt from the ownership rule | They act on no stack; a background pane must be able to quit on a fatal error |
| The notice line is always reserved | Deterministic geometry; a row that appears on demand would shift the layout |
| `Variant` takes a single `Leaf` or a `Layout` | A one-pane screen otherwise needs a wrapping group |

### 11.5 What was verified

Running the audit on the example corrected my own arithmetic: I had
predicted a honest minimum of 88×14 by hand. It is 51×14. That is the case
for the audit being a program rather than a paragraph, and it is the only
reason the harness was worth building. It was run once on this exact text:
the sample compiles against bubbletea v2.0.9, every catalog key is a real
v2 spelling, and the audit is clean on the documented screen.
