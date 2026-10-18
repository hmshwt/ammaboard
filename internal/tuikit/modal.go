package tuikit

import "charm.land/lipgloss/v2"

// Modal is transient content over the current screen: it captures the
// keyboard until esc pops it. Replacing the working context entirely is a
// FullBleed screen, not a modal.
type Modal interface {
	View(w, h int) string
}

// ModalStack is the esc-unwind chain: one press pops one level.
type ModalStack struct {
	stack []Modal
}

// Push opens a modal.
func (m *ModalStack) Push(v Modal) { m.stack = append(m.stack, v) }

// Pop closes the top modal; false when none is open.
func (m *ModalStack) Pop() bool {
	if len(m.stack) == 0 {
		return false
	}
	m.stack = m.stack[:len(m.stack)-1]
	return true
}

// Active reports an open modal.
func (m *ModalStack) Active() bool { return len(m.stack) > 0 }

// Top is the open modal, if any.
func (m *ModalStack) Top() (Modal, bool) {
	if len(m.stack) == 0 {
		return nil, false
	}
	return m.stack[len(m.stack)-1], true
}

// Center centers a rendered modal body in a w×h area with padX side padding.
func Center(body string, w, h, padX int) string {
	return lipgloss.NewStyle().Width(w).Height(h).
		AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Center).
		Padding(0, padX).Render(body)
}
