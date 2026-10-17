# TUI

Stack: `bubbletea/v2` + `lipgloss/v2` + `bubbles/v2` (ultraviolet arrives transitively
via the Charm stack — never a direct dependency). The UI is the only layer issuing Cmds.

**Status: v1.1 shipped** — three equal-height panes filling the frame (1 status line
above, 3 shell lines below), each pane = fixed header + scrollable body: program
(disassembly), machine (CPU view fixed, memory scrollable), numbers (stats fixed,
tick log scrollable, one blank line between major ticks). Scrolling: `j/k` or arrows
on the focused pane, `g/G` top/bottom, `f` re-follows (pc / newest events; auto-follow
while running). The tracked instruction is white on dark green — a highlighted line,
not a pointer. `ammaboard tui [file.amm]` (no file = built-in counter demo). Deferred:
Blue Screen overlay (faults currently show in the state line + red log lines; the CLI
renders the full panel), hex/asm toggle, timeline view, editor, mouse.

## Layout
```
┌ status bar: chapter · tick · mode · machine state ─────────┐
│ LEFT program     │ CENTER machine      │ RIGHT numbers     │
├ shell (4 lines): command prompt ───────────────────────────┤
└ status bar: keyhints · acceptance goal · progress ─────────┘
```
- **Left — program:** disassembly (toggle hex/asm), assembly editor from day 5, tabs per
  program (nice/rogue), chapter brief/primer (maximized = reading mode).
- **Center — machine:** registers, memory hexdump (all ≤32KB navigable), timeline view
  (cycles × phases — central days 8/16/17), tick-log stream with faults highlighted.
- **Right — numbers:** live stats derived from the log, never hand-counted (cycles, hit
  rates, faults, wait states); the acceptance goal block, live checked; "simplified for
  focus" + "vs reality" notes.

## Mechanics
- One focused pane; tab / shift-tab cycles; maximize toggle (lazygit-style) — *v1: focus
  border only, maximize pending*. Keys are contextual to the focused pane.
- Shell (v1: `run`, `step [n]`, `reset`, `speed n`, `quit`, `help`; keys space/s/+−
  mirror them). Later: `load <payload>`, `diff`, `export trace`, `seek <tick>`,
  `watch <reg|addr>`, `break <addr|kind>` — the acceptance checker scripts the same
  surface.
- `?` = help overlay generated from the keybinding registry — docs cannot drift from
  behavior.
- Progressive disclosure: chapters activate panes (day 4: registers + memory; timeline
  day 8; fault panel day 10; cache grid day 13). The UI grows as the machine does.
- Rendering rules: numbers and grids, never animation; state changes highlight briefly.
  The UI is a pure view over the tick log + state; speed control = N cycles per frame,
  semantics never change.

## Blue Screen (freeze snapshot)
Trigger: any fault (plus manual `freeze`). The tick loop stops mid-phase; the machine
freezes.

- **Content** (one `snapshots` row): full registers, pc, status, complete memory (≤32KB —
  affordable by the sizing rule), pending bus value, the fault record, and a cause chain:
  the last N events plus the disassembled instruction at pc, highlighted.
- **Presentation** (full-screen overlay): plain-English cause first — "Your rogue program
  stored to 0x0F28 at tick 411.2. That page belongs to the nice program. Here is the
  exact instruction, the register that pointed there, and the last 20 things the machine
  did." Then registers + guilty instruction, then hexdump with the address marked.
  Footer: **"0% complete: nothing is collecting. Everything is already here."**
- **Keys:** `j/k` scroll · `t` jump into the tick log at the fault · `r` rewind (replay —
  the fix for the original BSOD's sin) · `q` unfreeze.
- **Day 12+:** the learner's kernel fault routine renders the screen; faulting code passes
  a 64-byte message (their first printk). Rewriting the BSOD routine itself is the
  day-12 exercise.

## Open
Keymap (vim vs lazygit conventions) — decide with the prototype. Editor: bubbles textarea
for days 4–8, `$EDITOR` escape hatch for long programs. Mouse: ignored, keyboard-first.
Timeline layout when per-component phase sets differ.
