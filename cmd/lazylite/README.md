# lazylite — the tuikit vocabulary, running

A lite lazygit whose entire job is to make the naming table concrete.
The data is a deterministic fixture (a fake repo snapshot — no git, no
network): what is demonstrated is the **architecture**, not version control.

```
ammaboard$ go run ./cmd/lazylite
```

> **Screens switch. Panes tile. Cursors point. Details stack. Modals capture.**

## Ten minutes, every term

| Term | Press | What you see |
|---|---|---|
| **Screen** (switch) | `1` `2` `3` | status → log → branches. Flat registry, atomic jump; the chrome's `screen <name>` changes. Stacked contexts don't survive a jump — they belong to a place. |
| **Grid** (rows × slots) | — | status = `Fixed(1) / Fill / Fixed(1)` rows; the fill row splits `Fixed(44) / Fill` into two slots. |
| **Chrome** | — | top row: status (always names screen · pane · cursor). bottom row: global key hints. Content the host furnishes; space the kit reserves. |
| **Pane** (tile) | `tab` | the focus ring cycles files → commits. The accent border follows; equal pane heights are structural. |
| **Cursor** (point) | `j` `k` `g` `G` | the highlighted line moves; the window follows it; the chrome names the target (`cursor internal/tuikit/frame.go (2/10) M`). The app always knows what you're pointing at. |
| **Detail** (stack) | `enter` or `+`, then `esc` | pushes the cursor's target onto the detail stack (file diff, commit body, branch info); `esc` pops exactly one level, back to the origin screen. Screens switch; details stack. |
| **Modal** (capture) | `?`, then `esc` | generated help that knows where you are: it merges global + screen + focused-pane keymaps and prints your current place. It cannot drift — it is a derivation, not a document. |
| **Palette** | `:` | every action available **here**, derived from the cursor's context; type to filter (the host grab); entries show their direct keys, so the palette teaches the keyboard. |
| **Action** | `space` `c` | the declared write path. Mutations reflect immediately (immediate mode — the next frame re-derives). |
| **Danger → confirm** | `d` / `r` / `x` | danger actions never run directly: they push a **confirm context** — a place whose keys route through the kit (`enter/y` confirm, `n` cancel, `esc` unwind), showing a rendered preview of the effect. |
| **Audit** | `go test ./cmd/lazylite` | `AssertCleanWiring` verifies geometry at declared minimums, key-scope discipline, region counts, chrome shape — before startup, as a test. |
| **No-cursor pane** | `3`, then `tab`, then `:` | the worktree pane owns no cursor: the palette degrades to the universal set only. Cursor-owning is optional, per pane. |

## The unwind rule

`esc` pops the innermost context, exactly one level:

```
confirm → palette → detail → leave screen → nothing
```

One global binding (`esc · back`) implements the whole contract, because
every context is a real place with a real stack entry — not a flag.

## Where each term lives in the code

| Term | Code |
|---|---|
| Screen | `screens.go` — six `NewScreen` declarations, flat registry |
| Grid / slots | `scr.Frame.Row(Fixed/Fill).Cols(...)` in each screen |
| Chrome | `chrome()` in `main.go` — two rows, same shape on every screen |
| Pane | `filesPane / commitsPane / logPane / branchesPane / worktreePane` in `panes.go` — pure per-frame `Region` values |
| Cursor | `cursorState` (index + kit `Scroll` follow) in `main.go`; `cursorKeys()` in `panes.go` |
| Detail stack | `details []detail` + `pushDetail`/`unwind` in `main.go`; `detailPane` in `panes.go` |
| Modal | `helpModal` in `help.go` — implements `tuikit.Modal` |
| Palette | `paletteState` + `actionsHere`/`filteredActions` in `panes.go`; typing grabbed in `handle()` (`main.go`) |
| Action / Danger | `action` type + `pushConfirm` and the `*File/*Commit/*Branch` mutations |
| Audit | `TestAuditClean` in `main_test.go` via `tuikit.AssertCleanWiring` |

## What the tests prove

Every term is exercised through the **real routing ladder** — keys are
driven headlessly through `App.Update`, so the ladder walk (region →
screen → global, first consumer wins) is the code under test, not a mock:

- the audit is clean (zero errors, zero warnings)
- the cursor moves, clamps, and follows the focus ring
- screens switch atomically; contexts belong to places
- the detail stack pushes and pops exactly one level
- danger actions confirm before mutating; esc cancels cleanly
- the palette derives contextually, filters from typed keys, and still
  routes danger actions through confirm
- the cursorless pane never offers cursor-scoped actions
- help is a true capture modal, toggled and popped per the contract

## What this demo is not

It is not the extraction, and it is not v0.1.0 of the standalone kit. The
cursor / detail / palette layer here is implemented **host-side**, exactly
as specced, so the vocabulary can be evaluated before it lands in the kit.
When the kit grows these primitives (with audit rules for cursor bounds and
danger-confirmation), lazylite's host-side machinery collapses into kit
calls — and that collapse is the point: everything you see hand-rolled in
`panes.go` is something a TUI app otherwise builds by hand.
