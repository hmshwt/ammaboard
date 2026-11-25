# AmmaBoard — Complete Critique Compendium

**Date:** November 25, 2026
**Scope:** All critiques of the AmmaBoard project across the full discussion — two pre-pivot versions (initial exploration, then a verified deep dive), one post-pivot critique after the strategy change, and the further steps to alpha.

**Verified project state at time of critique:**
- 10,460 LOC of Go; all tests pass (`go test ./...`, 13 packages, ~5s); `go vet` clean; working tree clean; zero untracked files.
- Last commit: **Oct 20, 2026** (36 days of silence as of this document).
- Remote: `github.com/hmshwet/ammaboard`, single author identity (Himshwet Gaurav, 18 commits).
- Machine core (isa 533 + asm 400 + machine 2,584 + ticklog 717 + cmd 664 ≈ 4,900 LOC, 47%), `internal/tui` 1,245 (12%), `internal/tuikit` + `testrender` 3,661 (35%), dead `internal/ui` 1,070 (10%).
- Batch CLI, all three examples, timer interrupts, and the Blue Screen path verified working by direct execution.

---

# Section 1 — Pre-Pivot Critique

*(The original goal: a Dec 1, 2026 beta of an advent-calendar course — 24 doors, 16 chapters, date-gating, primers.)*

## 1.1 Version 1 — First-Pass Critique (Initial Exploration)

### Usability

**Strengths:**
- **Clean CLI**: `ammaboard run` and `ammaboard tui` commands well-designed with intuitive flags (`-echo`, `-dump-full`, `-max`).
- **Good error handling**: clear errors for unknown opcodes, unaligned access, budget exhaustion.
- **Determinism by construction**: single-threaded, phase-stepped design guarantees it — same inputs = same event streams.
- **Pluggability proof**: the echo device demonstrates new hardware needs only one type + one `Attach` call, zero edits elsewhere.
- **Complete event logging**: every interaction lands in SQLite + readable `trace.txt`; faults and execution fully traceable.
- **Blue Screen**: excellent fault rendering — what went wrong, where, why — with register/memory dump and cause chain.

**Issues:**
- **TUI requires TTY**: fails without a terminal (`bubbletea: could not open TTY`), limiting development and demos in headless environments.
- **No persistence across runs**: the SQLite DB is rebuilt each run; no saved state or session recovery.
- **Limited addressing**: 16KB RAM hard ceiling; restrictive for complex programs (reasonable for educational scope).
- **No assembler error recovery**: errors cause immediate exit; no "best effort" assembly or partial results.
- **TUI keybinding gotchas**: e.g., space produces `KeySpace` (String "space", Text " ") rather than plain " " — traps for contributors.
- **No chapter/door system yet**: the advent calendar / chapter engine is planned but not implemented.
- **Limited documentation interaction in TUI**: `help` exists but the Blue Screen overlay is deferred — faults show in state line + red log lines only.

### Feasibility

**What works well:**
- ISA design: 20 instructions with consistent encoding, clean and extensible.
- Motherboard broker: phase-stepped single-threaded design, elegant and deterministic.
- Component model: bus-only interaction via `ReadBus`/`WriteBus` with window routing, well-enforced.
- Timer + interrupts: level-triggered, shadow registers, trap vector 0x0008 — all correct.
- Parity suite: assembler ↔ ISA ↔ CPU conformance tests provide strong guarantees.
- Tick log: SQLite-backed event logging, lightweight and queryable.
- Blue Screen: comprehensive, informative fault rendering.

**What's tight / risky:**
- TUI complexity: bubbletea + tuikit combo is ~1,200+ lines with intricate geometry/focus/modal logic; resizing, key-routing, or modal-stacking bugs would be hard to track down.
- 16KB memory ceiling: hardcoded `ramSize = 16 * 1024` and `ramBase = 0x0000`; larger memories require sizing changes.
- No chapter engine: without chapter files/payload format, the advent calendar concept can't be realized — **the biggest gap**.
- Go 1.25 module: cutting-edge; toolchain compatibility could become an issue.

**Overall feasibility:** **High** — core machine simulator solid, well-tested, extensible; TUI needs care; chapter engine is the main missing piece.

