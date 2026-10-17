# Implementation

## Layout
```
cmd/ammaboard/            CLI entry; wiring.go constructs components and Attaches them
internal/isa/             opcodes, encoding, Specs table, disassembler, parity suite
internal/asm/             two-pass assembler, form-driven (no per-mnemonic knowledge)
internal/machine/         broker + contracts (machine.go), motherboard (motherboard.go)
internal/machine/cpu/     CPU: one file per instruction, registry, execute dispatch
internal/machine/ram/     RAM component
internal/machine/timer/   day-8 timer MMIO device
internal/machine/echo/    demo echo device (pluggability proof)
internal/machine/bluescreen/  fault panel renderer
internal/ui/              TUI (bubbletea v2): three panes + shell over the live machine
internal/ticklog/         SQLite event/snapshot log + text export
```

## Rules
- **Package per component**; the package boundary enforces bus-only interaction.
  Entangled sub-parts (TLB + walker) stay in one package — component, not concept.
- `internal/machine` root holds only broker + contracts. Board-level integration tests
  live in an external `machine_test` package so the root imports no components.
- Interfaces and packages are named as if third parties will read them.

## Adding an instruction
1. A row in `isa.Specs` (opcode, mnemonic, form).
2. A file in `internal/machine/cpu/` registering an executor in `init()` — duplicate
   registration panics at startup.
3. Tests. If the form is new: one encoder in `asm` plus a canonical case in the parity
   suite.

The execute loop is untouched. If the form already exists, the assembler needs no change.

## Adding a device
Implement `Component`, plus whichever of `Reader`, `Writer`, `ByteReader`, `ByteWriter`,
`InterruptLine` apply, then `Attach` it in `cmd/ammaboard/wiring.go`. Zero edits anywhere
else — the echo device is the standing proof.

## Tests
- Table-driven per component; every instruction gets golden register/memory cases.
- `TestDeterminism`: two runs ⇒ byte-identical event tables.
- `internal/isa/parity_test.go`: assembler ↔ Specs ↔ CPU parity.
- Goal: one golden-trace fixture per chapter (diff ordered `SELECT * FROM events`).

## Storage (ticklog)
One SQLite file per run (`modernc.org/sqlite`, pure Go — keeps `CGO_ENABLED=0`). Owned by
the run, never by the machine: flush semantics cannot affect machine behavior.

```
runs(id, chapter, machine_version, schema_version, started_at)
events(id, run_id, tick_major, phase, component, kind, detail)
snapshots(id, run_id, tick_major, tick_minor, kind, state_blob)
```

- `machine_version` stamps the generation ("AmmaBoard-2"). `schema_version` = 1; `Open`
  migrates legacy files in place (column added, old rows stay 0); read-side exporters
  treat a missing column as 0 and refuse files written by a newer schema. Bump the
  constant whenever the schema changes.
- Snapshot kinds: `initial` (BOOT), `final` (FROZEN on halt/fault, PAUSED on budget cut),
  `bluescreen` (frozen fault panel).
- Event kinds — **frozen once golden fixtures exist**: CPU emits `fetch decode alu load
  load_byte store store_byte branch call goto halt rfi status fault`; board emits `major
  latch commit bus_read bus_write_post bus_read_byte bus_write_byte_post irq irq_cleared
  interrupt fault`.
- Detail strings carry `(was N)` old-value visibility; numbers show every interpretation
  (unsigned/signed/hex) when they diverge.

## Text export
`trace export` / the CLI's `-trace` writes one human-first file per run: a legend, then
per cycle a comment line (number + what it accomplishes, longhand terms like
`register_6`, `program counter`), a dashed rule, and indented event rows (terse,
greppable). The events table is untouched — rendering lives only in `ticklog.Export`.
No custom binary trace format, ever.

## CLI
```
ammaboard run program.amm [-o ammaboard.db] [-trace trace.txt] [-max 10000] [-echo] [-dump-full]
ammaboard tui [program.amm]
```
- Default wiring: RAM 16KB at 0x0000, timer at 0x4020, entry 0x0000.
- `run` is the batch path: full SQLite log + text export + verdict (see above).
- `tui` is the interactive path (docs/ui.md): three panes + shell; no file = built-in
  counter demo. Needs ≥141×26; speed = major ticks per frame (default ×1).
- `-max N`: cycle cap. Default 10000 (~1s worst case including log + export — a runaway
  fails fast and cheap). `0` = uncapped; negatives rejected.
- `-echo`: wires an echo device at 0x4000. `-dump-full`: print every RAM word, not the
  changed ones.
- Output: verdict line, final machine state; on a fault also the Blue Screen, frozen into
  the log. Budget cut prints `paused: budget exhausted (program did not halt)` with a
  `PAUSED` snapshot.
