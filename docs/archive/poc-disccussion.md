# Critique of the external trace review

Written 2026-10-16. Subject: an outside review of `trace.txt` (the `examples/counter.amb`
run). The reviewer saw only the trace, not the code or planning docs — stated up front in
their "questions I can't see answered" section. This critique cross-checks every claim
against the implementation (`internal/machine/`) and the planning docs. Companion to
`poc.md`; feeds `detailed-plan.md` M1/M2.

**Verdict in one line:** the review's two "fix before stage 9" items are real, its
polish list is mostly fine, but roughly half its factual frame is wrong about this
machine, and most of its "decide before stage 9" questions are already decided in code
or in PLANNING.md/motherboard.md.

---

## 1. What the review got right

| Claim | Verdict | Evidence |
|---|---|---|
| Latch is already a write batcher; "1 write(s) committed" is forward-looking | **Correct** | `runPhase` iterates `m.bus.pending` (a slice) and applies every write; multiple posts per phase already work (`motherboard.go`, `pendingWrite`, phase-latch loop) |
| ALU ops emit no EXECUTE line — register change is invisible in timing | **Real gap** | `Tick` EXECUTE: `OpAdd/OpAddNumber/OpSubtract` call `writeReg` with no `Logf` (`cpu.go`); trace tick 2 has FETCH+DECODE only |
| PC display inconsistent at halt (`pc=0x0020` vs dump `pc=0x0024`) | **Real bug** | halt uses `c.PC-4`, dump prints `c.PC` (`cpu.go`); both "correct" per phase, but unlabeled |
| Memory-mapped I/O from the start, fixed-width ISA, visible phase labels | Correct | `Reader/Writer` window routing; `isa.Encode` fixed 32-bit; `Schedule` |
| Unmapped-address status field implies a non-OK path worth pinning | Correct instinct, but **already implemented** — see §3 |

The PC catch deserves emphasis: it's actually *worse* than the reviewer found. `c.PC`
is always the next-fetch address (a clean invariant), but messages subtract 4
selectively: `halt` says `pc=0x0020` (own address), while `branch`/`call` say
`pc 0x001c -> 0x000c` (next-address → target, own address 0x0018 never shown). Three
renderings of one register. The reviewer found two of the three.

## 2. Where the review is wrong about this machine

