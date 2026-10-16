# AmmaBoard

The AmmaBoard-1 PoC: a tiny deterministic machine simulator in Go, chapter one of a course
that teaches how computers actually work — from punched cards to M-series — by making you
build every piece. *(Planning docs live in `../motherboard/`.)*

This PoC proves the **motherboard architecture** (`../motherboard/motherboard.md`):

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

## Tests

```sh
go test ./...
```

They assert the PoC's five acceptance criteria, including **determinism** (two identical runs
produce identical event streams) and **pluggability** (a new device wired with zero edits
elsewhere).

## Status

PoC. Next: phase-set confirmation, chapter engine, the TUI, and the advent calendar — see
`../motherboard/PLANNING.md`.

License: Apache-2.0.
