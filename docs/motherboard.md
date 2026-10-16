# Motherboard — hardware design & implementation

Living document for machine architecture and its Go implementation.
Companion docs: PLANNING.md (curriculum/decisions), poc.md (PoC scope).

---

## 1. Execution model — AGREED (sessions 7–8, discussed)

**Elm-like structure (not a copy), motherboard as broker, explicit phase-based clock.**
No goroutine-per-component. Channels unnecessary: components are plain objects ticked by the
motherboard; determinism by construction. (Earlier single-loop sketch superseded.)

### Component interface (simple, deliberately small)
Each component implements `Tick(msg TickMsg)` — the TEA-like update — plus exposes its state
for rendering. Hardware components never issue side-effect Cmds; only the UI layer does.

### The clock: major/minor ticks
- **Major tick** = one machine cycle. Displayed as nanoseconds (integer multiple of 100ns base).
- **Minor tick** = a *named phase* within the cycle (e.g. `ADDR`, `DATA`; CPU: `FETCH`,
  `DECODE`, `EXECUTE`). Internals are phase-based; nanoseconds are a derived display mapping
  from a timing table (which later lets P-cores vs E-cores differ on day 22 without logic changes).
- Each component keeps **prev + current state**, tagged with `(major, minor)` — this makes
  read-old vs read-new *explicit* (some components legitimately react to pre-tick state).

**Clock-semantics disclaimer (TODO for us both: brush up before pipeline chapters):** what we
implement is closest to discrete-event/delta-cycle simulation (SystemC-like: named phases, stable
reads within a phase, state updates at edges) — almost the same as real clocking, not the same.
We ship a one-paragraph note in the app saying exactly that. The payoff of "not same": explicit
control over the clock lets us pause *inside* a phase and deliberately mis-order phases to show
learners races in slow motion.

### Motherboard as broker (root component)
- Defines and owns the phase schedule + tick order; ticks all components each minor.
- Keeps the state map of all components and the interconnection graph (which component is wired
  to which bus, with what permissions).
- Components never hold references to each other. A read is `ReadBus(busId)`; motherboard checks
  connectivity/permissions and returns a **uniform struct**: `(value, BusStatus)` with
  `OK / NotConnected / NotPermitted / Idle`. Failures are also logged as events (teachable:
  "DMA read bus 0x8000 — not connected" on day 19). No Go-error branching mid-Update.
- **Write timing:** `WriteBus` posts a value *pending* on the bus; it becomes visible to readers
  only at the phase latch (between phases). Everyone within a phase sees a stable bus — two
  readers in the same phase can never see different values.
- Double-buffering is limited to bus latches / small output registers, NOT full internal state
  (RAM contents are not snapshot per tick — observers sampled the bus, like real hardware).
- Every `(major, minor, component, msg)` interaction is logged → per-step observability is
  structural, and time-travel debugging = replaying the tick log.

### Multicore (day 17) and speed control
- N CPU components ticked round-robin by the motherboard within the same phases.
- UI speed = run N majors per animation frame; never changes machine semantics. Sim remains
  headless-runnable for tests and the acceptance checker.


## 2. Component model (Go)

**Name & version (session 10, revised; corrected 2026-10-16): the machine is
`AmmaBoard-{generation}`.** ("Amma" = mother; plain "Board" spelling is canonical.)
History: first draft "MaadarBoard" rejected (one syllable from a Hindi expletive); the pun
spelling "AmmaBored" was considered and dropped — "AmmaBoard" is the chosen name.
Generation = ISA stage, not app release: AmmaBoard-1 = boot ten (days 4–5), -2 = +bytes/
shifts/interrupts (day 7), -3 = +TLB/protection (day 10), onward per growth stages.
`machine_version: "AmmaBoard-N"` stamped in every trace's `runs` row. Doors announce their
generation. Experimental "version mode" (sidequest) = an unnumbered next generation.
Brand thread: day-12 campaign becomes "Look Amma, I made my own Blue Screen of Death".
CLI binary may use a short alias (`amma`); canonical romanization AmmaBoard.
License: **Apache-2.0** for the whole project (repo + machine + chapters code).
Future: if it takes off, FPGA kits running AmmaBoard are a plausible community spin-off
(hardware=soft, tiny ISA + 32KB RAM is well within soft-core/CPLD-class territory).

