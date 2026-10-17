# Decisions — assumed defaults, recorded for review

Written 2026-10-16, during the stage-2 implementation pass. Per instruction, every
decision below was made **unilaterally** ("no questions, just assume") and is written
down here for owner review. All are cheap to reverse now and expensive after
golden-trace fixtures freeze the event vocabulary (detailed-plan.md M2/M3) — flag
objections before then.

Companions: PLANNING.md (decisions of record), poc.md (ISA stages),
motherboard.md (machine architecture), poc-disccussion.md (trace-vocabulary critique),
detailed-plan.md (milestones this pass draws from).

## Scope of the pass

**D0 — "next stage" = ISA stage 2 + day-8 interrupt hardware + trace-vocabulary
fixes.** Stage 2 per poc.md §1 (`load_byte store_byte shift_left shift_right
shift_right_arithmetic bit_and bit_or allow_interrupts forbid_interrupts
return_from_interrupt`), the timer + interrupt delivery of day 8 (poc.md §2), and the
five trace fixes parked in poc-disccussion.md §6 — pulled forward from M2 because they
change event kinds and must precede any golden fixture.

## ISA and CPU

- **D1 — Opcode numbers 11–20, poc.md §1 order.** `load_byte(11) store_byte(12)
  shift_left(13) shift_right(14) shift_right_arithmetic(15) bit_and(16) bit_or(17)
  allow_interrupts(18) forbid_interrupts(19) return_from_interrupt(20)`.
- **D2 — Shifts and bitwise ops are R-type `rd, rs, rt`; shift amount = `rt & 31`.**
  No immediate-shift encoding until the sugar layer needs one (three encodings only:
  R, I, B — PLANNING.md ISA).
- **D3 — `load_byte` zero-extends; `store_byte` stores the low byte; byte ops have no
  alignment constraint** (that is their point); word ops keep the 4-byte alignment
  fault. Stage 2 has no sign-extending byte load — `shift_right_arithmetic` covers sign
  work, per poc.md's stage-2 rationale.
- **D4 — Byte access rides the same bus via optional interfaces.** Components may
  implement `ByteReader`/`BusReadByte` + `ByteWriter`/`BusWriteByte`; the broker routes
  byte ops inside existing windows. A window owner without byte support answers
  `NOT_PERMITTED` (refuses nonsense, symmetrically). New events: `bus_read_byte`,
  `bus_write_byte_post`.
- **D5 — Status register: `IE` = bit 0, `PRIV` = bit 1 (dormant until stage 3).** No
  CSR file — a flag bit carries the lesson (PLANNING.md, decided).
- **D6 — System ops are zero-operand; rd/rs/rt/imm are reserved-must-be-zero,** enforced
  in both `Encode` and `Decode`. This makes the documented "machine faults on nonzero
  reserved bits" real for the first time (the teachable fork).
- **D7 — Interrupt context lives in CPU shadow registers, not memory.** A taken
  interrupt parks `pc` and status in `intPC`/`intStatus`, clears IE (no nesting — the
  handler runs forbidden), and aims pc at the trap vector. No stack exists yet (day 7
  introduces calling conventions); memory-based context switch is day-20 material.
  `return_from_interrupt` restores both; an rfi outside a handler faults `stray_rfi`.
- **D8 — Trap vector is fixed at 0x0008.** Vectored dispatch arrives with a second
  interrupt source, if ever.
- **D9 — Interrupt seam (adopted from poc-disccussion.md §3):** the board delivers at
  the latch edge of the previous cycle, before the next FETCH. Lines are level-triggered:
  refused (IE clear) lines stay raised and retry at the next edge; at most one delivery
  per cycle; delivery event kind `interrupt`, attributed to `board`, phase FETCH.
- **D10 — The "interrupt controller" is broker state** (an ordered pending set) until a
  second source exists; then it becomes a real component with priority encoding. One
  wire does not need an arbiter.

## Devices

- **D11 — Timer: MMIO at 0x4020, 8 bytes.** `+0` interval (write arms and restarts the
  counter, 0 disarms; read = current interval); `+4` pending (read = 1 while raised,
  write = acknowledge). The counter advances once per major tick, on FETCH. Word
  accesses only. Always attached in `wiring.go` — harmless while disarmed, and it makes
  the day-8 chapter a store away.
- **D12 — Machine generation is stamped `AmmaBoard-2`** (motherboard.md §2: generation =
  ISA stage). Single source: `machine.MachineVersion`, used by the tick-log `runs` row
  and the state-dump banner.

## Trace vocabulary (freeze point for golden fixtures)

- **D13 — Every ALU op emits a uniform `alu rX <= a OP b = c` EXECUTE line.** Option 1
  of poc-disccussion.md §4: registers do not latch in this machine (poc.md §3 deviation),
  so per-register latch lines would print a lie.
- **D14 — PC convention: `c.PC` is the next-fetch address, always.** The halt line shows
  both values (`clean stop at pc=0x0020 (next fetch 0x0024)`); `branch`/`call`/`goto`
  lines carry `at=` (this instruction) and `next=`; not-taken branches are now logged
  too; `StateDump` labels `pc=… (next fetch)` and adds the halted-instruction line.