### Improvements Suggested (v1)

**Architecture/Design:**
1. **Chapter payload format**: define a data-driven chapter format (12-byte LE `origin, length, entry` header per D21) — enables the advent calendar system.
2. **TUI headless mode**: render to JSON/CSV instead of requiring a TTY — enables CI testing and non-interactive demos.
3. **Session persistence**: save machine state (registers, memory, PC) alongside the tick log for post-hoc inspection/resumption.
4. **Configurable RAM size**: CLI flag or config instead of hardcoded 16KB, keeping the "sizing rule" a documentary constraint.

**Usability:**
5. **Assembler `--lenient` flag**: output partial assembly with warnings instead of hard errors.
6. **Label-based addressing documentation**: the `li` sugar is absent but the immediate supports absolute byte labels — document the idiom more clearly.
7. **TUI minimum size enforcement**: runtime `TooSmall` notice when terminal is undersized (141×28 constraint).
8. **Speed control UX**: numeric `speed N` command for precise control beyond ×2/÷2.

**Approach:**
9. **Decouple tick log from CLI**: ensure no circular deps (already well-structured, keep it so).
10. **Document form-driven assembler growth**: adding instructions = `isa.Specs` row + CPU file + tests, no core changes — document explicitly for contributors.
11. **Blue Screen first**: port the existing CLI BSOD renderer to the TUI as the first overlay (per roadmap M2) — immediate fault feedback in the interactive path.

### Next 5 Steps Proposed (v1 — for the course-product alpha)

1. **Freeze event vocabulary & build golden trace fixtures** — M2 marked "next"; run all chapter programs, capture `SELECT * FROM events` as golden fixtures; freezes D13–D27 and enables the parity suite to catch drifts.
2. **Implement chapter engine + payload format** — 12-byte LE header (`origin, length, entry`) + program bytes; `internal/chapters` package loading chapter definitions (program + expected trace + primer + links); `run` validates against golden traces.
3. **Port Blue Screen to TUI** — existing CLI renderer as TUI overlay; keys `t` jump to tick log at fault, `r` rewind/replay, `q` unfreeze.
4. **Add chapter 1–2 prototypes + screenshots** — plugboard simulation + program shuffling; generate terminal screenshots for launch (Show HN, r/programming, AoC-adjacent); "Day 1 is the storefront."
5. **Add `ammaboard tui --check` wiring audit** — `AssertCleanWiring` test helper as a `--check` flag; pure audit of keymap conflicts, focus-ring counts, frame geometry at min dimensions; fail-fast before `tea.NewProgram()`.

**Additional recommendations (v1):** Consider Go 1.24 pin for broader toolchain compatibility; add `--free` flag to bypass date-gating during testing; create a `review.md`-driven CI check validating golden-trace diffs on every PR; document the `li` idiom (the immediate ceiling covers the whole 32KB space).

> **Note:** This first pass echoed the README's "18-bit immediate" claim — that was stale; the code truth is a **16-bit signed immediate with a 7-bit opcode** (see D40). Corrected in Version 2.

---

## 1.2 Version 2 — Deep-Dive Critique (Verified, Honest)

*(After examining git history, the committed `review.md`, per-package line counts, doc-vs-code drift, and running everything.)*

### The Fact That Reframes Everything

**Last commit: Oct 20, 2026. Today: Nov 25, 2026 — 36 days of silence.**

The roadmap (status dated 2026-10-17) targets **beta Dec 1, 2026 — six days away** — with milestones M2 (Nov 8), M3 (Nov 15), M4 (Nov 20), M5 (Nov 24). **Every milestone date passed with zero commits.** Missing per the project's own status table: chapter engine, progress save, date-gating, release build, stage-3 hardware, payloads + acceptance checker, TUI chapter views, all primers — i.e., the entire course. Even the pre-planned escape hatch ("ship doors 1–8 polished") is gone — zero doors exist.

The Oct 16–20 cadence (20 commits in 5 days) is the sprint cadence that builds machines but cannot sustain content. The honest conclusion: **the Dec 1 beta is not salvageable** — the useful deliverable is a re-entry plan, not a plan pretending the calendar didn't happen.

