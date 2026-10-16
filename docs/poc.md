# PoC Notes

Living document for the PoC → MVP → beta implementation work.
Scope lock (session 7): **content through day 11** ("your own MMU") ships Dec 2026.
Play days 1,2,4,5,7,8,10,11 (8 chapters); buffer days 3,6,9,12; days 13+ = labeled locked doors.

---

## 1. Instruction inventory needed to reach MMU day (day 11)

Question: how many *primitive* (encoded) instructions must exist by day 11?
Answer: **19**. Grown in three stages, each instruction justified by a specific chapter need.
(Assembler pseudo-ops — `li mov nop call ret jmp bge bgt not neg` — are sugar, not counted.)

### Stage 1 — days 4–5 (stored program, assembler): 10 instructions
| # | Instr | Needed for |
|---|---|---|
| 1 | `load_word` | load word |
| 2 | `store_word` | store word |
| 3 | `add` | arithmetic |
| 4 | `add_number` | arithmetic + pointer bump (sp walk day 7) |
| 5 | `subtract` | arithmetic |
| 6 | `jump_if_equal` | loops |
| 7 | `jump_if_not_equal` | loops |
| 8 | `call` | jump & link (calls, day 7; return addr → fixed r7) |
| 9 | `goto` | return / indirect jump (day 7) |
| 10 | `halt` | clean termination for snapshots |

Turing-complete. Days 1–3 (relay/cards) need **zero** instructions — narrative + mechanical sim.

### Stage 2 — days 7–8 (subroutines, interrupts): +10 → 20 total at day 8
| # | Instr | Needed for |
|---|---|---|
| 11 | `load_byte` | byte strings (day 7); PTE field extraction (day 11) |
| 12 | `store_byte` | byte strings |
| 13 | `shift_left` | building addresses/flags |
| 14 | `shift_right` | logical shift for field extraction |
| 15 | `shift_right_arithmetic` | sign-preserving arithmetic |
| 16 | `bit_and` | masking (VPN/offset split, day 11) |
| 17 | `bit_or` | composing PTEs/addresses (day 11) |
| 18 | `allow_interrupts` | interrupt enable (day 8) |
| 19 | `forbid_interrupts` | interrupt disable (day 8) |
| 20 | `return_from_interrupt` | interrupt return (day 8) — **decided: own instruction** (fold-into-
      goto = two-personality jump, hidden state; violates simplicity maxim) |

### Stage 3 — days 10–11 (protection, MMU): +1 → **21 total**
| Instr | Needed for |
|---|---|
| `flush_tlb` | TLB invalidation on page-table change (day 11) |
| privilege bit (status reg) | user/kernel modes; page-walk runs privileged |
| fault plumbing (trap vector + fault type/vaddr/pc) | day 10 limit violation, day 11 page fault —
  built **dormant into the PoC CPU** (execute-loop signature carries Fault from day one) |

Key observation: **base+limit (day 10) needs no new instructions** — pure hardware. The page-table
walk (day 11) needs nothing new either — it's just `lb/sll/and/or` already on the machine. The
memory system stays instruction-poor by design; that IS the lesson.

**Bottom line: 21 encoded instructions by day 11, 1 status register, fault plumbing.**
Everything the rogue/nice acceptance test needs is on the machine by day 8.

## 2. Machine components needed by day 11
- CPU (boot-ten → stage-2 → stage-3 ISA), registers, pc, status reg (priv, IE)
- Bus + RAM (byte-addressable, 32-bit space; early chapters use low KBs)
- Timer + interrupt controller (day 8)
- Base+limit unit (day 10)
- Page-table walker + TLB + `tlb.flush` (day 11) — walk authored by learner in toy ISA,
  consulted as privileged routine
- Fault plumbing: trap vector, fault register (type, vaddr, pc) — dormant from PoC

## 3. PoC scope — NARROWED (session 9): motherboard first — **DONE (built in ../AmmaBored)**
PoC = **motherboard + boot-ten instructions + everything dumped to a file**. Build on top later.
Excluded: chapter engine, bubbletea UI, progress/save, rogue/nice payloads, TLB/cache/BaseLimit.

**Status: all five criteria pass** (`cd ../AmmaBored && go test ./...`, plus CLI smoke):
1. Program runs — `examples/counter.amm`: 9 words, 25 major ticks, clean halt ✓
2. Trace complete — SQLite `.db` + `trace.txt` export; BOOT/FROZEN snapshots ✓
3. Determinism — `TestDeterminism`: two runs, identical event streams ✓
4. Observability — every fetch/decode/execute/bus/latch logged with (major, phase, component) ✓
5. Pluggable — `EchoDevice` via one `Attach` in `cmd/ammabored/wiring.go`, zero edits elsewhere ✓

Deviations from design docs (deliberate, PoC-only): no prev/current double-buffering (the tick
log is the history); phases = FETCH/DECODE/EXECUTE only (the accepted 99% set, confirmed by use).

**Definition of "works" (each observable):**
1. A program runs: boot-ten asm → assembled → loaded → executed to `halt`.
2. Trace is complete: SQLite log has every event; `trace export` = readable text dump;
   state at any tick reconstructible by replay.
3. Determinism enforced: same program twice ⇒ byte-identical event tables; a deliberately
   broken phase order is caught by the test harness.
4. Observability real: from the dump alone, answer "what was r3 at tick 40.2?" without re-run.
5. Components pluggable: wire a fake echo-device via `main/wiring.go` with ~20 lines and zero
   edits elsewhere — proves the component contract, the thing everything else builds on.

## 4. Open questions (PoC)
1. ~~`return_from_interrupt` vs fold into `goto`~~ — decided session 10: own instruction.
2. TLB semantics granularity: per-1KB-page only, or also a "walk always" debug mode? (Pages are
   1KB in our machine — see motherboard.md §5.)
3. ~~Snapshot diff format~~ — resolved session 10: Blue Screen freeze snapshot (see ui.md §9).
