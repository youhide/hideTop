package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/youhide/hideTop/internal/ui"
)

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.showHelp || m.showDetail != nil || m.showNetwork {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.overlayScroll -= mouseWheelStep
			if m.overlayScroll < 0 {
				m.overlayScroll = 0
			}
		case tea.MouseButtonWheelDown:
			m.overlayScroll += mouseWheelStep
			if mx := m.overlayMaxScroll(); m.overlayScroll > mx {
				m.overlayScroll = mx
			}
		}
		return m, nil
	}
	if m.confirmKill != 0 {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.handlePanelScroll(msg, -1) {
			return m, nil
		}
		m.moveSelection(-mouseWheelStep)
	case tea.MouseButtonWheelDown:
		if m.handlePanelScroll(msg, 1) {
			return m, nil
		}
		m.moveSelection(mouseWheelStep)
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionRelease {
			return m, nil
		}
		// Click on a sortable column header row toggles that sort.
		if msg.Y == m.computeProcDataY()-2 {
			if sf, ok := ui.ProcessSortAtX(m.width, msg.X); ok {
				if m.sortBy != sf {
					m.sortBy = sf
					m.treeView = false
					m.resolveSelection(m.filteredProcesses())
				}
				return m, nil
			}
		}
		procs := m.filteredProcesses()
		if len(procs) == 0 {
			return m, nil
		}

		// Compute where process data rows start on screen.
		// We need: header lines + metrics lines + proc panel overhead
		// (border top + title row + column header row + separator row).
		procDataY := m.computeProcDataY()

		// Compute viewport start (same logic as RenderProcesses)
		selectedIdx := ui.DisplayIndexForPID(procs, m.treeView, m.selectedPID, m.lastSelectedIdx)
		maxRows, _ := m.procViewport()
		viewStart := 0
		if maxRows > 0 && selectedIdx >= maxRows {
			viewStart = selectedIdx - maxRows + 1
		}

		clickedIdx := viewStart + (msg.Y - procDataY)
		if pid, ok := ui.PIDAtDisplayIndex(procs, m.treeView, clickedIdx); ok {
			m.selectedPID = pid
			m.lastSelectedIdx = clickedIdx
		}
	}
	return m, nil
}

// handlePanelScroll scrolls a metric panel under the mouse pointer. It returns
// true when the pointer is over a scrollable panel (temperature or network) and
// the wheel event was consumed there instead of the process list. dir is -1 for
// wheel-up and +1 for wheel-down.
func (m *Model) handlePanelScroll(msg tea.MouseMsg, dir int) bool {
	switch m.metricPanelAt(msg.X, msg.Y) {
	case "temp":
		v := m.tempScroll + dir
		if mx := ui.TemperatureScrollMax(m.snap.Temperature); v > mx {
			v = mx
		}
		if v < 0 {
			v = 0
		}
		m.tempScroll = v
		return true
	case "net":
		v := m.netScroll + dir
		if mx := ui.NetworkScrollMax(m.netDelta, m.conns.Listening); v > mx {
			v = mx
		}
		if v < 0 {
			v = 0
		}
		m.netScroll = v
		return true
	}
	return false
}
