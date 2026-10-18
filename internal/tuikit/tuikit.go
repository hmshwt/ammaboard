// Package tuikit is the terminal kernel beneath the AmmaBoard TUI: a thin,
// boring layer over bubbletea and lipgloss that turns layout, screens, and
// key routing into declarative data.
//
// Contract (docs/tuikit.md):
//
//   - Compose, never wrap. The app root stays a plain tea.Model; raw
//     tea.Msg/tea.Cmd flow through untouched. The kit routes, never rewrites.
//   - Dependencies: bubbletea/v2, lipgloss/v2, stdlib. Never bubbles, never
//     internal/*, never ultraviolet.
//   - Immediate mode: regions are pure per-frame derivations of (machine
//     state + event tail + owned view state). No buffers, no dirty flags, no
//     revision counters — purity is the cache.
//   - State lives in components; policy lives in composites.
//
// Non-goals, fenced on purpose: mouse events (rects are exported for when a
// requirement appears), theming engine, widget zoo, animation/transitions,
// screen stacks, nested screens, nested focus rings, caching/dirty tracking,
// state management, and anything machine-aware.
package tuikit