**Correction to v1:** the README's "18-bit immediate / 5-bit opcode" is stale; since D40 the encoding is `[op:7 | rd:3 | rs:3 | rt:3 | imm:16]`. The README is wrong about the project's own ISA — part of a doc-drift pattern below.

### Usability (verified by running it)

**Genuinely good — better than most course-ware ships at 1.0:**
- The batch CLI is a pleasure: verdict line, state dump, two diffable artifacts.
- The **Blue Screen is the best artifact in the repo** — plain-language cause, the disassembled offender, memory around the fault, the cause chain, a pointer into the trace. The emotional signature of the whole course, and it already works.
- The stop model vocabulary (`halt` / `fault` / `budget_exhausted` = pause) is honest and consistently rendered everywhere.
- Trace readability decisions (D24–D26: per-cycle comment lines, `(was N)`, every numeric interpretation) show someone who has watched a learner actually read a trace.

**Problems a first-time stranger hits:**
1. **The README misstates the ISA** — shows `[opcode:5 | rd:3 | rs:3 | rt:3 | immediate:18]`, the encoding abandoned in D40. First thing a Show HN reader sees. Brand damage; a 30-minute fix.
2. **TUI minimum 141×28** is larger than most default terminal windows (80×24). The TooSmall notice is graceful, but a stranger's first run is likely a refusal. README needs a warning; better, reflow at ~100×30 before launch.
3. **The shell can't load a program** — `run · step · reset · speed · quit · help` is fine for a demo, but iterating on an exercise means restarting the binary per attempt. `load <file.amb>` is the highest-value unbuilt TUI command (ui.md already promises it).
4. **Artifacts land in the CWD** — `ammaboard.db` + `trace.txt` scatter wherever the user is. Default to a `runs/` directory or `$TMPDIR`; keep `-o`/`-trace` overrides.
5. **No onboarding moment** — nothing explains the three panes. One README GIF of the TUI plus one of the BSOD would carry the storefront.

### Feasibility

**The machine: done, and done well.** By the project's own estimate the machine is ~30% of the work, and it is *finished*: deterministic by construction, 20 instructions with real parity guarantees, level-triggered interrupts with re-sampled delivery, a stop model that survived a hostile external review — most of `review.md`'s verified defects were genuinely fixed via D28–D34. That response discipline is rare and commendable.

**The course: 0%.** By the project's own risk register, content is ~70% of the work. The project is ~30% done, not ~80% — the machine *feels* like more because it is the hard, fun, verifiable part.

**The codebase shape tells the real story:**

| Layer | LOC | Share |
|---|---|---|
| Machine core (isa, asm, machine, ticklog, cmd) | ~4,900 | 47% |
| `internal/tui` | 1,245 | 12% |
| `internal/tuikit` (+ `testrender`) | 3,661 | 35% |
| `internal/ui` — **dead**, superseded by D41 | 1,070 | 10% |

**Over half the repository is UI infrastructure. The TUI framework (`tuikit`, 3,247 LOC) is larger than the entire CPU + ISA + assembler + RAM + timer + echo combined** — built for a course with zero chapters to display. The engine and chassis are beautiful; the product — the thing a learner pays attention to — does not exist in any form.

**Timeline math:** alpha = chapter engine + one complete door + BSOD-in-TUI + release build. At the demonstrated Oct pace, ~1.5–2 weeks of focused work. At a realistic part-time re-entry pace, 2–3 weekends per step — **~3–4 weeks to a defensible alpha**. A Dec 1 beta would require all of that *plus* 8 doors of content in 6 days — not with a solo dev.

**Honest recommendation:** slip publicly. Nobody is waiting yet — zero users makes this the cheapest slip that will ever be announced. Take the time to make the first doors flawless. "Never broken doors" was always the policy; the only new decision is the date.

### What to Improve / Approach Changes (v2)

