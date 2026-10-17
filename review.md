I've read the docs, the full implementation, and verified several suspicions with scratch tests. Here's my critique, in the three layers you asked about.

---

## 1. The idea

**What's genuinely good**

- **The gap is real and well-chosen.** NandToTetris stops at a toy; CS:APP/Drepper are passive. "History as necessity" (pain → invention → mechanic) is the right spine, and the pain catalogue maps every invention to a feature a working dev actually meets (false sharing, TLB flush cost, Spectre → KPTI).
- **Inverted exercises are the strongest original mechanic.** Shipping the workload and making the *learner* build the machine/kernel is consistent with the day-23 capstone, needs no sandbox, and grades as diffable state instead of vibes. Very few teaching tools do this.
- **Determinism + the event log is a rare dual-purpose design.** Same substrate is the product's observability feature, the test strategy (golden traces), and the lesson ("faults are data"). The Blue Screen homage with "the wall of hex you can actually read because you built every byte" is the best one-line articulation of the whole course.
- **Constraints-with-names culture.** "Friction only when friction is the lesson", mandatory "what we simplified / vs reality" sections, recorded deviations. This is what keeps a fidelity-vs-teaching project honest.

**What I'd push on**

- **The storefront is a different product than the engine, and the plan doesn't say so.** Doors 1–2 (plugboard patch, deck reorder) are bespoke interactive simulations that share almost nothing with the tick machine. `detailed-plan.md` frames them as "zero ISA = pure engine/UI work", but they're neither engine nor machine — they're two one-off games. Risk #2 ("day-1 experience is the storefront") is named, but the mitigation (do doors 1–2 first) puts your *least reusable* work on the critical path. I'd prototype door 1 **and** door 4 in parallel — they're the two ends of the UX spectrum, and door 4 exercises the machine engine you'll reuse 20 times.
- **Two seams the headline features silently depend on are unspecified:**
  - Day 5 "write the assembler" in a toy ISA that has no input device until day 19–20. Someone (the UI?) has to feed source text into RAM and collect output — that's a designed interface, and it exists nowhere in the docs.
  - Day 11 "your own MMU" says the page-walk is learner-authored in toy ISA and "consulted as a privileged routine" — but who traps, in what registers the walker is entered, how it returns, and what the CPU expects on TLB miss is unwritten. That's the beta's hook; a one-page spec now is cheaper than discovering it during M4.
- **"Any dev with ≥2 years"** still assumes fluency in hex and two's complement. Fine, but that assumption is load-bearing for the primers and should be a stated, tested calibration (e.g., re-check it with a non-author playthrough at M2).
- The **end condition** ("stops when the community can play DOOM") is memorable and useful as a north star; it's also unbounded. The plan already caps the release scope, so just don't let the north star creep into 2026 scope.

---

## 2. Code structure

**What's good**

- Clean, small package seams: `isa` (encode/decode only), `asm` (two-pass), `machine` (broker + components), `ticklog` (storage/export). No cycles, no goroutines, pure-Go SQLite, single binary.
- The **broker + window routing + optional interfaces** (`Reader`/`Writer`/`ByteReader`/`ByteWriter`) is the right amount of machinery. `Env` keeps components from referencing each other; posted writes / phase latch are implemented for real and tested (`TestWritesVisibleOnlyAfterLatch`).
- Faults/BusStatus as data, Blue Screen as a frozen snapshot — the docs' claims are actually true in code, which is rarer than it should be.
- Tests are uniform and meaningful: end-to-end through the real wiring, determinism at machine *and* export level, timer round-trip, fault screens. `go vet` and `gofmt` are clean.

**Verified defects** (I reproduced all of these)

1. **Immediate range check is wrong** — `internal/isa/isa.go:100`. `Encode` accepts `imm` up to `2^18-1`, but `Decode` sign-extends bit 17. So `add_number r1, r0, 131072` encodes fine and silently decodes as `-131072`. The check should be `imm > (1<<17)-1`. The test (`isa_test.go:36`) checks `2^17-1` and `min-1` but never values just above `2^17-1`, so it misses this. Cheap fix, but it's exactly the kind of thing whose windows close when golden fixtures freeze.
2. **Stores to unowned addresses vanish silently** — `motherboard.go` `commit()`. A `store_word` to `0x5000` logs `bus_write_post`, routes nowhere, commits nowhere, emits nothing, and the machine reports a clean halt. A `load_word` from `0x5000` faults `bus_NOT_CONNECTED`. This breaks the "refuses nonsense, symmetrically" claim in `decisions.md` D4/README, and it's precisely the learner mistake the machine exists to catch loudly. Same silent drop for writes into a window whose owner isn't a `Writer`. Suggest a `BusConflict`-style fault at the latch (or, minimally, a loud dropped-write event).
3. **Level-triggered interrupts are actually queue-triggered** — `deliverInterrupts` (`motherboard.go:232`) removes the line only on delivery, while the device can clear it independently. Repro: timer raises at cycle 8, program acks at +4 while IE is clear, `allow_interrupts` at 28 → the stale line is delivered at 29, even though the timer's next real raise is at 32. Controller state is duplicated (`board.irqs []string` vs `timer.pending bool`). Day 8 leans on this exact semantics, so it's worth fixing now: have delivery re-sample the device's line state, and add a test for ack-before-delivery.
4. **Budget exhaustion is reported as "clean halt"** — `main.go` prints `stopped: clean halt` for `-max 3` on `counter.amm` because `Motherboard.HaltedBy()` returns `{Kind:"halt"}` whenever there is no `Fault`, conflating "no fault" with "halted". Once the TUI shows a green acceptance row for a truncated run, this gets expensive. Distinguish `ran_to_budget` from `halt`.
5. Smaller things:
   - Halt/fault state lives in two places (`CPU.haltedBy` and `Motherboard.Fault`, with `HaltedBy` on both). Two sources of truth; it will bite when an arbiter or device can also halt the machine.
   - `machine.Event` and `ticklog.Event` are duplicate structs plus a conversion at the boundary. One shared type would delete a layer.
   - The broker formats CPU status detail (`fromStatus&^StatusIE`) — the board knows CPU register semantics; fine today, drift-prone as `PRIV` wakes up.
   - `ticklog.Export` regex-parses `detail` strings (`reALU`, `reLoad`, …) to build cycle comments. This makes **event detail strings a de facto API** — the freeze-point discipline (D13–D17) is doing the real work here; say so explicitly in the docs and keep the golden tests per chapter. (Typed details are the eventual fix; not now.)
   - `readEvents` in `main.go` re-opens SQLite just to render the last 8 events for the Blue Screen; the buffer would do.
   - Fault kinds mix case: `alignment` / `unknown_opcode` / `stray_rfi` vs `bus_NOT_CONNECTED`. Pick one convention before fixtures.
   - `describeHalt` (board) and `haltSummary` (main) render the same fact twice.

