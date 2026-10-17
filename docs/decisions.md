# Decision log

One line each; full reasoning in `archive/` if needed. All are cheap to reverse until
golden-trace fixtures freeze the event vocabulary — flag objections before then.

## ISA / CPU
- **D1** — Opcode numbers 11–20 assigned in poc.md §1 order.
- **D2** — Shifts and bitwise ops are R-type `rd, rs, rt`; shift amount = `rt & 31`; no
  immediate-shift encoding until sugar needs one.
- **D3** — `load_byte` zero-extends; `store_byte` stores the low byte; byte ops have no
  alignment constraint; word ops keep the 4-byte alignment fault.
- **D4** — Byte access rides the bus via optional `ByteReader`/`ByteWriter` interfaces;
  word-only window owners answer `NOT_PERMITTED`.
- **D5** — Status register: `IE` = bit 0, `PRIV` = bit 1 (dormant until stage 3); no CSR
  file.
- **D6** — Zero-operand ops reserve their fields (must be zero), enforced in both `Encode`
  and `Decode`.
- **D7** — Interrupt context lives in CPU shadow registers (`intPC`/`intStatus`); IE
  clears on entry (no nesting); stray `rfi` faults.
- **D8** — Trap vector fixed at 0x0008.
- **D9** — Interrupt seam: delivery at the latch edge before the next FETCH; lines
  level-triggered; refused lines retry; at most one delivery per cycle; event `interrupt`,
  attributed to `board`, phase FETCH.
- **D10** — The interrupt controller is broker state until a second source exists.

## Devices
- **D11** — Timer MMIO at 0x4020 (8 bytes): `+0` interval (write arms/restarts, 0
  disarms), `+4` pending/ack; advances once per major on FETCH; always attached.
- **D12** — Generation stamped `AmmaBoard-2` via `machine.MachineVersion` (single source).

## Trace vocabulary (freeze point for golden fixtures)
- **D13** — Every ALU op emits a uniform `alu rX <= a OP b = c` EXECUTE line (registers
  don't latch in this machine; latch lines would lie).
- **D14** — PC is next-fetch, always; the halt line shows both addresses; branch/call/goto
  lines carry `at=`/`next=`; not-taken branches are logged; the state dump labels the
  convention.
- **D15** — Latch commits are one event per write, naming the target:
  `commit ram[0x0040] <= 0x…`.
- **D16** — `bus_write_post` keeps its name ("posted write" is the term of art; the primer
  closes the vocabulary gap).
- **D17** — `-dump-full` CLI flag for every-word RAM dumps.

## Content / tooling
- **D18** — `examples/timer.amm` demonstrates the day-8 handler pattern (publish, ack,
  disarm; main spins on the published value).
- **D19** — Labels are valid `add_number` immediates (absolute byte addresses) — the
  documented address-loading idiom (no `lui`).
- **D20** — TLB granularity: per-1KB page only, plus a "walk always" debug flag.
- **D21** — Payload header: 12-byte little-endian `origin, length, entry`.
- **D22** — Exercise editor: plain text input + step/run, no syntax frills.
- **D23** — Name: AmmaBoard (owner-settled).
- **D24** — trace.txt v2: per-cycle comment line + dashed rule + indented events; the
  events table and diffing are untouched.
- **D25** — Cycle comments use longhand terms (`register_6`, `program counter`); event
  rows stay terse.
- **D26** — Was/now visibility: every state change logs its old value at the source
  (`(was N)`); numbers show every interpretation when they diverge; no-op loads are facts
  too.
- **D27** — Blue Screen on any improper halt: plain-language cause per fault kind, the
  disassembled word, registers, memory window, last events; frozen as a `bluescreen`
  snapshot; fetch faults also log a `fault` event.

## Defect fixes + structure (2026-10-17)
- **D28** — Immediates are strictly 18-bit signed; `Encode` refuses wider; the assembler
  surfaces line-numbered errors instead of panicking.
- **D29** — Dropped writes fault at the latch, symmetrically with loads; `WriteSite{PC,
  Word}` names the posting instruction for the fault and the Blue Screen.
- **D30** — Interrupt lines are level-triggered end-to-end: the board re-samples the
  device at the delivery edge (`InterruptLine` contract); acked-before-delivery lines are
  dropped with an `irq_cleared` event.
- **D31** — (superseded by D34).
- **D32** — `.db` files carry `schema_version` (=1): migrate legacy files in place, refuse
  newer schemas on read.
- **D33** — Package-per-component layout is real (`machine/{cpu,ram,timer,echo,bluescreen}`);
  one file per instruction registered in `init()`.
- **D34** — Stop state is not a fault: `StopReason` ∈ {halt, fault, budget_exhausted,
  none} separate from `Fault`; `CleanHalt` vs `Halt`; budget = resumable pause; Blue
  Screen renders only on faults.
- **D35** — Default cycle budget 10,000 majors (`-max 0` = uncapped; negatives rejected):
  a runaway fails in ~1s instead of writing ~10⁶ rows.
- **D36** — `isa.Specs` is the single source of truth; the assembler is form-driven; the
  parity conformance suite fails the build on drift; halt's operand fields are now
  reserved like the other zero-operand ops.
- **D37 — TUI v1 shipped (bubbletea v2 + lipgloss v2, Go 1.25-pinned).** Stack:
  `charm.land/bubbletea/v2 v2.0.9` + `charm.land/lipgloss/v2 v2.0.6` + `charm.land/
  bubbles/v2 v2.2.1` (textinput); ultraviolet rides along transitively exactly as
  intended — never a direct require. v2.1.0/bubbletea and ultraviolet@latest need
  Go ≥1.26, so the pins are the newest Go-1.25 tags; bump them together when the
  toolchain moves. Scope (owner-confirmed): all three panes (disassembly with
  next-fetch marker / registers + pc-page memory hexdump / stats + event tail),
  shell `run · step [n] · reset · speed n · quit · help` mirrored by keys
  (space/s/+−/:/?/q/tab), program via `ammaboard tui [file.amm]` with the built-in
  counter as fallback, Blue Screen deferred (faults show in state line + log for
  now). New: `isa.Disassemble` (word → source, assembler-round-tripped by test);
  `buildBoard` takes an event sink so the TUI runs without SQLite; speed control
  = N majors per 33ms frame, machine semantics untouched. v2 key gotchas learned:
  space keypress is `KeySpace` (String "space", Text " "), not " ".
- **D38 — TUI layout contract (owner-requested, v1.1).** The frame is 1 status line
  + three equal-height panes + exactly 3 shell lines; pane equality is structural
  (every pane padded/clipped to the same row count — locked by test). Each pane =
  fixed header + scrollable body: machine keeps the CPU view fixed and scrolls
  memory; numbers keeps stats fixed and scrolls the tick log, grouped by major
  tick with one blank separator line between ticks. The tracked instruction is a
  white-on-dark-green highlighted line (not a pointer character), backing up to
  the halted/faulting instruction once stopped. Scrolling: `j/k` (or arrows) on
  the focused pane, `g/G` ends, `f` re-follows (pc / newest events); follow is the
  default and resumes on reset. Keyhints moved into the top bar (the shell owns
  the bottom 3 lines); the program name shows the basename, not the full path.
