package tui

// styles.go — the palette. The kit's Styles own the region furniture (borders,
// titles, dim labels) via DefaultStyles; these are the host's own colors for
// the chrome, the cursor highlight, and verdict lines.

import "charm.land/lipgloss/v2"

var (
	cAccent  = lipgloss.Color("#4FA6FF")
	cDim     = lipgloss.Color("#6a737d")
	cRed     = lipgloss.Color("#f14c4c")
	cGreen   = lipgloss.Color("#3fb950")
	cCursor  = lipgloss.Color("#006400") // dark green
	cWhite   = lipgloss.Color("#ffffff")
	cBar     = lipgloss.Color("#1c2128")
	cBarText = lipgloss.Color("#e6edf3")

	stTop    = lipgloss.NewStyle().Background(cBar).Foreground(cBarText)
	stBottom = lipgloss.NewStyle().Foreground(cDim)
	stDim    = lipgloss.NewStyle().Foreground(cDim)
	stFault  = lipgloss.NewStyle().Foreground(cRed)
	stHalt   = lipgloss.NewStyle().Foreground(cGreen)
	stCursor = lipgloss.NewStyle().Foreground(cWhite).Background(cCursor)
	stHelp   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(cAccent).Padding(1, 2)
)