- **D15 — Latch commits are one event per write, naming the target:**
  `commit ram[0x0040] <= 0x…`, `commit timer[0x4020] <= 0x…` (replaces "N write(s)
  committed"). The reviewer's suffix-on-the-post-line fix was rejected as impossible —
  writes route to their target at the latch, not at post time (poc-disccussion.md §4).
- **D16 — `bus_write_post` keeps its name.** "Posted write" is the term of art and
  motherboard.md §1 documents the semantics; the primer closes the vocabulary gap
  instead of a rename (poc-disccussion.md §5).
- **D17 — `-dump-full` CLI flag** switches RAM dumps (console print and both BOOT/FROZEN
  snapshots) to the every-word view via the optional `VerboseStateReporter` interface.

## Content and tooling

- **D18 — `examples/timer.amm`** demonstrates the day-8 pattern: handler publishes a
  count, acknowledges the line, disarms the timer at the target count; main spins on the
  published value. Data area at 128 — note `64(r0)` is decimal and would overlap code in
  programs larger than the counter's 9 words (this bit us once already).
- **D19 — Assembler accepts labels as `add_number` immediates** (absolute byte
  addresses) — this was already implemented (`immOrLabel`, non-branch); it is now the
  documented way to load addresses, since there is no `lui`.

## Detailed-plan §2 defaults, adopted to close M1

- **D20 — TLB granularity:** per-1KB page only, plus a "walk always" debug flag.
- **D21 — Payload header:** 12-byte little-endian `origin, length, entry` (to be specced
  in motherboard.md when the day-5 exercise lands).
- **D22 — Exercise editor scaffolding:** plain text input + step/run, no syntax frills.
- **D23 — Launch/repo name:** AmmaBoard (settled by owner 2026-10-16 — "ammaboard was
  the chosen one"; motherboard.md §2 corrected accordingly).
- **D24 — trace.txt rendering v2 (owner-requested, iterated twice):** each cycle opens
  with a comment line — its number and what it accomplishes
  (`; cycle 21 — subtract: r6 <= 2 - 2 = 0`; interrupt delivery outranks the
  instruction) — followed by a dashed rule and the cycle's indented events. The
  `major` / `end of major tick` bookkeeping rows are not rendered: the comment + rule
  are the clock boundary. Legend header on top; column header dropped (columns are
  named in the legend). Rendering only in `ticklog.Export` — the events table,
  golden-trace diffs, and the determinism test are untouched. Supersedes the
  poc-disccussion.md §5 recommendation against indentation (owner's call).
- **D25 — comments expand shorthand for onboarding (owner-requested):** cycle comments
  use full terms — `r0..r7` → `register_0..register_7`, `pc` → `program counter`,
  `mem[` → `memory[`, `ie=1/0` → `interrupts enabled/disabled` — applied only in the
  comment/prose layer; event rows stay compact so the machine record stays terse and
  greppable. Legend states the convention.
- **D26 — was/now visibility (owner principle): "the player gets complete visibility,
  even when the machine misbehaves."** Every state change is recorded with its old
  value at the source — the machine logs `… (was N)` on alu/load effects and
  `commit … (was 0x…)` on latch commits (broker reads the old value before writing) —
  and the cycle comment phrases it humanly: `subtract: register_6 was 2, now is
  4294967295 (= -1 signed, 0xffffffff)`. Numbers show every interpretation when they
  diverge (unsigned decimal, signed, hex); loads keep "was 0, now is 0" even when
  nothing changed, because a no-op read is also a fact. Event kinds unchanged; detail
  strings gained the `(was …)` suffix — golden fixtures must be written from this
  shape onward.
- **D27 — Blue Screen on improper halt (owner request; ui.md §9 in the CLI):** any halt
  that is not a clean stop renders the full panel — plain-language explanation per fault
  kind, the disassembled offending word, complete registers/status, a memory window
  around the faulting address (marked to the word), the board's address windows, and the
  last 8 log events with the fault marked. The panel is frozen into the log as a
  `bluescreen` snapshot row; fetch faults now also log a `fault` event so the chain is
  complete. The dead `CPU.Fault` field was removed; fault words travel in `Fault.Word`
  for every fault kind. Clean halts never blue-screen.

## Defect fixes + structure pass (owner-requested, 2026-10-17)

- **D28 — immediates are strictly 18-bit signed.** `isa.Encode` now refuses
  `imm > 2^17-1` (previously 2^17..2^18-1 passed Encode but Decode sign-extended
  them into the negative range — `131072` silently executed as `-131072`). The
  assembler surfaces Encode errors as line-numbered source errors via a small
  `enc` helper; `MustEncode` is no longer used in `asm`, so bad source never
  panics the tool. ISA stage 3 (base+limit, chunks) inherits the same edge.
- **D29 — writes that land nowhere fault at the latch, like loads.** A posted
  write no window owns (or whose owner refuses it — byte into word-only MMIO,
  unaligned word) used to be silently dropped while the same address faulted on
  load. The broker now stops the machine in `commit()`: a `fault` event
  (`… dropped at latch`) plus `bus_NOT_CONNECTED` / `bus_NOT_PERMITTED` on the
  board. `Env.WriteBus`/`WriteBusByte` carry a `WriteSite{PC,Word}`, filled by
  the CPU, so the fault names the posting instruction and the Blue Screen can
  disassemble it; unattributable writers pass the zero site. Stores and loads
  now refuse nonsense symmetrically.
- **D30 — interrupt lines are level-triggered all the way down: the board
  re-samples the device at the delivery edge.** New `InterruptLine` contract
  (`InterruptRaised() bool`), implemented by the timer from its own `pending`;
  the raise-time queue no longer delivers stale events. A line acknowledged
  before delivery is discarded with a board `irq_cleared` event; a line refused
  while IE is clear stays up and retries (unchanged). Removes the duplicated
  controller state that could deliver after an ack.
- **D31 — `budget_exhausted` is its own stop reason.** `-max` is a harness cap,
  not a machine event: a run that reaches the cap mid-flight was reported as
  "clean halt" because "no fault" was conflated with "halted". `HaltedBy()` on
  board and CPU now returns kind `budget_exhausted` when nothing halted; the CLI
  prints "budget exhausted (program did not halt)", the state banner says "budget
  exhausted (still running)" (pre-run: "not started"), the frozen snapshot is
  `PAUSED` instead of `FROZEN`, and no Blue Screen is rendered (nothing faulted).
  Clean halts and faults are unaffected.
- **D32 — `.db` files carry `schema_version`.** `ticklog.SchemaVersion = 1`,
  stamped on every run row; legacy files (pre-versioning) are migrated in place
  — the column is added, old rows stay 0, never rewritten — and read-only
  exporters treat a missing column as version 0 without running DDL. Both
  `Export` and `EventsReadback` refuse files written by a *newer* schema instead
  of misreading them. This is the versioning promise of motherboard.md §7 made
  concrete; bump the constant whenever the schema changes.
- **D33 — package per component is now real.** `internal/machine` holds only the
  broker + contracts (machine.go) and the motherboard (motherboard.go); the
  components live in `internal/machine/cpu`, `.../ram`, `.../timer`, `.../echo`,
  and the Blue Screen in `.../bluescreen` — the package boundary enforces the
  bus-only rule from motherboard.md §8. Inside `cpu`, one file per instruction
  (`add.go`, `load_word.go`, `jump_if_not_equal.go`, …) registered in `init()`;
  the execute dispatch (execute.go) is untouched when instructions are added
  (duplicate registration panics at startup). Component tests live in their
  package; board-level integration tests are an external `machine_test` package
  so the root imports no components.
- **D34 — stop state is not a fault (supersedes the kind soup of D31).**
  `Fault` is a fault taxonomy again; how the machine *ended* is a separate,
  first-class value: `StopReason` ∈ {none (Run not called), `halt` (the program
  ran halt), `fault` (see `Motherboard.Fault()`), `budget_exhausted` (the run's
  cycle cap was reached mid-flight)}. `Env` gains `CleanHalt()` beside
  `Halt(Fault)`: a clean stop carries no record. `HaltedBy()` is gone in favour
  of `Fault() *Fault` (nil unless something faulted) and `StopReason()`. Budget
  exhaustion is explicitly a *pause*: the machine is neither halted nor faulted,
  so another `Run` resumes it. The Blue Screen renders only for `StopFault` (a
  screen anywhere else would be a lie); the frozen snapshot is `PAUSED` on a cap
  and `FROZEN` on halt/fault.
- **D35 — the default cycle budget is 10,000 majors (`-max 0` = uncapped).**
  100,000 was picked before anything was measured; a runaway at that cap wrote
  ~10⁶ event rows before the user saw anything. 10,000 is an order of magnitude
  above the longest chapter program (~31 ticks) and about a second of worst-case
  work including SQLite writes and text export, so a runaway loop fails fast and
  cheap. `-max 0` opts out for long honest runs; the CLI rejects negatives.
- **D36 — one ISA table keeps assembler and CPU in parity.** `isa.Specs` is the
  single source of truth for every instruction: opcode, mnemonic, operand
  `Form` (zero/RRR/RRI/load/store/branch/call/goto). The assembler no longer
  switches on mnemonics — it parses by form with one encoder per form, so a new
  instruction whose form exists needs *no* assembler change; `Opcode.Name()`
  and the reserved-fields rule read from the table too. The CPU keeps one file
  per instruction and exposes `Executable`/`Executables` for the conformance
  suite in `internal/isa/parity_test.go`: every spec must assemble to its own
  opcode and have an executor, and the CPU must not register an opcode that has
  no spec (the build fails on drift). Tightening: `halt` now follows the
  reserved-fields rule (operand fields must be zero, like the other
  zero-operand ops) — hand-rolled halt words with junk fields fault instead of
  quietly working.

