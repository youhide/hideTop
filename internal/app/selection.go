package app

import (
	"fmt"
	"strings"

	"github.com/youhide/hideTop/internal/metrics"
	"github.com/youhide/hideTop/internal/ui"
)

func (m *Model) moveSelection(delta int) {
	procs := m.filteredProcesses()
	if len(procs) == 0 {
		return
	}

	count := ui.ProcessDisplayCount(procs, m.treeView)
	if count == 0 {
		return
	}

	idx := ui.DisplayIndexForPID(procs, m.treeView, m.selectedPID, m.lastSelectedIdx)
	if idx < 0 {
		if delta < 0 {
			idx = count - 1
		} else {
			idx = 0
		}
	} else {
		idx += delta
		if idx < 0 {
			idx = 0
		}
		if idx >= count {
			idx = count - 1
		}
	}

	if pid, ok := ui.PIDAtDisplayIndex(procs, m.treeView, idx); ok {
		m.selectedPID = pid
		m.lastSelectedIdx = idx
	}
}

// moveToStart selects the first process in the display order.
func (m *Model) moveToStart() {
	procs := m.filteredProcesses()
	if pid, ok := ui.PIDAtDisplayIndex(procs, m.treeView, 0); ok {
		m.selectedPID = pid
		m.lastSelectedIdx = 0
	}
}

// moveToEnd selects the last process in the display order.
func (m *Model) moveToEnd() {
	procs := m.filteredProcesses()
	count := ui.ProcessDisplayCount(procs, m.treeView)
	if count == 0 {
		return
	}
	if pid, ok := ui.PIDAtDisplayIndex(procs, m.treeView, count-1); ok {
		m.selectedPID = pid
		m.lastSelectedIdx = count - 1
	}
}

// filteredProcesses returns processes matching the current search query and filters.
func (m Model) filteredProcesses() []metrics.ProcessInfo {
	procs := m.snap.Processes

	if m.hideSystem {
		hidden := make(map[string]bool, len(m.cfg.FilterUsers))
		for _, u := range m.cfg.FilterUsers {
			hidden[u] = true
		}
		var filtered []metrics.ProcessInfo
		for _, p := range procs {
			if p.User != "" && !hidden[p.User] {
				filtered = append(filtered, p)
			}
		}
		procs = filtered
	}

	if m.searchQuery == "" {
		return procs
	}
	query := strings.ToLower(m.searchQuery)
	var result []metrics.ProcessInfo
	for _, p := range procs {
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.User), query) ||
			strings.Contains(fmt.Sprintf("%d", p.PID), query) {
			result = append(result, p)
		}
	}
	return result
}

// resolveSelection updates m.selectedPID and m.lastSelectedIdx and returns
// the current index. Safe to call from Update handlers (value receiver + return).
func (m *Model) resolveSelection(procs []metrics.ProcessInfo) int {
	if m.selectedPID == 0 {
		m.lastSelectedIdx = -1
		return -1
	}

	idx := ui.DisplayIndexForPID(procs, m.treeView, m.selectedPID, m.lastSelectedIdx)
	pid, ok := ui.PIDAtDisplayIndex(procs, m.treeView, idx)
	if !ok {
		m.selectedPID = 0
		m.lastSelectedIdx = -1
		return -1
	}

	m.selectedPID = pid
	m.lastSelectedIdx = idx
	return idx
}