**Structural divergences from agreed design (unrecorded)**

`motherboard.md` §8 AGREED on *package per component* and *one file per instruction with a registry, execute loop untouched*. Reality: one flat `machine` package and a large `switch` in `cpu.go`. That's a defensible PoC simplification — but `poc.md` §3 lists the deviations it took (no double-buffering; three phases) and omits this one. At 10 instructions it's fine; at 21 (stage 3) the switch is where all the risk sits, and "adding an instruction = one file + register + test" won't be true. Either record the deviation now or restructure before M4.

---

## 3. Planning

**What's good**

- M1–M7 with a dependency graph, acceptance defined as diffable artifacts, slack computed, and an honest slip policy ("labeled locked doors, never broken doors").
- The **freeze-point discipline** (fix event vocabulary *before* golden fixtures) and `decisions.md`'s D-numbers with reversal costs are unusually strong practice. The `poc-disccussion.md` cross-check of an external review — verifying each claim against code instead of absorbing it — is exactly the right instinct.
- Risks named are the *actual* top risks, including discoverability ("existential, not quality"), and monetization is ranked honestly with realistic numbers.

**What I'd change**

- **The schedule doesn't match the stated risk.** Risk #1 says content is ~70% of the work, but 3 of 7 weeks go to machine milestones (M3–M5) and M2 — chapter engine + TUI shell — hasn't started with 45 days left. The keystone interface is the **chapter data format** (it's engine↔content). Spec that before writing either side; it's the cheapest way to de-risk M2/M6 simultaneously.
- **`internal/ui` is the biggest unknown and ui.md is still draft** (keymap undecided, shell command set, acceptance-checker rendering). The progressive-disclosure design ("UI grows as the machine does") is right — but M2 should produce one chapter end-to-end, not generic pane machinery. Also: `ui.md` should state the runaway-learner-program behavior (budget vs break vs pause) — it's the same gap as code issue #4.
- **Trace schema is a product surface** ("queries are data", user SQL expected). `Runs` has `machine_version` but no `schema_version`. Adding one now costs nothing and is impossible-ish after strangers have `.db` files. Same for the progress file format in M7.
- **Doc promises without milestones:** "published queries via sqlc" (`motherboard.md` §7) appears in no milestone; either schedule it or drop it from the architecture doc. In a project whose brand is "the docs and code agree", orphan promises are worse than missing features.
- **Golden-trace fixtures**: the plan freezes vocabulary in M2/M3 but the fixture *shape* (exact diff surface, per-chapter files, how a deliberate re-render like D24/D25 doesn't invalidate them) is undesigned. The D24/D25/D26 changes in this very worktree show how fluid the render layer still is — freeze the *events table* contract, not `trace.txt`, and say that in the plan.
- Process nit: the worktree currently mixes a coherent unit (Blue Screen + README + decisions D24–D27) with WIP edits, and `trace.last.txt` isn't in `.gitignore`. Commit the Blue Screen as its own change; add the stray artifact.

---

## Priority list

1. Fix the four verified defects before any vocabulary/fixture freeze: **immediate range** (`isa.go:100`), **silent dropped stores** (`motherboard.go` commit), **stale IRQ delivery** (`motherboard.go` + `timer.go`), **budget-vs-halt** (`main.go`/`HaltedBy`). Each is small; each is exactly what the "refuses nonsense" / determinism brand promises.
2. Add a round-trip property test for every immediate (and use it as the template for future stage-3 encodings).
3. Record the package-layout/registry deviation in `poc.md` §3 — or commit to the registry before stage 3.
4. Write one-pagers for the three unspecified seams: chapter file format, day-5 assembler input path, day-11 walker calling convention. These gate M2 and M4; prose is cheaper now than code later.
5. Add `schema_version` to `runs` and decide the "budget exhausted" vocabulary while the event vocabulary is still soft.

Bottom line: the idea is strong and the engineering doc culture is better than most funded projects; the machine itself is close to trustworthy, with four small-but-embarrassing exceptions that are cheap to close now and expensive after fixtures freeze. The real risks aren't in `internal/` at all — they're the TUI, the chapter format, and the two bespoke door-1/2 simulations, none of which have a spec yet.
