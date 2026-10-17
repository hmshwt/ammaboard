# Roadmap

Goal: **beta Dec 1 2026 = doors 1–11** (8 chapters, through *your own MMU*). Full 16
chapters = Dec 2027. The date is hard; an honest slip ships 1–8 polished + `--free`,
never broken doors.

## Status (2026-10-17)
| Done | Missing |
|---|---|
| Broker motherboard: phases, posted-write bus, event-sourced log | Chapter engine (`internal/chapters`) |
| ISA stages 1–2 (20 instructions) + assembler + parity suite | Progress save, date-gating, release build |
| Timer + interrupt delivery (level-triggered, shadow registers, trap vector) | Stage 3: base+limit, PRIV, MMU/TLB, `flush_tlb` |
| Stop/fault model, Blue Screen (CLI), dropped-write faults, budget pause | Rogue/nice payloads + acceptance checker |
| SQLite log with `schema_version`, trace export v2 | TUI chapter views (fault overlay, timeline, editor) |
| Determinism, parity, integration tests; CLI + TUI v1 (three panes + shell) | Primers + rest-day reading sets (doors 1–11) |

## Milestones
| # | Milestone | Window | Status |
|---|---|---|---|
| M1 | Plan + specs locked | by Oct 23 | done |
| M2 | Chapter engine + TUI shell + doors 1–2 | Oct 24 – Nov 8 | **next** — event vocabulary is ready to freeze into golden fixtures |
| M3 | ISA stage 2 + timer/interrupts + chapters 4/5/7/8 | Oct 27 – Nov 15 | machine done; chapter content remains |
| M4 | Day 10–11 machine: base+limit, PRIV, MMU/TLB | Nov 2 – Nov 20 | |
| M5 | Rogue/nice payloads + acceptance checker | Nov 14 – Nov 24 | |
| M6 | Content: primers, readings, source checks | Oct 27 – Nov 24 | parallel, per door |
| M7 | Launch plumbing + hardening → beta | Nov 17 – Nov 30 | |

## Open decisions (unspecified seams)
1. **Chapter file format** — payloads + machine config + expected traces + primer + links,
   as one data-driven definition. Blocks M2; needs a one-page spec.
2. **Day-5 assembler input path** — what the learner types into and how the editor hands
   source to the assembler.
3. **Day-11 walker calling convention** — how the CPU consults the learner-authored
   privileged page walk: register contract, fault-on-fault semantics.
4. TUI keymap; textarea vs `$EDITOR`; timeline layout for differing phase sets.
5. Legacy `.db` compatibility policy at beta (migrate vs read-only).

## Risks
1. **Content velocity is the project** (~70% of the work). The engine exists to make a
   chapter cheap; M2–M4 are means, never ends.
2. **Day 1 is the storefront** — screenshots come from doors 1–2; prototype them first.
3. **Attribution accuracy** — every primer gets a 20-minute source check; HN fact-checks
   for free.
4. **TUI viz ceiling** — numbers and grids, never animation; hit-rate counters beat
   blinking cache lines.
5. **Discoverability is existential** — launch comms (Show HN, r/programming,
   AoC-adjacent) are a designed deliverable. Most advent players quit by day 5.

## North star
"Someday, someone will run DOOM on AmmaBoard" — one frame per thought (1 FPT). The
day-13–17 machine has everything DOOM needs: interrupts, framebuffer-ish memory,
`multiply`, honest cycle timing. Done = a stranger, unaided, understands the machine well
enough to make it play.