```go
type Component interface {
    Tick(msg TickMsg) // one phase; ordering owned by the Motherboard
}
```
Draft packages:
- `internal/machine` — motherboard (root: schedule, state map, interconnections, ReadBus/
  WriteBus broker), cpu, ram, bus, cache, walker, tlb, timer, intc, baselimit, ticklog
- `internal/isa` — opcodes, encodings (R/I/B), disassembler
- `internal/asm` — assembler + pseudo-op expansion (used by day-5 exercise, day-2+ payload tools)
- `internal/payload` — hex-binary format (header: origin, length, entry), loader
- `internal/chapter` — chapter data: payloads, machine config, observation lists, expected traces
- `internal/ui` — bubbletea models (the ONLY layer issuing Cmds); `internal/progress` — save state

State ownership: components never reference each other — all interaction via `ReadBus`/
`WriteBus`, brokered and permission-checked by the motherboard. No shared maps, no locks
(single-threaded). Determinism invariant enforced by a test that runs every chapter twice and
compares full tick logs.

## 3. Hardware map to curriculum (what exists when)
| Chapter | Hardware added |
|---|---|
| days 1–3 | none (narrative/mechanical sims) |
| 4–5 | CPU (boot-ten), RAM, Bus, EventLog |
| 7–8 | stack convention is software; Timer + IntC; status reg (IE) |
| 10 | BaseLimit unit (hardware-only change) |
| 11 | Walker + TLB + privilege bit + trap vector; `flush_tlb` |
| 13–14 | Cache (config-driven: direct-mapped → set-assoc, policy pluggable) |
| 16 | pipeline counters (visualization: numbers/grids, not animation) |
| 17 | second + third CPU structs, MESI state machines, `swap_atomic add_atomic` |
| 19–20 | Device (MMIO range), DMA engine |
| 22 | P-core/E-core structs (same ISA, different timing table) |

Fault plumbing lives in the CPU execute-loop signature from the first commit (dormant → day 10),
per poc.md.

## 4. Determinism rules (invariants, CI-enforced)
1. No wall-clock time, no goroutines, no maps iteration in sim state or event emission ordering.
2. RNG only via seeded PRNG owned by the sim.
3. Every chapter run is reproducible: same binary + same inputs ⇒ identical event log bytes.
4. UI speed control = run N cycles per tick; NEVER changes machine semantics.

## 5. Memory sizing — AGREED (session 8)

**Principle: RAM is the smallest power-of-two that shows the chapter's mechanic completely.**
Default **16KB**, hard ceiling **32KB**. Never large "just in case."
- If a payload doesn't fit → shrink the payload (soft target: concepts explained in ≤1024 bytes).
- Every chapter ships a **"vs reality" note**: real numbers (x86: 4KB pages, 64B lines, ~1:1000
  cache ratio) vs our scaled-down ones — users/devs must always know what was scaled.
- Small memory = full-machine hexdumps on one screen + zero memory management in the simulator
  itself (a whole class of simulator bugs eliminated).

Consequences:
- **Pages: 1KB → 16 frames** (not real 4KB — undemoable in 16KB; noted in "vs reality").
  32-bit virtual addresses kept; two-level walk, each table small enough to hexdump.
- **Cache: 64B lines (real), 16–32 lines.** Real ratio is ~1:1000; we exaggerate deliberately
  so the learner can watch it work — said openly in the note.
- Performance is explicitly a non-goal; determinism and correctness are the only invariants.
  Full tick-log retention, per-tick snapshots, scrub-back: all affordable at these sizes.

## 6. Bus contention — AGREED (session 8): deferred by the frictionless-surface maxim

**Project maxim: introduce friction only when friction is the lesson** (like frictionless
surfaces in physics class). Simplifications are deferrals with a name, never silent lies.

- Days 4–18: exclusive write slots per phase → contention impossible by construction. If a
  chapter config or learner kernel ever produces one: loud halting `BusConflict` fault,
  framed as the hardware fault it is ("two drivers on one wire").
