I'll check the claims about Bubble Tea and existing alternatives first, then give you the critique.

## Verdict

The core idea is strong: one binding table drives routing, help, audit and receipts. But **don't freeze this draft**. It specifies the static wiring well and the runtime (async, input, mutation timing) poorly, and some sample APIs can't work in Go.

## 1. Raison d'être

| Issue | Fix |
|---|---|
| "Everyone reinvents this badly" has no evidence or prior-art comparison. [tuilib](https://pkg.go.dev/github.com/jsdrews/tuilib) has a shell and screen stack, [ninebox](https://pkg.go.dev/gitlab.com/atomic-blend/gu-cli/ninebox) (AGPL) feeds footer and help from one action table, and [ochazuke](https://pkg.go.dev/github.com/unixsuperhero/ochazuke@v0.49.0) treats bindings as data (no importers listed). | Lead with what I didn't see in their docs: **verifiable wiring** (audit, dry-run, receipts). Name the pains observed in lazylite and ammaboard. |
| The clean-slate exercise "converging" is not independent validation. | Require one new app, built by someone else, before the tag. |
| "Library, not framework", yet it owns Run, routing, layout and reserved keys. | Call it a small framework with escape hatches, and specify them. |
| Adoption rungs aren't independent: audit and tests need your screens and actions. | Ship the table, resolver and audit as a standalone package, or drop the claim. |

## 2. Blockers

1. **Cursor discovery fails.** An embedded `kit.Cursor` field shadows its own promoted `Cursor()` method. I tested it: the interface check returns false. The sample's `p.cursor` doesn't exist either. **Fix:** name the method `KitCursor()`.
2. **Constructors collide with types.** `Action` can't be both a type and a function in one package (confirmed); `Screen` and `Pane` have the same problem. **Fix:** keep the type names; use `NewScreen`, `Leaf`, `Do`.
3. **No async path.** Handlers show no `tea.Cmd` return, panes have no `Update` or `Init`, and `kit.Run` owns the model, so "your Update" is undefined. **Fix:** handlers return `tea.Cmd`; add optional `Init` and `Update(msg)`. Order: app, then every pane, including background ones.
4. **Danger actions can hit the wrong item.** The target resolves when the action "runs", after confirm. If data refreshes meanwhile, the cursor clamps to a neighbor. **Fix:** capture the target ID at keypress, show it in the confirm, recheck on yes, abort if gone. Do the same for palette entries.
5. **Text input is impossible.** `?` and `:` stay reserved during capture, so filters and commit messages can't contain them. **Fix:** a text-capture mode where only `esc` and `ctrl+c` are intercepted, with paste support and a way to host bubbles inputs.
6. **Render isn't pure.** `Track` mutates inside `Render`, and dispatch or dry-run may see stale IDs. A pure `Offset` also can't remember sticky scroll position. **Fix:** panes expose `ItemIDs()`; the kit syncs cursor and scroll after each update and resize. `Render` only reads.
7. **The minimum-size fold is unsound.** With `Ratio(1,10)` on width 100, a pane needing 30 gets 10, although the sum of minimums can pass. **Fix:** run the real layout solver at `Min` and check every leaf. Search upward for the honest minimum. Define rounding and `Fixed` overflow.
8. **`kit.Selected` contradicts "no package state".** A free function can't see the injected theme. **Fix:** put the theme in `Slot`, call `s.Selected(line)`, and delete `SetTheme`.

## 3. Architecture gaps

| Gap | Proposed fix |
|---|---|
| No API for opening modals, pushing details, focusing, quitting or showing errors | Typed methods on `Ctx`, plus a status notice |
| Drill-down has no parameters; a repeated screen shares cursors | Screen factory per push; detail screens get no digit |
| Pressing a digit mid-drill has odd stack behavior | `goto` resets the stack |
| Pane keys can shadow digits or quit; modals can trap `ctrl+c` | Pin `ctrl+c`; shadowing needs explicit `Override` |
| Tab focus is mentioned but missing from the catalog; blur has no way back stated | Add `FOCUS_NEXT/PREV`; drop blur |
| One key per action, yet defaults use `enter`/`y` | `Keys []string` |
| Mechanics are unnamed but overrides match by name | Separate `ID` from palette `Title` |
| No availability state, so help offers actions that can't run | Pure `Enabled(ctx)` returning a reason; show it in footer, palette and receipts |
| `Keys()` is "static" only by convention | Call once at wire-up and cache |
| Audit needs unique pane names but says no uniqueness checks | Check screen and pane names |
| `Slot` size vs borders, headers and multi-line rows is unclear | `Slot` is the content area; panes declare body height and row heights |
| Overflowing output breaks layout | Clip and pad by cell width; record a violation |
| Empty item list flips `Scroll` into line mode; short content may count as pinned | Explicit mode field; pin only after tail intent |
| Duplicate or removed item IDs are undefined | Report duplicates; keep last index for nearest-row recovery |
| `Receipt` has no schema; `Debugf` concurrency unstated | Fixed fields and outcome enum; thread-safe buffer |
| Escape hatch is asserted, not designed | Expose `Init/Update/View` and a raw-key pane interface |
| Bubble Tea version unstated | Target [v2](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md): new view and key types, `ctrl+u` spelling (not `ctrl-u`); pass background-color messages through |

## 4. Scope and document

- **Type budget is already broken:** `Scroll`, `Item`, `Handler`, `Keyer`, `Notifier` and constraint types aren't counted. Judge concepts a day-one user must learn instead.
- **Cut or defer:** positional group IDs (use optional labels and tree paths), the debug overlay (dump receipts to a log first), and the 12-type target.
- **Refuse explicitly or design:** multi-select, key chords, pane zoom or collapse on small terminals.
- **Separate documents:** spec, migration delta and decision log are mixed.
- **Add:** panic recovery with terminal restore, blocking-handler rules, a versioning policy.

## Suggested order

1. Compile a day-one example plus an async refresh, a filter input and a drill-down.
2. Fix the runtime contract: messages, sync phase, target safety, input capture.
3. Router, receipts, audit, `kittest`.
4. Footer, palette, theme.
5. Tag after the outside app works.

The Go checks used 1.24.7; the prior-art comparison rests on search excerpts and a partial read of ochazuke's page, not a code review. I didn't see your `internal/tuikit` code.