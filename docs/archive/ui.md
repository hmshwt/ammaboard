# ui.md — TUI layout & interaction plan (DRAFT — to be revised)

Plan only; layout/keys/hierarchy all open to revision. Built with bubbletea + lipgloss + bubbles.
Derived from UX reference: three-column simulator (program | machine | stats) — simplified,
TUI-native, lazygit-style navigation.

---

## 1. Screen grammar: 3 panes + shell + two status bars

```
┌──────────────────────────────────────────────────────────┐
│ status bar (top): chapter · tick · mode · machine state  │
├───────────────────┬───────────────────┬──────────────────┤
│                   │                   │                  │
│  LEFT             │  CENTER           │  RIGHT           │
│  program          │  machine          │  numbers         │
│                   │                   │                  │
│                   │                   │                  │
├───────────────────┴───────────────────┴──────────────────┤
│ shell (4 lines): instruction/command prompt              │
├──────────────────────────────────────────────────────────┤
│ status bar (foot): keyhints · acceptance goal · context  │
└──────────────────────────────────────────────────────────┘
```

### Left pane — program
- Disassembly of loaded payload(s): address, hex, decoded mnemonic (toggle hex/asm).
- Editor mode for learner-written assembly (day 5+); tabs per program (nice/rogue).
- Chapter brief / primer text reachable here (or as a maximized-left view).

### Center pane — machine
- Registers grid, memory grid (hexdump; the whole ≤16–32KB is navigable), bus/status lines.
- **Timeline view** (cycles × named phases) — on-demand tab, central for days 8/16/17.
- Tick-log view (scrolled event stream; fault lines highlighted; enter = jump to tick).

### Right pane — numbers
- Live stats for the chapter (cycle count, hit rates, faults, wait states...) — derived from
  tick log, never hand-counted.
- **Acceptance goal block**: this chapter's success criteria, live checked, with pass/fail
  state — the challenge goal is one of these rows, highlighted.
- "What have we simplified for focus" + "vs reality" notes live at pane bottom.

## 2. Focus & maximize (lazygit-style)
- One pane has focus; keys are contextual to the focused pane.
- Maximize toggle: focused pane expands to full screen; same key toggles back (e.g. `m`,
  or `1/2/3` to focus left/center/right and `m` to maximize — exact keys TBD at implementation).
- Tab / shift-tab cycles focus. Maximized center = full-screen hexdump/timeline debugging.
- Chapter narrative: maximized-left doubles as reading mode.

## 3. Shell (bottom, 4 lines)
- Command line: `run`, `step [n]`, `load <payload>`, `diff`, `reset`, `export trace`,
  `seek <tick>`, `watch <reg|addr>`, `break <addr|kind>` etc.
- Keeps hands on keyboard; everything clickable/toggleable also has a shell command
  (shell = the scripting surface for the acceptance checker too).
- History via readline-ish bubbles; commands logged into the tick log as events.

## 4. Status bars
- **Top (context)**: chapter/day title, tick (major.minor), run state (paused/running/speed),
  active machine config (toggles: cache? mmu? base+limit?), acceptance summary chip.
- **Foot (keys)**: contextual keyhints for the focused pane + progress (e.g. "doors 3/8 ·
  goal: rogue faults, nice exits 0").

## 5. Help
- Global `?` = quick-help overlay: full keyboard map + shell command reference, always
  available, every screen. Help content generated from the same registry as the keybindings
  (no drift between what works and what's documented).

## 6. Progressive disclosure (per chapter)
- Chapters activate panes/features; day 4 shows registers + memory only; cache grid appears
  day 13; timeline day 8; fault panel day 10; by day 23 the screen approaches the full
  reference layout. The UI grows as the machine does — complexity is earned.
- Chapter config declares active panes/widgets (same data-driven chapter file).

## 7. Rendering rules
- Numbers and grids only — NO animation (per motherboard.md determinism/simplicity rules);
  changes highlight briefly, state is text.
- UI is a pure view over the tick log + current state (SQLite/storage engine is source of
  truth); every UI action = command → motherboard; UI never mutates machine state directly.
- Speed control (run N majors/frame) lives in the status bar; semantics never change.

## 8. (reserved)

## 9. The Blue Screen — freeze snapshot (session 10)

Homage to the BSOD, done properly. The original told you something happened, a percentage of
nothing, and that Windows was collecting info you'd never see. Ours confesses in plain English
with receipts.

**Trigger**: any fault (`limit_violation`, `page_fault`, `bus_conflict`), chapter `halt`, or
manual `freeze` command. Tick loop stops mid-phase (phased clock lets us stop *between* minors);
machine enters `frozen` state.

**Content** (one `snapshots` row): full registers, pc, status word, complete memory dump
(≤32KB — affordable by the sizing rule), pending bus value, fault record (type, vaddr, pc,
phase), plus a **cause chain**: the last N tick-log events and the disassembled instruction
at pc, highlighted.

**Presentation** (full-screen overlay, the maximize mechanic):
1. Plain-English cause: *"Your rogue program stored to address 0x0F28 at tick 411.2. That page
   belongs to the nice program. Here is the exact instruction, here is the register that pointed
   there, and here are the last 20 things the machine did."*
2. Registers + guilty instruction in disassembly.
3. Hexdump with the offending address highlighted.
Footer: **"0% complete: nothing is collecting. Everything is already here."**

**Keys (frozen state)**: `j/k` scroll dump · `t` jump into tick-log at the fault ·
`r` rewind to last checkpoint (event-sourcing replay — the fix for the BSOD's original sin:
on ours you can just go back) · `q` unfreeze.

**Learner customization (day 12+)**: the provided kernel fault routine renders the Blue Screen;
faulting code passes a **64-byte message** that the routine displays on it (the learner's first
printk — their words on the blue wall). Ambitious learners may **rewrite the kernel's BSOD
routine itself and render whatever they want (day 12 buffer-day exercise: "Look Amma, I made
my own Blue Screen of Death" — see PLANNING.md day 12).

## 10. Open questions
1. Exact keymap (lazygit conventions vs vim conventions) — decide with a real prototype.
2. Editor: bubbles textarea for assembly, or shell-invoked `$EDITOR`? (Lean: textarea for
   days 4–8, $EDITOR escape hatch for longer programs.)
3. Mouse support (bubbles has some) — probably ignore; keyboard-first product.
4. Timeline layout when phases per component differ (CPU vs bus) — depends on motherboard.md
   phase-set decision.
