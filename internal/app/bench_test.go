package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// BenchmarkWheelEvent measures one mouse wheel notch, which used to re-render
// every metric panel three times (hit-test, line count, then View) before the
// layout cache.
func BenchmarkWheelEvent(b *testing.B) {
	m := busyModel(130, 40)
	msg := tea.MouseMsg{X: 10, Y: 10, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown}
	b.ResetTimer()
	for b.Loop() {
		next, _ := m.handleMouse(msg)
		_ = next.(Model).View()
	}
}

// BenchmarkView measures a full frame render.
func BenchmarkView(b *testing.B) {
	m := busyModel(130, 40)
	_ = m.View() // warm the cache the way a running app would
	b.ResetTimer()
	for b.Loop() {
		_ = m.View()
	}
}

// BenchmarkArrowKey measures one selection move, which rebuilds the process
// display list (and in tree mode walks the whole forest) once per helper call.
func BenchmarkArrowKey(b *testing.B) {
	for _, tree := range []bool{false, true} {
		name := "flat"
		if tree {
			name = "tree"
		}
		b.Run(name, func(b *testing.B) {
			m := busyModel(130, 40)
			m.treeView = tree
			m.selectedPID = m.snap.Processes[0].PID
			b.ResetTimer()
			for b.Loop() {
				m.moveSelection(1)
			}
		})
	}
}