- Arbitration (grants, wait states, bus stealing) = future chapter material: bundle with
  day 19–20 (DMA) if 2026 bandwidth allows, else a season-2 "when devices and CPUs fight"
  chapter.
- PoC obligation: shape the `WriteBus` API so an arbiter can slot in later without rework
  (writes as requests, grants as events); build NO arbiter logic now.

### Every chapter ships a mandatory section: "What have we simplified for focus"
- Names each simplification + why + where the full story gets told (later chapter / rest-day
  reading / season 2).
- Doubles as authoring QA: if the author can't write this section, the chapter's teaching
  goal isn't decided yet.


## 7. Storage engine — LOCKED (session 8)

**SQLite, pure-Go driver (`modernc.org/sqlite`), one .db file per run, owned by the motherboard.**
- No cgo — keeps `CGO_ENABLED=0`, single static binary, cross-compiled releases (founding
  requirement). Performance is a non-goal; slower pure-Go driver costs nothing that matters.
- Event writes batched in transactions per major tick; flush semantics never affect machine
  behavior (logger sits outside machine semantics).
- Schema sketch:
  - `runs(id, chapter, started_at, machine_version)`
  - `events(run_id, tick_major, tick_minor, phase, component, kind, payload_json)`
  - `snapshots(run_id, tick_major, tick_minor, kind, state_blob)` — chapter start/end + faults
    (state at any tick = replay events; per-step state dumps rejected as redundant)
  - `queries(id, name, sql)` — published queries are DATA; upgradable with app updates
- **Published queries via sqlc**: schema + typed, tested queries (hit rates, faults, wait
  states) as the supported interface; power users write raw SQL. User queries = feedback
  channel on what the machine teaches.
- Golden-trace tests = run chapter twice, diff ordered `SELECT * FROM events`.
- Text door: `trace export` dumps a run as plain text (grep/diff/eyeball). No custom binary
  trace format ever gets written.
- Known cost: trace files are version-sensitive → `machine_version` stamped in `runs`; old
  files migrate or open read-only. Decide at beta.


## 8. Implementation layout — AGREED (session 8)

- **Package per component** (`internal/machine/cpu`, `.../ram`, `.../cache`, …): the package
  boundary enforces "no direct references, bus only." Entangled sub-parts (TLB + walker) stay
  in ONE package — package per component, not per concept.
- **One file per instruction inside `package cpu`** (`cpu.go`, `add.go`, `load_word.go`,
  `jump_if_not_equal.go`, …):
  idiomatic Go, zero ceremony, one obvious place per instruction. Families that share shape may
  merge into one file (`logic.go`) later without breaking anything. Instructions registered in
  a registry keyed by mnemonic; adding an instruction = add a file + register it + its tests,
  execute loop untouched.
- **One table-driven test harness per component**; every instruction gets the same style of
  golden register/memory cases. Uniform test shape = uniform contribution shape.
- **Wiring in `main`** (`main/wiring.go`): construct components, hand to motherboard. Later
  (v2): generate wiring from chapter config.
- Interfaces/packages named as if third parties will read them — cheap now, and the
  component+registry+SQLite-trace contracts are what would enable a community ecosystem
  later. **"LearnISC"-style platform/branding = v3–v4 idea**, deliberately not a design
  constraint now (scope gravity).

## 9. Open questions
1. ~~Tick-log memory~~ — resolved by §5: full retention, trivially affordable.
2. ~~Snapshot format~~ — resolved session 10: **Blue Screen freeze snapshot** (see ui.md §9).
3. ~~Bus contention semantics~~ — resolved by §6: loud halting `BusConflict` fault pre-day-19;
   arbitration deferred, `WriteBus` shaped for it.
4. Phase sets: CPU phases (FETCH/DECODE/EXECUTE) composed of bus phases (ADDR/DATA) — **accepted
   in principle (99%), final confirm pending user's read-up**. Blocks only the Tick loop's
   concrete shape, not the architecture.
5. TODO (both of us): brush up on real clocking semantics (edge-triggered flip-flops, setup/
   hold, delta cycles) before building day 16–17 content, so the disclaimer in §1 is accurate.
