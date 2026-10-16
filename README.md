# AmmaBoard

The AmmaBoard-1 PoC: a tiny deterministic machine simulator in Go, chapter one of a course
that teaches how computers actually work — from punched cards to M-series — by making you
build every piece. *(Planning docs live in `docs/`.)*

This PoC proves the **motherboard architecture** (`docs/motherboard.md`):

- **Elm-like components** — each component is a plain object with a `Tick(phase)`; the
  motherboard is the broker. No goroutines, no channels, no locks: same inputs + same cycles
  = identical machine state, always.
- **Phased clock** — every major tick (one cycle) runs named minor phases
  (`FETCH → DECODE → EXECUTE`); bus writes are posted and become visible only at the
  phase latch. Read-old/write-new is explicit, like real hardware.
- **Everything is logged** — every bus read, write, decode, branch, and fault lands in a
  SQLite tick log (pure Go driver, single `.db`). `trace.txt` is the human-readable export.
- **Refuses nonsense** — unknown opcodes and unaligned word accesses halt the machine with a
  fault record (kind, pc, address, phase). Faults are data, not exceptions.
- **Pluggable** — new hardware is one type + one `Attach` call in `cmd/ammaboard/wiring.go`.
  Try `-echo`.

## Run it

```sh
go run ./cmd/ammaboard run examples/counter.amm -echo
```

You'll see the final registers and memory, plus two artifacts:

- `ammaboard.db` — the full tick log (query it: `sqlite3 ammaboard.db 'select * from events'`)
- `trace.txt` — the same trace, readable line by line

## The boot ten (AmmaBoard-1)

32-bit words, 8 registers (`r0` is hardwired zero, `r7` is `call`'s link register),
encoding `[opcode:5 | rd:3 | rs:3 | rt:3 | immediate:18]`, destination-first grammar:

```asm
load_word  r1, 4(r2)          ; r1 = memory[r2 + 4]
store_word r1, 4(r2)          ; memory[r2 + 4] = r1
add        r1, r2, r3         ; r1 = r2 + r3
add_number r1, r2, 10         ; r1 = r2 + 10
subtract   r1, r2, r3         ; r1 = r2 - r3
jump_if_equal     r1, r2, loop
jump_if_not_equal r1, r2, loop
call       sub1               ; pc -> sub1, r7 = return address
goto       r3                 ; pc = r3
halt
```

## Stage 2 (AmmaBoard-2)

Days 7–8 hardware: bytes, shifts, bitwise ops, and interrupts — the machine grows, the
encoding doesn't (same 32-bit word):

```asm
load_byte  r1, 3(r2)          ; zero-extended byte; no alignment needed
store_byte r1, 3(r2)          ; low byte only
shift_left / shift_right / shift_right_arithmetic  r1, r2, r3   ; amount = r3 & 31
bit_and / bit_or           r1, r2, r3
allow_interrupts / forbid_interrupts / return_from_interrupt    ; no operands
```

A timer at `0x4020` raises an interrupt line every N major ticks (`+0` = interval,
`+4` = pending/acknowledge). Delivered lines aim pc at the trap vector `0x0008` with
interrupts auto-forbidden; `return_from_interrupt` resumes. See it run:

```sh
go run ./cmd/ammaboard run examples/timer.amm
```

Trace highlights: every register change prints an `alu` line, latch commits name their
target (`commit ram[0x0040] <= 0x…`), and the halt line shows both PC conventions.
`-dump-full` prints the whole RAM instead of nonzero words.

## Tests

```sh
go test ./...
```

They assert the PoC's five acceptance criteria, including **determinism** (two identical runs
produce identical event streams) and **pluggability** (a new device wired with zero edits
elsewhere).

## Status

PoC + ISA stage 2 (bytes/shifts/interrupts, timer, trace-vocabulary fixes — see
`docs/decisions.md`). Next: chapter engine, the TUI, and the advent calendar — see
`docs/PLANNING.md` and `docs/detailed-plan.md`.

License: Apache-2.0.
