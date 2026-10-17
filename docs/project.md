# Project

## Vision
NandToTetris one level up: an interactive path from "what is a stored program" to "why is
my service 40× slower at 33M elements", with each invention introduced by the pain it
solved. History is necessity, not trivia.

## How it teaches
- **Inverted exercises:** the app ships fixed workloads; the learner builds or modifies
  the machine/kernel to make them behave. The machine is the artifact.
- Learner code is **toy assembly running on the machine itself** (page walk, cache policy,
  scheduler, kernel). No sandbox needed — worst case they crash their own TUI.
- **Everything is observable:** every run leaves a complete, diffable event log; faults are
  first-class data, not stack traces.
- Ends with a bridge to real systems (Linux source, perf, M-series), not a toy goodbye.

## Format
- Advent calendar: 24 doors, Dec 1–24, 16 chapters on a 2-play/1-rest cadence.
- Doors date-gated by default; `--free` mode for testers. Day 24 = finale/epilogue.
- Rest days carry curated reading — "dig deeper" is a designed feature, not filler.
- Audience: any dev with ≥2 years experience. Chapters never assume systems vocabulary.
- Each chapter ships a primer (pain → invention → mechanic, history ≤30%) and 2–4 curated
  links, plus a mandatory "what we simplified for focus" section.

## Name, license, launch
- Machine: **AmmaBoard-{generation}**, generation = ISA stage. CLI alias may shorten to `amma`.
- Day-12 campaign: "Look Amma, I made my own Blue Screen of Death."
- License: Apache-2.0 (repo, machine, chapters).
- Launch Dec 1 2026 = beta, doors 1–11 (through *your own MMU*); days 12+ render as
  labeled locked doors ("unlocks 2027"). Single binary via GitHub Releases.
- Leaderboard/shared features: separate app later, never on the critical path.

## Monetization (settled)
The 2026 launch is free forever and is the marketing asset. If ever monetized, in order:
paid Season 2, book of expanded primers, corporate perf workshops, sponsors. Real ROI is
reputation, not revenue.

## Non-goals
- Simulator performance. Determinism and correctness are the only invariants.
- Animation of any kind — numbers and grids only.
- Full computer history. Hard cap: 16 chapters, cut by "does this make you a better
  low-level engineer", not "is this famous".
- In-process interpreted learner Go (rejected: ceremony + sandbox risk for zero value).
