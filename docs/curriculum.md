# Curriculum

16 chapters, one relatable leap each ("one X → what about many X?"). Play days
1,2,4,5,7,8,10,11,13,14,16,17,19,20,22,23; rest days 3,6,9,12,15,18,21,24. Day 24 is the
finale, not a chapter.

| Day | The leap | You build | Year |
|---|---|---|---|
| 1 | No program → a program is wiring you re-solder | Patch a plugboard | 1944 |
| 2 | Fixed wiring → a program you can shuffle (cards) | Reorder a deck; watch control flow change | 1945 |
| 4 | Program outside the machine → one program in memory | Hand-assemble + run on the boot-ten machine | 1949 |
| 5 | Written painfully → written easily | The assembler | 1951 |
| 7 | One blob → reusable pieces | Calling convention: call / goto r7 + stack | 1951 |
| 8 | One thing at a time → seemingly many | Timer interrupt handler | 1954–58 |
| 10 | One program in RAM → many (one bug kills all) | Two programs collide; add base+limit | 1961 |
| 11 | Shared memory → each believes it has its own | **Your own MMU**: page walk + TLB | 1962 |
| 13 | CPU fast, memory slow → keep data close | Direct-mapped cache, live hit rate | 1968 |
| 14 | A cache that works → one that thrashes | Reproduce thrashing; set-assoc + LRU | 1970s |
| 16 | One instruction at a time → many in flight | Pipeline hazard toy; branch predictor | 1978→90s |
| 17 | One core → cores that lie to each other | False sharing; amoswap; toy MESI | 1983/2005 |
| 19 | CPU does everything → devices do work | MMIO handshake; PIO vs DMA timing | 1970 |
| 20 | CPU watches devices → devices notify | Interrupt-driven ring-buffer driver | 1980s |
| 22 | One core type → many (battery vs burst) | P/E-core scheduler | 2020 |
| 23 | Everything separate → one machine, YOUR kernel | Capstone | — |

Multi-user is folded into days 10–11 (protection → privilege bit) and 22 (your workload
vs background).

## Rest-day pairings
3 relay-era memoirs · 6 ENIAC / von Neumann "First Draft" · 9 CTSS / early timesharing ·
12 the BSOD tour (crash your machine every way, read the hex wall you built) · 15 Drepper's
memory paper · 18 MESI + false-sharing writeups · 21 M-series / big.LITTLE · 24 map into
real Linux, perf, kernel contribution.

Primer spine (security ↔ performance): every security fix answers a perf/sharing feature;
every perf feature becomes a security liability (caches → side channels, speculation →
Spectre, DMA → IOMMU).

## Beta scope
Doors 1–11 (8 chapters), buffer days 3, 6, 9, 12. If the calendar slips: ship doors 1–8
polished, the rest behind `--free`; never broken doors.

## Hardware timeline
| Days | Hardware added |
|---|---|
| 1–3 | none (narrative/mechanical sims) |
| 4–5 | CPU (boot ten), RAM, bus, event log |
| 7–8 | stack is a software convention; timer + interrupt delivery; status register (IE) |
| 10 | base+limit unit (pure hardware, no new instructions) |
| 11 | walker + TLB + privilege bit + trap vector; `flush_tlb` |
| 13–14 | cache (config-driven: direct-mapped → set-assoc, pluggable policy) |
| 16 | pipeline counters |
| 17 | second/third CPU, MESI, `swap_atomic add_atomic` |
| 19–20 | MMIO devices, DMA engine |
| 22 | P/E cores — same ISA, different timing table |