- **"Registers and memory only change state at the clock edge" — false.** `writeReg`
  stores into `c.Regs` immediately, mid-EXECUTE (`cpu.go`). Only bus writes are posted
  and committed at the latch (`runPhase`). This is not a nuance: the review's preferred
  fix (option 2, "emit `latch r2 <= 5` because that matches what the hardware is actually
  doing") would print a lie. Registers don't latch here. Option 2 as written is only
  valid if we *change* the machine to post register writes — which `poc.md` §3 explicitly
  deviated from ("no prev/current double-buffering; the tick log is the history").
- **"The board speaks bus protocol, the cpu speaks instruction protocol" — inverted.**
  The broker (board) *emits* `bus_read`/`bus_write_post` events but attributes them to
  whoever is ticking (`m.event` → `m.ticking`), so all bus lines appear under `cpu` in
  the trace; the board emits only `major` and `latch` (`motherboard.go`). The
  architectural conclusion (MMU slots in at the broker) survives; the description of the
  trace doesn't match it.
- **"You're word-addressed" — wrong.** RAM is a byte array; PC advances by 4 *bytes*;
  addresses are bytes (`addr=0x0040`, `ram: 0x0000..0x4000 (16384 bytes)`). Word-*granular*
  accesses on a byte-addressed machine — which is exactly why `load_byte`/`store_byte`
  are stage-2 additions (poc.md §1) and why misalignment faults already exist. The
  reviewer's "cleanest way to add byte addressing" is a description of what's already
  shipped, minus the byte-sized ops landing day 7.
- **"The `echo@0x4000` line shows…" — that line is not in this trace.** It appears only
  with `-echo`; `counter.amb` ran without it.
- **Stage numbering doesn't exist in this project.** The curriculum is 24 doors / 16
  chapters (days 1–24); "stage" already means *ISA growth stage* (six of them, PLANNING.md).
  The review's "before stage 9 / stages 9–16" timeline is unmoored: interrupts are day 8,
  MMU day 11, kernel capstone day 23 — several of its "decide before stage 9" items are
  due *earlier* than it thinks, and its "stage 9's kernel will need a stack" is off by
  ~14 days.
- **"The CPU still speaks physical addresses to its own internal bus; the MMU translates
  virtual-to-physical" — self-contradictory.** If the CPU spoke physical addresses there
  would be nothing to translate. (The real design note: poc.md's sketch gives the CPU a
  pluggable `Translate`; broker unchanged.)

## 3. Questions the review raises that are already answered

| Review question | Already decided | Where |
|---|---|---|
| Unmapped read behavior? | Broker returns `NotConnected` (logged), CPU halts with a `bus_<status>` fault — same path that will carry permission faults later | `ReadBus` fallthrough; `cpu.go` fetch: `halt(Fault{Kind: "bus_"+st…})`; motherboard.md §1 status set |
| Run off the end of memory? | Same mechanism: fetch at an unmapped PC → `NotConnected` → machine halts. No wraparound exists to remove | Same code path |
| Is the major tick the accounting unit? | Yes — major tick = one cycle; ns display is a derived mapping; acceptance budgets are cycle counts (~10× reference) | motherboard.md §1; session-5 challenge format |
| Minor ticks before you need them? | **Already shipped.** Minor ticks exist — they are the named phases (FETCH/DECODE/EXECUTE); the phase column *is* the minor tick. Multi-cycle accesses are wait-states/phase-schedule territory, planned, not a structural gap | motherboard.md §1; poc.md §3 deviations ("confirmed by use") |
| Stack pointer: fixed register or special-purpose? | Decided: no special SP. `call` links through fixed r7; the stack is RAM walked with ordinary registers (`add_number` exists for the day-7 sp walk) | `isa.go` OpCall; PLANNING.md ISA ("fixed link register r7; generality becomes sugar later") |
| Where does the privilege bit live? | Decided: one status register (priv, IE), no CSR file, checked at decode; fault plumbing already dormant in the CPU signature | PLANNING.md ISA DECIDED; poc.md §1 stage 3 |

Only one of the five questions is genuinely open: **the interrupt seam** ("board checks
pending interrupts at the latch edge, before next fetch"). The docs place the timer +
interrupt controller at day 8 but don't pin the micro-seam. The reviewer's proposal is
sensible and matches the phase-latch design; adopt it as an M3 design note in
`detailed-plan.md` rather than a new decision.

## 4. The two real fixes — done correctly

**1. Make register writes observable.** Real gap, wrong remedy. Option 2 (latch lines for
registers) requires either lying in the trace or restructuring the register file into
posted writes, contradicting a recorded PoC deviation. Take the reviewer's option 1:
a uniform EXECUTE line per state-changing instruction (`EXECUTE cpu alu r2 <= 0 + 5 = 5`).
Cost: one `Logf` per ALU op. Semantics unchanged, trace uniform, tick shapes identical.
If we ever want edge-committed register files, that's day-16 pipelining pressure, not now.

**2. Pick one PC convention.** Real, and broader than reported (halt vs branch/call vs
dump). The invariant worth teaching is "PC = next fetch address"; instruction-position is
derived (`PC-4`). So: label the dump honestly — `pc=0x0024 (next fetch; halted at
0x0020)` — rather than the reviewer's option A, which would print a value the register
doesn't hold (and the Blue-Screen snapshot legitimately wants both). Bring the
`branch`/`call` lines onto the same vocabulary (`at=0x0018 next=0x001c -> 0x000c`).

**3. (Their point, our correction) Device access visibility.** Real: a write to the echo
mailbox is indistinguishable from a RAM write in the CPU-side line. But the proposed
`addr=0x4000 (echo)` suffix is impossible as stated — writes are routed to their target
at the *latch*, not at post time, so the posting line can't know the device. The correct
seam is the one the machine already has: emit the commit with its target —
`latch commit echo@0x4000 <= 0x…` / `latch commit ram[0x0040] <= 0x…`. That makes the
device-decode step visible (their actual goal, and the MMU's future interception point)
*and* gives memory commits per-target visibility, which their option 2 wanted for the
wrong state.

## 5. Polish triage

- `bus_write_post` naming: keep. "Posted write" is the term of art and motherboard.md §1
  documents the exact semantics; the review's reading ("after the fact") is the
  learner-vocabulary gap the primer should close, not a rename. **But** event-kind strings
  are about to be frozen by golden-trace fixtures — any rename (theirs: `bus_write_pending`)
  must land in M2 or never. Default: keep.
- `--dump-full` RAM flag: fine, CLI polish, M2.
- "echo empty line should show what was last written": self-contradictory — empty means
  never written. Best available: `echo@0x4000: empty (mailbox, last value none)`. Cosmetic.
- Tick-number indentation: originally rejected for `trace.txt` (machine-diffable
  export; tick grouping belongs in the TUI per ui.md). **Superseded by owner decision
  2026-10-16** — the export now renders cycle-grouped, indented blocks with per-cycle
  summary comments (`decisions.md` D24); the events table stays untouched, so the
  diffing argument no longer applies.
- Subtract-then-branch explainer for day 4–5: agreed, one sentence in the primer. It's a
  content note, not a code note — there is no `cmp` because flags were deliberately not
  modeled; say that.

## 6. Disposition

Fold into the plan (no doc changes made beyond this file):

- **M2 (before golden fixtures freeze event vocabulary):** ALU EXECUTE line, PC-convention
  cleanup (dump + branch/call lines), latch-commit-per-target event, `bus_write_post`
  keep/rename decision closed, `--dump-full`.
- **M3 design note:** interrupt seam at the phase latch, before next fetch (reviewer's
  proposal, adopted).
- **Day 4–5 primer:** subtract-then-branch / no-flags note.
- Nothing from §2/§3 requires action — the machine already decided those; where the review
  and the code disagree, the code is the ground truth and the docs above cite it.

The review's net assessment ("mostly polish, not architecture") is fair, and its two
priority fixes were the right two to pick — even though its reasoning for fix #1 was
based on a timing model this machine deliberately doesn't have, and it missed that its
closing "fork in the road" (minor ticks) was paved in motherboard.md §1 before the PoC
was written.