1. **Break the framework loop — hard freeze on `tuikit` now.** D41 built it; D43–D45 then added an audit engine, a golden-render test framework, a receipt ring, and an upward emission channel. All excellent engineering; all pure gold-plating relative to the #1 risk ("content velocity is the project"). The previous external review already said *"M2 should produce one chapter end-to-end, not generic pane machinery"* — and the next three weeks went to pane machinery. The kit is good enough; hack around its limits in `internal/tui`; do not touch the kernel again before launch.
2. **Build the course in reverse.** Door 1 (plugboard) and door 2 (deck reorder) are bespoke simulations sharing nothing with the machine — the *least* reusable work, on the critical path. Door 4 ("hand-assemble + run the boot ten") reuses every existing asset: assembler, machine, TUI, BSOD. Build **door 4 first** as the pipeline prover, then backfill 1–2 (or rescope them as intro interactions on the machine itself — nobody has pre-ordered the plugboard).
3. **Write the three one-pagers open since October.** Chapter file format (#1 open decision, blocks everything — the contract between the engine you have and the content you don't), day-5 editor input path, day-11 walker convention. All three are *prose*, not code. A 5-week stall is exactly what unspecified seams cause — the next commit becomes hard to choose.
4. **Kill the dead code.** `internal/ui` (1,070 LOC) superseded by D41, imported by nothing, still running in the test suite — delete it; the decision log records why. Same sweep: `counterDemo` duplicated in `cmd/ammaboard/tui.go` ("keep in sync by hand" — a future bug report) → `go:embed` the example; the `machine.Event`↔`ticklog.Event` conversion in `readEvents` (flagged in the project's own review, still present); fault-kind casing still mixes `alignment` with `bus_NOT_CONNECTED`.
5. **Add CI — 15 lines.** A project whose entire brand is determinism and parity has no `.github/` directory. `go vet + go test ./...` on push. The parity suite is the best asset; make it enforce itself.
6. **Treat docs staleness as a defect class.** README encoding drift, broken archive links (`docs/motherboard.md`, `docs/PLANNING.md`, `docs/detailed-plan.md` all actually live under `docs/archive/`), `docs/README.md` still saying "D1–D36" (now D45), roadmap status frozen at 2026-10-17. Extend the constraints-with-names culture with one rule: *a change that changes behavior updates the doc in the same commit.*

### Next 5 Steps Proposed (v2 — pre-pivot alpha)

**Alpha definition planned against:** a stranger downloads one static binary for their OS, opens a real door with a primer and a machine-checked exercise, steps through their program in the TUI, crashes it and gets the Blue Screen *in the TUI*, and their progress persists.

1. **Re-baseline the plan in writing (half a day).** Update `roadmap.md` to reality: all milestone dates passed, last commit Oct 20, new alpha definition, new date (~3–4 weeks out). Decide the public posture (quiet slip — no waiting audience, which makes it free). Update the stale docs/README table and the roadmap status date. *A decision, not code — everything hangs off it.*
2. **Write the chapter-format one-pager + the other two seams (half a day).** Chapter = `id/day/title/primer(md)/payload(words+entry)/machine-config(wiring)/acceptance(golden events or final-state predicate)/links`. Engine reads it; acceptance checker executes it; TUI renders the primer. No code yet — prose, one page, reviewable in 20 minutes. The #1 open decision since Oct 17; it gates everything.
3. **Minimal `internal/chapters` + door 4 complete (1–2 weekends).** ~300–500 LOC given everything else exists: loader, acceptance check (run payload → diff final state / golden event stream), and one *finished* chapter — day-4, primer written, payload built, golden trace committed, acceptance passing in CI. Add `load <file.amb>` to the TUI shell in this step. Proves the pipeline all 15 remaining doors flow through — the single most valuable artifact the repo is missing.
4. **Blue Screen in the TUI (1 weekend).** `tuikit` already has `FullBleed` screens designed for exactly this ("the fault screen: shell hidden, its own `r` → reset" — D42), and the renderer already exists in `internal/machine/bluescreen`. Wire the fault screen: cause-first layout, `j/k` scroll, `t` jump to tick log at the fault, `r` reset, `q` unfreeze. The CLI panel is the course's signature moment; making it live in the interactive path is the highest emotional-value alpha feature.
5. **Hygiene + release sweep (1 weekend).** Delete `internal/ui`; fix README (encoding, archive links, terminal-size warning, a GIF); fix the small defects (counterDemo embed, fault casing, event conversion); add GitHub Actions CI; add date-gating skeleton with `--free` (a date check in the chapter loader is enough for alpha); progress save (a tiny JSON next to the binary); `CGO_ENABLED=0` cross-build via goreleaser or a Makefile (modernc sqlite makes this trivial). Then **hallway-test with 3–5 devs who've never seen it** on door 4, fix what confuses them, and call it alpha.

**What NOT to do in the alpha window:** no stage-3 hardware (MMU/TLB — door 10–11 content isn't close); no new ISA instructions; no further `tuikit` work; no doors 1–2 bespoke sims; no leaderboard/shared anything.

**Bottom line (v2):** the machine is trustworthy; the engineering culture (decision log, parity tests, review response) is genuinely better than most funded projects; and the product doesn't exist yet — not because the wrong machine was built, but because a *second* machine (a UI framework) was built when the first was already done. The course is 70% of the work and 0% of the code. The five steps get to a real alpha in roughly a month; the only unrecoverable thing is the Dec 1 date — better to say so now, while nobody is watching.

---

# Section 2 — Post-Pivot Critique

*(The pivot: alpha = explain AmmaBoard's architecture and extensibility — no advent toy, a date-free roadmap of chapters to come — and extract the TUI framework (`tuikit`) as a separate portfolio project for job interviews. Launch surfaces: polished repo, GitHub Actions builds, LinkedIn post, Show HN post, blog post.)*

## Reading the Pivot

The old alpha (course product) required the missing 70% — content, chapters, acceptance. The new alpha (architecture showcase + framework extraction) requires only **presentation, packaging, and communication** of work already done and tested. That is the difference between a death march and ~2–3 part-time weeks.

But the pivot inverts what matters: **the repo itself is now the product.** Every stale doc, dead package, and broken link that was a "margin note" during course development is now a product defect, because both audiences — a Show HN stranger and a hiring engineer — judge the repo in under five minutes.

**Extraction verification (done before critiquing):** `tuikit` is genuinely self-contained — its only internal reference is `testrender`, which moves with it; dependencies are bubbletea/v2, lipgloss/v2, and stdlib only. D41's fence held. The remote already exists (`github.com/hmshwt/ammaboard`) with a consistent author identity.

## A. ammaboard Is a Workspace, Not Yet a Showcase

1. **The README misstates the ISA.** Still documents `[opcode:5 | … | immediate:18]`; the truth since D40 is `[op:7 | rd:3 | rs:3 | rt:3 | imm:16]`. In an architecture-first repo, the encoding line is the single most-quoted fact on the page. **Fix #1.**
2. **Three broken doc links** in README (`docs/motherboard.md`, `docs/PLANNING.md`, `docs/detailed-plan.md` — all actually in `docs/archive/`), and `docs/README.md` still advertises "decisions D1–D36" (it's D45) and omits `tuikit.md` from its table.
3. **`review.md` sits at the repo root, committed.** It's a stale defect review; a stranger will read its "verified defects" as *current* bugs — when in fact D28–D34 fixed them. Nothing in the repo says so. This file actively hurts in an interview walkthrough. Archive it with a one-line header ("superseded by D28–D34") or move it to `docs/archive/`.
4. **Dead code contradicts the architecture story.** `internal/ui` (1,070 LOC, superseded by D41/D42) is still present and still running in the test suite — ~10% of the codebase. Delete it; the decision log already records why.
5. **`counterDemo` is duplicated** in `cmd/ammaboard/tui.go` with a literal "keep in sync by hand" comment — a visible defect class in the first file a reader opens after `main`. `go:embed` the example.
6. **The repo documents its own missed deadlines.** `roadmap.md` status is frozen at 2026-10-17; `project.md` still narrates "launch Dec 1 2026 = beta" as the plan. Under the pivot these become the *vision* — rewrite them **date-free** ("chapters to come", curriculum as design), or the repo's own planning docs read as evidence of a slipped project. The blog can tell the honest "I re-scoped" story; the repo docs shouldn't have to.
7. **No CI, no releases, no tags, no badges.** The first thing an engineer checks after the README.
8. **Small UX care-flags interviewers notice:** run artifacts (`ammaboard.db`, `trace.txt`) dump into the CWD; the TUI demands 141×28 (fine, but warn in the README); fault-kind casing mixes `alignment` with `bus_NOT_CONNECTED`.

## B. tuikit: Clean Boundary, Wrong Shape for Its New Job

The code is ready to leave home. The *presentation* isn't:

- **Its docs are internal decision records, not a framework README.** `docs/tuikit.md` and `internal/tuikit/ARCHITECTURE.md` assume ammaboard context and reference D-numbers from another project's decision log. A hiring engineer gives it five minutes: pitch → diagram → quickstart → features → "why not raw bubbletea?" → tests. None of those exist yet.
- **Its headline features are genuinely differentiating** — the declarative wiring audit (`Verify`/`Issue`), the receipt ring, golden-render visual tests, keymap scopes with `NoShadowing`. These are the interview story, and they're currently buried deep in a decision log.
- **No self-contained example.** After extraction, the only real consumer lives in a different repo. A tiny demo app (`examples/` in the new repo) is what makes a framework evaluable alone.
- **Becoming public raises the API bar.** ~40 exported symbols were named "as if third parties will read them" — now they actually will. Needs: doc comments on every export, a `CHANGELOG`, semver discipline, and the **"kit never imports bubbles; hosts may"** boundary stated in the README (`internal/tui` imports `textinput` directly — the correct pattern; document it).
- **Expect the interview question and answer it in writing:** *"why build a framework instead of using raw bubbletea/lipgloss?"* The real answer is strong — the layout contract leaked into arithmetic (`height-9`), help text drifted from behavior, overlays grew an if-chain — but it exists only in D41's rationale. Move it to the README's "Why" section; that paragraph will do more interview work than a thousand lines of code.

## C. The Story for Strangers

- **What pays off in ten seconds, in order:** the Blue Screen (`go run ./cmd/ammaboard run examples/bsod.amb`), "every tick is a row in SQLite," the determinism test, the `-echo` pluggability proof.
- **What can't be shown in text:** the TUI — it needs GIFs (vhs, from the same Charm stack, is the obvious tool).
- The pivot **removes the worst HN trap** ("advent calendar where?") — provided the posts frame the roadmap as vision with no dates.
- The differentiator sentence, ready-made: **"a toy machine where every bus read, write, and fault is a row in SQLite — faults are data, not exceptions."**

## D. What the Pivot Honestly Costs

- The teaching vision is **deferred, not dead** — keep `curriculum.md` as design, cut the dates.
- **Two repos = two maintenance surfaces:** ammaboard must pin tuikit by tag (a `replace` directive only for local dev), or the showcase breaks whenever the framework moves.
- The brand claim **"docs and code agree"** now has to actually hold on day one of being public.

---

# Section 3 — Further Steps (Post-Pivot Alpha Release Plan)

**Alpha definition:** a stranger can clone or download, understand the architecture from the README and docs, crash the machine in ten seconds and get the Blue Screen, see the TUI in GIFs (and run it), clone the extracted framework repo and evaluate it standalone in five minutes — and find both repos green in CI, with release binaries attached.

## Step 1 — Extract tuikit into its own repo *(1 weekend)*

Create `github.com/hmshwt/tuikit`. Move `internal/tuikit` + `testrender` verbatim (the boundary is verified clean). The real work is authorship, not code:

- **README:** elevator pitch, one diagram, <10-line quickstart, feature list (audit / receipts / golden-render tests / keymap scopes up front), a "why not raw bubbletea" section (port D41's rationale), limitations, license, provenance note ("extracted from ammaboard, where it runs a real app").
- **A small example app** in `examples/` so it is evaluable standalone.
- **API pass:** doc comments on every exported symbol, `CHANGELOG.md`, tag `v0.1.0`.
- **History:** either preserve it or start fresh with a well-messaged initial commit + provenance note — recommended: start fresh (the old commits reference `internal/` paths and another project's decision log).
- **ammaboard side:** swap the import, add the `require` (+ local `replace`), delete `docs/tuikit.md` from ammaboard, keep both test suites green.

## Step 2 — Hygiene pass: make ammaboard stranger-ready *(1 weekend)*

- Delete `internal/ui`; `go:embed` the counter demo; unify fault casing; drop the `machine.Event`↔`ticklog.Event` conversion; move `review.md` into the archive with a header.
- **README rewrite, architecture-first:** accurate encoding, fixed links, an ASCII/mermaid diagram of the broker + phases, "crash it in 10 seconds" (`bsod.amb` first, not `counter.amb`), and — since this is the alpha's stated goal — a proper **"Extending AmmaBoard"** section: *add a device* (echo as the worked example, one type + one `Attach`), *add an instruction* (Spec row + CPU file + parity case; the execute loop untouched). The recipes already exist in `implementation.md`; promote them to the README with runnable code.
- **Docs restructure:** date-free `roadmap.md` ("chapters to come"), curriculum kept as vision, fix the `docs/README.md` table.

## Step 3 — CI + release plumbing, both repos *(half a day)*

- GitHub Actions per repo: `gofmt`/`go vet` + `go test ./... -race` on linux/mac/windows (Go 1.25 — the `go.mod` directive requires it; the `-race` run is a nice badge for a determinism-branded project, and `emit.go`'s mutex deserves it).
- Release workflow on tag: `CGO_ENABLED=0` cross-builds (modernc sqlite already guarantees this works) — linux/mac amd64+arm64, windows amd64 — attached to a GitHub Release with checksums. This *is* the "single static binary" proof cited in the HN thread.
- Badges in both READMEs. Pin tuikit by tag in ammaboard's `go.mod`.

## Step 4 — Demo assets: make the invisible visible *(half a day + iteration)*

- **Three vhs tapes:** (1) the TUI three-pane run with step/speed keys, (2) the `bsod.amb` crash → Blue Screen, (3) `timer.amb` showing interrupt events landing in the log. Embed in the README — a TUI project's storefront is a GIF.
- Polish the examples' comments (they are the first programs strangers read), and give run artifacts a `runs/` directory instead of the CWD.

## Step 5 — The launch sequence: blog → Show HN → LinkedIn *(2–3 evenings for the blog; hours for the posts)*

- **Blog post (the load-bearing artifact):** the architecture tour with the named decisions — broker-not-god-object, phase-latched posted writes (read-old/write-new), faults-as-data with the stop-reason taxonomy (never conflate budget with halt), one event substrate serving observability *and* golden tests *and* the future course, the parity suite, determinism-by-construction (no goroutines, no map-order in state paths), and the honest tuikit story cross-linking both repos. 1,500–2,500 words, diagrams and code excerpts, at a stable URL.
- **Show HN:** Tue–Thu morning, title carrying the differentiator ("Show HN: AmmaBoard – a toy machine where every tick is a row in SQLite"). The first comment should be the author's: architecture rundown, "chapters to come" framing, and what it explicitly *isn't* yet. Pre-write answers to the predictable objections: *why not NandToTetris / QEMU / a real RISC-V*, *why SQLite*, *isn't it a toy*. Point to the pre-built binary and the GIFs.
- **LinkedIn:** the professional narrative — built a deterministic simulator, event-sourced design, extracted a reusable TUI framework — linking the blog + both repos. Post it after the HN thread has heat and link the discussion as social proof.
- **Optional cross-posts:** r/golang show-your-project, lobste.rs.

**Order rationale:** repo ready → CI green → assets exist → blog live → HN (the blog is the substance) → LinkedIn (after the HN spike, linking the discussion as proof).

**Total: roughly 2–3 weeks part-time — shippable as a "holiday showcase" in December, a far better position than the December beta that already got away.**

## What NOT to Do Under the Pivot

- Don't start the chapter engine "because the roadmap mentions it" — the roadmap is now prose, not scope.
- Don't add tuikit features during extraction. Extraction only; new kit features are post-alpha.
- Don't keep `internal/ui` "just in case."
- Don't put dates in the new roadmap — the repo's public history already contains one set of missed ones.

## Closing Assessment

The pivot is sound: it converts the strongest assets — a trustworthy machine, a disciplined decision culture, a genuinely self-contained framework — into portfolio value at the cost of polish and writing, not new engineering. The only real risk left is **presentation debt**, and that is the cheapest debt the project has carried since it started.
