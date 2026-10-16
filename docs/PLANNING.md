# Motherboard — Planning Notes

Working title: **motherboard** (from repo dir; confirm).
Living document. Append one section per session. Do not rewrite history; add dated entries.

---

## Session 1 — 2026-10-14

### Vision (one line)
An interactive terminal course — built in Go with Bubbletea — that teaches how CPUs and
low-level systems actually work, from punched cards to Apple M-series, by having the learner
*build and fix* each piece themselves. NandToTetris, one level up, zero ceremony (no QEMU,
no toolchain installs — one binary).

### Why this works (prior art + gap)
- **NandToTetris** proved "build it to understand it", but stops at a simple Hack machine + toy OS.
  No MMU, no caches, no multicore, no performance intuition.
- **CS:APP / Drepper's "What every programmer should know about memory"** cover the memory
  hierarchy but are passive: books, labs with heavy ceremony.
- **Turing Complete / Human Resource Machine** games prove interactive hardware teaching works,
  but don't connect to real-world dev concerns (cache thrashing, false sharing, kernel reading).
- Gap: an interactive path from "what is a stored program" to "why is my Go service 40x slower
  when the array is 33M elements instead of 32M", with history as the *reason each invention existed*.

### Differentiators (stated)
1. History taught as *necessity*: each component introduced by the pain it solved
   (two programs clobbering each other → MMU; CPU–memory speed gap → cache).
2. Zero ceremony: single static Go binary, TUI, progress saved locally.
3. Exercises that produce real artifacts: your own MMU code, your own cache policy,
   your own scheduler — run against the same CPU core the app uses.
4. Ends with a bridge to the real world: "here is the map into Linux kernel source /
   perf / M-series unified memory" — not "here is a toy, goodbye".

### Product format
- **Advent calendar**: 24 doors, Dec 1–24, but **16 chapters** on a 2-play/1-rest cadence
  (see curriculum of record). Play door ≈ 30–60 min; rest doors = curated deeper reading.
- Doors date-gated by default, with a `--free` mode for testers/reviewers.
- Local progress persistence (state file under user config dir).
- Day 24 is finale + "where to go next" epilogue, not a new chapter.

### Curriculum of record — 16 chapters, 2-play/1-rest cadence (session 4, refined session 5)
75 years in 24 hectic days rejected. Cadence: play days 1,2,4,5,7,8,10,11,13,14,16,17,19,20,22,23;
rest days 3,6,9,12,15,18,21,24. **Rest days carry the curated articles + deeper primers**
("dig deeper" is a designed feature). Day 24 = finale/epilogue.
Audience: ANY dev with ≥2 years coding experience — not intermediate/expert systems folks.
Each chapter = one relatable leap, "one X is good → what about many X?":

| Day | The leap | Exercise (you build) | Year |
|---|---|---|---|
| 1 | No program → a program is wiring you re-solder | Patch a plugboard to fix a bug | 1944 |
| 2 | Fixed wiring → a program you can shuffle (cards) | Reorder a deck, watch control flow change | 1945 |
| 4 | Program outside the machine → one program living in memory | Hand-assemble + run on the boot-ten machine | 1949 |
| 5 | One program written painfully → many written easily | Write the assembler | 1951 |
| 7 | One long blob → many reusable pieces | Calling convention: jal/jalr + stack | 1951 |
| 8 | CPU does one thing → many things, seemingly at once | Timer interrupt handler | 1954–58 |
| 10 | One program in RAM → many in RAM (one bug kills all) | Watch two programs collide; add base+limit | 1961 |
| 11 | Many sharing memory → each believes it has its own | **Your own MMU**: page-table walk + TLB | 1962 |
| 13 | CPU fast, memory slow → keep nearby data close | Direct-mapped cache, live hit rate | 1968 |
| 14 | Cache that works → one that thrashes (and the fix) | Reproduce thrashing; set-assoc + LRU | 1970s |
| 16 | One instruction at a time → many in flight | Pipeline hazard toy; branch predictor | 1978→1990s |
| 17 | One core → many cores that lie to each other | False sharing; amoswap; toy MESI | 1983/2005 |
| 19 | CPU does everything → devices do work | MMIO handshake; PIO vs DMA timing | 1970 |
| 20 | CPU watches devices → devices notify CPU | Interrupt-driven ring-buffer driver | 1980s |
| 22 | One core type → many core types (battery vs burst) | P/E-core scheduler, M-series-style SoC | 2020 |
| 23 | Everything separate → one machine booting YOUR kernel | Capstone on the full machine | — |

