package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/youhide/hideTop/internal/ui"
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	w := m.width
	if w == 0 {
		w = 80
	}
	h := m.height
	if h == 0 {
		h = 24
	}

	if m.showHelp {
		return ui.RenderHelpOverlayScroll(w, h, m.version, m.overlayScroll)
	}

	if m.showDetail != nil {
		return ui.RenderProcessDetailScroll(*m.showDetail, w, h, m.overlayScroll)
	}

	if m.showNetwork {
		return ui.RenderNetworkOverlay(m.conns, m.collectingConns, w, h, m.overlayScroll)
	}

	// Header
	batteryLabel := ui.RenderBattery(m.snap.Battery)
	refreshLabel := fmt.Sprintf("  refresh %s", m.cfg.RefreshInterval)
	if w < 40 {
		refreshLabel = fmt.Sprintf(" %s", m.cfg.RefreshInterval)
	}
	var header string
	if m.refreshFlash {
		header = ui.TitleStyle.Render("hideTop") +
			lipgloss.NewStyle().Bold(true).Foreground(ui.ColorTitle).Render(refreshLabel)
	} else {
		header = ui.TitleStyle.Render("hideTop") +
			ui.SubtleStyle.Render(refreshLabel)
	}
	appendHeader := func(part string) {
		if lipgloss.Width(header)+lipgloss.Width(part) <= w {
			header += part
		}
	}
	appendHeaderMessage := func(msg string) {
		remaining := w - lipgloss.Width(header) - 2
		if remaining <= 0 {
			return
		}
		header += "  " + lipgloss.NewStyle().Bold(true).Foreground(ui.ColorRed).Render(fitRunes(msg, remaining))
	}
	appendHeaderIfRoom := func(msg string, color lipgloss.Color) {
		part := "  " + lipgloss.NewStyle().Bold(true).Foreground(color).Render(msg)
		appendHeader(part)
	}
	if m.collecting {
		appendHeader(ui.SubtleStyle.Render("  collecting"))
	}
	if m.paused {
		appendHeaderIfRoom("paused", ui.ColorYellow)
	}
	if stale := m.snap.Status.StaleMetrics(); len(stale) > 0 {
		label := "stale:" + strings.Join(stale, ",")
		if w < 40 {
			label = "stale"
		}
		appendHeaderIfRoom(label, ui.ColorYellow)
	}
	if batteryLabel != "" {
		appendHeader("  " + batteryLabel)
	}
	if m.killMsg != "" {
		appendHeaderMessage(m.killMsg)
	}

	// Decide layout: two-column if wide enough
	twoCol := w >= 110
	var colL, colR int
	if twoCol {
		colL = w / 2
		colR = w - colL // handles odd widths
	} else {
		colL = w
		colR = w
	}

	// Render panels at appropriate widths
	metricsSection := m.buildMetricsSection(colL, colR, twoCol)

	// Filter processes and resolve PID-based selection.
	procs := m.filteredProcesses()
	selectedIdx := ui.DisplayIndexForPID(procs, m.treeView, m.selectedPID, m.lastSelectedIdx)

	procState := ui.ProcessViewState{
		SortBy:      m.sortBy,
		SelectedIdx: selectedIdx,
		SelectedPID: m.selectedPID,
		SearchQuery: m.searchQuery,
		Searching:   m.searching,
		TreeView:    m.treeView,
		HideSystem:  m.hideSystem,
		TotalProcs:  len(m.snap.Processes),
	}

	// Count lines used by fixed panels to size the process panel.
	usedLines := strings.Count(header, "\n") + 1
	usedLines += strings.Count(metricsSection, "\n") + 1
	usedLines += 1 // help bar

	emptyProc := ui.RenderProcesses(nil, procState, w, 0)
	procOverhead := strings.Count(emptyProc, "\n") + 1

	procRows := h - usedLines - procOverhead
	if procRows < 1 {
		procRows = 1
	}
	procPanel := ui.RenderProcesses(procs, procState, w, procRows)
	helpBar := ui.RenderHelp(w)

	view := lipgloss.JoinVertical(lipgloss.Left,
		header,
		metricsSection,
		procPanel,
		helpBar,
	)

	// Safety: never render taller than the terminal, otherwise the top status
	// bar gets scrolled off-screen in the alt-screen buffer. Keep the top rows
	// (header + metrics) so the status bar is always visible.
	if lipgloss.Height(view) > h {
		lines := strings.Split(view, "\n")
		view = strings.Join(lines[:h], "\n")
	}

	return view
}

func fitRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return string(r[:maxRunes])
	}
	return string(r[:maxRunes-3]) + "..."
}
