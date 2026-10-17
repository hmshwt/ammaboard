# Machine architecture

## Execution model
Single-threaded, event-sourced, deterministic by construction. No goroutines, no channels
in the machine: components are plain objects ticked by the motherboard.

- **Major tick** = one machine cycle. **Minor tick** = a named phase: `FETCH`, `DECODE`,
  `EXECUTE`. The motherboard owns the schedule and tick order.
- Registers change immediately at EXECUTE; bus writes are posted and commit at the phase
  latch. The tick log is the history — no double-buffered component state.
- Shipped disclaimer: this is delta-cycle simulation (SystemC-like) — close to real
  clocking, not identical. The difference is a feature: we can pause inside a phase and
  deliberately mis-order phases to show races in slow motion.
- Multicore (day 17): N CPUs ticked round-robin in the same phases — simulated
  concurrency, still deterministic.

## Motherboard = broker
- Components never reference each other. All interaction is bus reads/writes, routed and
  permission-checked by the board against each component's address windows.
- `ReadBus(addr)` returns `(value, status)`, status ∈ `OK / NOT_CONNECTED / NOT_PERMITTED`.
  No Go errors mid-tick: failures are events and faults.
- `WriteBus` posts a pending write tagged with a `WriteSite{PC, Word}` naming the posting
  instruction. Writes become visible at the phase latch — everyone within a phase sees a
  stable bus.
- A write no window owns (or its owner refuses) faults **at the latch**, symmetrically
  with loads: kind `bus_NOT_CONNECTED` / `bus_NOT_PERMITTED`, with the posting
  instruction named. Stores never silently drop.
- Bus contention is deferred by the maxim *introduce friction only when friction is the
  lesson*: exclusive write slots per phase make contention impossible before day 19; a
  conflict is a loud halting `BusConflict` fault. Arbitration is day 19–20 or season 2;
  the `WriteBus` API is shaped so an arbiter can slot in without rework.

## Stop model
Two vocabularies, never mixed:

- `Fault{Kind, Addr, PC, Phase, Word}` — fault taxonomy only: `unknown_opcode`,
  `alignment`, `stray_rfi`, `bus_NOT_CONNECTED`, `bus_NOT_PERMITTED`, … First fault wins.
- `StopReason` — how the run ended:
  - `halt` — the program ran `halt`; no fault record (`Env.CleanHalt()`).
  - `fault` — a component faulted; see `board.Fault()`.
  - `budget_exhausted` — the cycle cap was hit mid-flight. A **pause**, not a stop:
    nothing halted or faulted, and another `Run` resumes the machine.
  - none — nothing has run.
- The Blue Screen renders only for `fault` stops. Snapshots are `FROZEN` on halt/fault,
  `PAUSED` on a budget cut.

## Interrupts
- Lines are level-triggered: devices implement `InterruptLine.InterruptRaised()`, and the
  board re-samples the device at the delivery edge. A line acked before delivery is
  dropped (`irq_cleared` event) — stale state can never deliver.
- Delivery: at the latch edge of the previous cycle, before the next FETCH. At most one
  per cycle. Refused lines (IE clear) stay raised and retry.
- CPU context parks in shadow registers (`intPC`, `intStatus`); IE clears on entry — no
  nesting. `return_from_interrupt` restores both; a stray rfi faults `stray_rfi`.
- Trap vector is fixed at 0x0008. The "interrupt controller" is broker state (an ordered
  pending set) until a second source exists.

## Determinism (CI-enforced)
1. No wall-clock time, no goroutines, no map-iteration order in state or event emission.
2. RNG only ever seeded and sim-owned (none today).
3. Same binary + same input ⇒ byte-identical event tables — golden-trace tests.
4. UI speed control = run N cycles per frame; machine semantics never change.

## Memory sizing
RAM is the smallest power of two that shows the chapter's mechanic: default 16KB, hard
ceiling 32KB. If a payload doesn't fit, shrink the payload.

- Pages: 1KB → 16 frames (not real 4KB; undemoable in 16KB). 32-bit virtual addresses,
  two-level walk, tables small enough to hexdump.
- Cache: 64B lines (real), 16–32 lines. Real CPU:RAM ratio ~1:1000 is deliberately
  exaggerated so the learner can watch it work.
- Every chapter ships a "vs reality" note naming every simplification — users must always
  know what was scaled.
- Full tick-log retention and per-tick snapshots are affordable at these sizes.