Multi-user note: no dedicated door. Folded into days 10–11 (protection → privilege bit) and
day 22 (your workload vs background tasks). Cheapest explicit option if wanted: rename day 10
primer to "many programs, many users" rather than adding a chapter.

Rest-day reading pairings: 3 (relay-era memoirs), 6 (ENIAC/von Neumann "First Draft"), 9
(CTSS/early timesharing), 12 (**"Look Amma, I made my own Blue Screen of Death"** — the day-12
buffer output: a BSOD is a kernel's dying attempt to explain itself — a wall of hex gibberish
for anyone who didn't build the machine. Fresh off day 11, learners make their machine crash
on purpose, each way (page fault, limit violation, bus conflict), and their Blue Screen emits
the same classic hex wall — which they can actually READ, because they built every byte of it.
Plus the tour: the million other causes of blue screens and what each meant — bad memory, bad
drivers, races), 15 (**Drepper's memory paper** — after caches),
18 (MESI + false-sharing writeups), 21 (Apple M-series / big.LITTLE material),
24 (map into real Linux code, perf, kernel contribution guides).


### Technical sketch (draft)
- Go module, single binary. `bubbletea` + `lipgloss` + `bubbles` (viewport, list, textinput, progress).
- Packages (draft):
  - `internal/machine`: `Bus`, `RAM`, `CPU` (fetch/decode/execute loop, pluggable `Translate`),
    `Cache` (Device wrapping RAM), `MMU`, interrupt controller, timer, simple devices.
  - `internal/chapters`: data-driven chapter definitions (embedded), each declaring which
    machine components are active + exercise stubs + success criteria.
  - `internal/ui`: bubbletea model(s): door grid, narrative pane, machine view (registers,
    memory grid, cache stats), exercise editor.
  - `internal/progress`: local save state.
- Determinism: seeded PRNG, explicit step counts → reproducible traces, easy golden tests.
- Sim runs on tick `Cmd`s inside bubbletea `Update`; interactive speeds only, Go is plenty.

### ISA — DECIDED (session 3, 2026-10-14)
Custom toy ISA ("custom" decision from session 1 resolved). Design rules:
1. Fixed 32-bit words everywhere (registers, addresses, fixed-length 32-bit encodings).
   Early chapters only use the low KBs, but the machine never changes shape — MMU day needs
   real 32-bit VA + 4KB pages, cache day needs 64B lines. No 16-bit→32-bit migration ever.
2. Load-store RISC, 8 regs `r0`–`r7`; `r0` hardwired zero (buys immediate zero, NOP, reg-jumps).
3. ISA grows only when a chapter's pain demands it; comfort = assembler pseudo-ops introduced
   AFTER the learner has hand-written the painful expansion once.
4. Three encodings only: R, I, B — keeps the "write an assembler" exercise (day 5) doable.

**Stage 1 "boot ten"** (days 4–5, Turing-complete) — plain-language names, snake_case, locked:
`load_word store_word add add_number subtract jump_if_equal jump_if_not_equal call goto halt`

**Encoding (session 10):** first byte = **5-bit opcode + 3-bit destination register**; 3 bits per
register (8 regs); **18-bit immediate**; spare bits must be zero (reserved — machine faults on
nonzero, a teachable fork). 32 opcode slots; **leftover slots = syntactic sugar encodings**
(sugar decodes in-machine as aliases of primitives; sugar must ALWAYS expand to existing
primitives — never new semantics). Candidates: `nop move ret jump increment decrement not negate
jump_if_greater_or_equal`, ~8 slots, 1–2 permanently reserved.

**Growth stages** (pain-driven, mapped to the 16-chapter curriculum):
| Stage | Days | Adds | Rationale |
|---|---|---|---|
| 1 | 4–5 | boot ten | stored program, first assembler |
| 2 | 7–8 | `load_byte store_byte shift_left shift_right shift_right_arithmetic bit_and bit_or allow_interrupts forbid_interrupts return_from_interrupt` | strings, masking, interrupts |
| 3 | 10–11 | `flush_tlb`, privilege bit in status reg | MMU; learner authors page-walk in same ISA as privileged routine |
| 4 | 13–14 | `multiply` | cache workloads; cache adds no instructions (a teaching point itself) |
| 5 | 17 | `swap_atomic add_atomic` | coherence, false sharing |
| 6 | 22 | none | P/E scheduling is pure software — that IS the lesson |

Key pattern: memory-system evolution (MMU/cache/NUMA/MMIO) adds almost no instructions —
true of real hardware too; make it an explicit teaching point. Devices are just addresses.

**Sugar layer**: sugar gets real opcode encodings (see Encoding above) but expands to primitives;
`ret` = `goto r7` alias etc. Fixed link register **r7** for `call` (phone always rings at the
same desk); generality becomes sugar later.

**Operand grammar (locked)**: destination-first, 3 operands max; address syntax `offset(base)`;
`store_word r1, 4(r2)` = "store r1 TO that address" — consistent order everywhere; the real-world
reversed-store-order surprise costs a developer almost nothing and is called out in the primer.

**System model**: one status register (privilege bit, IE flag) + system instructions.
No CSR file — a full CSR space is fidelity we don't need; a flag bit carries the lesson.
Two's complement from day one.

### Exercise medium — DECIDED (session 2, 2026-10-14)
Learners write **toy assembly running on the machine itself** (NandToTetris-style): e.g. implement
the page-table walk / cache policy in the machine's own ISA, plugged into the CPU.
- Most faithful to "your own MMU code"; zero extra machinery (no interpreter, no version pinning).
- No sandboxing needed — single binary downloaded/updated from GitHub releases; learner code runs
  on their own machine, worst case they crash their own TUI. ~~yaegi / Go plugs~~ rejected: wasted
  energy, sandbox escape risk was only relevant for in-process interpreted Go.
- Consequence: exercise editor = assembly text input + step/run against the live machine.
- Kernel/scheduler-era exercises likewise live in the toy ISA ("write the kernel" is the capstone).

### Focus statement (session 2)
Energy goes to **content, not infrastructure**: the game itself, the per-chapter primers (short
written explainers shipped with each door), and curated external articles per chapter.
Leaderboard / shared features are a deliberately separate, later concern (see logistics).

### Challenge format — DECIDED (session 5)
Inverted exercises: the app ships the **workload** (fixed programs in toy binary), the learner
builds/modifies the **machine or kernel** to make it behave. Machine is the artifact — matches
day-23 capstone and needs no sandbox.
- Payload format: one line of hex per 32-bit word, tiny header (`origin, length, entry`).
  Forces learners through their own day-5 assembler to inspect payloads.
- Success = diffable machine state: register/memory snapshots, device output, cycle counts,
  AND **fault events as first-class data** (trap vector + status: fault type, faulting vaddr/pc).
  Small hardware addition to ISA stage 3; doubles as the lesson: an OS is a fault handler.
- Dual criteria (behavior + snapshot) to avoid blind pattern matching. Cycle budgets loose
  (~10x reference) — a nudge, not anti-cheat.
- Ramp: days 1–8 snapshot + output; day 10+ multi-program payloads with fault expectations
  (rogue/nice pair reused: day 10 limit-violation, day 11 page-fault); days 13–14 same program,
  different hardware config, success = hit-rate numbers; day 22 same payloads, different
  core-type assignment.
- Resolves the "chapter file format" open decision: chapter = payload binaries + initial
  machine config + expected snapshots/traces + primer + curated links.


### Pain catalogue for primers (session 5)
**Security:** rogue program reads/writes another → MMU/paging/rings (1961–62); buffer overflow →
stack canaries, NX (~2000s), ASLR; W^X / execute-only pages; cache side channels (flush+reload) →
cache partitioning, constant-time coding; **Spectre/Meltdown (2018)** → KPTI/retpolines —
security visibly taxing performance industry-wide; Rowhammer (2014); device DMA bypasses MMU →
IOMMU (VT-d/AMD-Vi); privilege escalation → SMEP/SMAP; freed-memory secrets → ARM MTE / Apple
memory tagging.
**Performance:** CPU–RAM gap (1x→>100x cycles) → L1/L2/L3, prefetch, TLB-as-cache — most modern
perf wisdom derives from this; interrupts (don't poll), DMA (don't copy); pipelining (1978),
branch prediction (1990s), OoO/speculation (Pentium Pro 1995) → later Spectre; context-switch
MMU flush cost → TLB ASID/PCID tags; private caches + shared memory → MESI (1983) + **false
sharing** (THE bug modern devs hit); weak memory ordering → barriers/atomics (TSO vs ARM);
lock contention → atomic cost; discrete-GPU copies → **M-series unified memory**; heterogeneous
cores → big.LITTLE (2011) / M1 P+E (2020) — scheduling problem moves to software, no new ISA.
**Meta-pattern (primer spine): every security fix answers a perf/sharing feature; every perf
feature becomes a security liability** (caches→side channels, speculation→Spectre,
DMA→IOMMU). Use for rest-day readings.

### Key open decisions (gate everything else)
1. ~~ISA~~ — decided session 3: custom, fixed 32-bit, load-store, boot-ten start, staged growth
   (see ISA section).
2. ~~Exercise medium~~ — decided session 2: toy ISA only, no yaegi/DSL.
3. ~~Content pipeline / chapter format~~ — largely resolved session 5: chapters are data
   (payload binaries + machine config + expected traces + primer + links); see Challenge format.
4. ~~Persona calibration~~ — resolved by the one→many leap framing (session 5): chapters never
   assume systems vocabulary; remaining detail (day 4–5 editor scaffolding) is a UI draft choice.
5. ~~Advent target year~~ — DECIDED session 6: launch Dec 2026, beta = doors 1–11.

### Risks (named)
1. **Content velocity is the project.** The engine is maybe 30% of the work; 16 polished,
   interactive chapters + 8 rest-day reading sets is the product. Mitigate: chapter template +
   heavy component reuse; build engine to make a new chapter cheap.
2. **History bloat.** Passive narrative kills the format. Cap history framing (~30% of a
   chapter max); every history beat must motivate a mechanic the learner then touches.
3. **Scope gravity.** 75 years of computing wants to become 60 chapters. Hard cap: 16 chapters
   in 24 doors (2-play/1-rest), cut by "does this make you a better low-level engineer",
   not by "is this famous".
4. ~~Sandbox escape~~ — moot (session 2): no in-process learner Go code; exercises run in the
   toy ISA on the user's own machine via their own downloaded binary.
5. **Deadline risk (advent)**: a date-locked launch punishes slippage publicly. Prefer
   shipping doors 1–11 polished over 24 rushed; free-mode lets late doors unlock via flag.

### Advent logistics notes
- Today 2026-10-14. Dec 1 2026 ≈ 7 weeks. Realistic: PoC (core machine + 2 chapters) in ~1–2 wks
  of focused work; beta = doors 1–11 (through **your own MMU**, day 11 — the strongest hook)
  feasible for Dec 2026 if content pipeline is built first; full 16 for Dec 2027.
- Launch shape: GitHub repo (Apache-2.0 — repo, machine, and chapters code), single-binary
  releases via GitHub (self-update command), doors unlock by date; spoiler flag.
  Machine name: **AmmaBoard-{generation}** (see motherboard.md §2); day-12 campaign
  "Look Amma, I made my own Blue Screen of Death."
- **Leaderboard / shared features: separate app later** (session 2 decision) — simple
  Cloudflare Workers deployment, decoupled from the game binary; the TUI stays offline-first,
  submission optional/explicit. Not on the critical path for PoC/MVP/beta.

### Content quality bar (session 2)
Each chapter = game + **primer** (short shipped explainer: the pain, the invention, the mechanic)
+ **curated reading list** (2–4 links: e.g. Drepper for cache days, Atlas papers for MMU days,
  Apple's M-series material for days 21–22). Primers are part of the deliverable per door, not
  an afterthought — content velocity planning must budget for writing them.

### Critical analysis + monetization (session 6)
**Advent year DECIDED: launch 2026**, chapters cut if needed. Declared beta scope:
**doors 1–11** (8 chapters, ends on *your own MMU*); days 12+ ship as labeled locked doors
("unlocks 2027"). Locked-with-label is honest; broken doors are not.
**Top risks ranked:** (1) content vs 7 weeks — scope-cut decision needed NOW, not Nov 20;
(2) day-1 experience is the storefront (screenshots come from doors 1–2; prototype them first);
(3) attribution accuracy — every primer needs a 20-min source check, HN fact-checks for free;
(4) TUI viz ceiling — numbers and grids, never animation (hit-rate counters > blinking cache
lines); (5) discoverability is the existential risk, not quality — launch plan (Show HN,
r/programming, AoC-adjacent communities) needs design like the engine does. Attrition: most
advent players quit by day 5.
**Monetization (ranked, honest):** free 2026 launch is the marketing asset, never paywalled.
1. Paid Season 2 / extended course 2027 (license key in-app) — best structural fit, $0–5k/yr.
2. Book of expanded primers (NandToTetris pattern: free course, paid book), $0–3k.
3. GitHub Sponsors — trivial, expect little.
4. Corporate perf-engineering workshops derived from material — highest $/hour, reputation-
   dependent.
5. Merch: skip.
Real ROI = reputation compounding (talks, consulting, kernel credibility), not direct revenue.
**Opening note ends with**: "Someday, someone will run DOOM on AmmaBoard." (North star for
contributors: DOOM needs interrupts, framebuffer-ish memory, `multiply`, honest cycle timing —
the day-13–17 machine has all four. Also the FPGA-kit flag, and full circle: DOOM made
millions of kids care about CPUs; AmmaBoard explains them.)
**Ad angle**: "Play the slowest DOOM ever: 1 FPT — one frame per thought." Flips perf shame
into the feature; literally true in our UI (stepping = thinking is the throttle; the learner's
comprehension sets the frame rate).
**End condition**: AmmaBoard stops when the community can play DOOM on it — one frame per
comprehension. Not downloads, not stars: a stranger, unaided, understanding the machine well
enough to make it play. That is "done."
**Sign-off**: "Every bit yours, everyone's invited." (Triple pun: letter-closing devotion,
Apache-2.0 promise — every bit of the machine is yours — and the curriculum's own truth:
everything is just bits. Everyone's invited = learner, contributor, DOOM porter, Amma's whole
family.)


### Session 7 notes
- Beta scope locked: **through day 11** (8 chapters; buffer days 3,6,9,12; halfway to 24).
- `poc.md` created: 19 encoded instructions to MMU day; component list; PoC acceptance test
  (rogue/nice, run with and without base+limit).
- `motherboard.md` created: execution model DECIDED — single-threaded cycle loop + event
  sourcing; goroutine-per-component REJECTED (nondeterminism breaks snapshot diffs; toy bus
  must be synchronous). One channel boundary: sim → UI snapshots. Multicore = N CPUs stepped
  round-robin (simulated concurrency, deterministic).


### Sidequest — encoding & "new version mode" (session 9, jotting; not discussed deeply)
Buffer-day pointer: **why UTF-8 is the way it is** (variable-length, backward compatibility
with ASCII, self-synchronization), what could be slightly improved (fixed-width alternatives,
sorting/comparison quirks, validation cost) — a nice "design under constraint" story that pairs
with the punched-card/byte era chapters.
Machine idea: a **CPU version-mode register** — players flip the machine into an experimental
"v2 mode" to try ISA/semantic variations safely without breaking the main line. Mechanics TBD;
candidate content for a buffer day or season 2.

### Next session agenda (session 8: PoC → MVP → beta plan)


1. ~~Lock ISA decision~~ — done session 3 (see ISA section).
2. ~~Exercise medium spike~~ — resolved session 2: toy ISA only.
3. Draft PoC → MVP → beta implementation plan: PoC = motherboard + boot ten + trace dump
   (superseded detail — see poc.md §3 for the live scope).
4. Prototype doors 1–2 FIRST (storefront risk).
5. Source-check pass on all historical attributions before beta.

### PoC → MVP → Beta plan
Written 2026-10-16: see **[detailed-plan.md](detailed-plan.md)**.
