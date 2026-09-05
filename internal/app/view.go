package app

import (
	"fmt"
	"strings"
	"time"

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

	// A destructive confirmation gets a real modal. As a header chip it could
	// be truncated away entirely on a narrow terminal, leaving the app waiting
	// for a keypress that sends a signal with nothing on screen to say so.
	if m.confirmKill != 0 {
		return ui.RenderConfirm(m.confirmTitle(), m.confirmBody(), "y confirm    any other key cancel", w, h)
	}

	header := m.renderHeader(w)

	metricsSection := m.metricsLayout().section

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

	procRows, _ := m.procViewport()
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

// collectSlowThreshold is how long a collection must be in flight before the
// header advertises it. Collections normally take 200-400ms (CollectCPU alone
// samples for 200ms), so showing the label immediately made it blink on every
// tick and shifted every chip to its right. Above this threshold the label
// stays put, which is when it actually tells the user something.
const collectSlowThreshold = 600 * time.Millisecond

// formatInterval renders the refresh interval at a stable width so that
// changing it with +/- does not shift the chips that follow.
func formatInterval(d time.Duration) string {
	return fmt.Sprintf("%-6s", d.String())
}

// renderHeader builds the top status line. Every segment is width-stable and
// appended only if it fits, so the line neither wraps nor reflows between
// frames.
func (m Model) renderHeader(w int) string {
	title := ui.TitleStyle.Render(fitRunes("hideTop", w))

	refreshLabel := "  refresh " + formatInterval(m.cfg.RefreshInterval)
	if w < 40 {
		refreshLabel = " " + formatInterval(m.cfg.RefreshInterval)
	}
	style := ui.SubtleStyle
	if m.refreshFlash {
		style = ui.HeaderStyle
	}
	header := title
	if lipgloss.Width(header)+lipgloss.Width(refreshLabel) <= w {
		header += style.Render(refreshLabel)
	}

	appendHeader := func(part string) {
		if lipgloss.Width(header)+lipgloss.Width(part) <= w {
			header += part
		}
	}
	appendHeaderIfRoom := func(msg string, color lipgloss.Color) {
		appendHeader("  " + lipgloss.NewStyle().Bold(true).Foreground(color).Render(msg))
	}

	if m.collectionIsSlow() {
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
	if batteryLabel := ui.RenderBattery(m.snap.Battery); batteryLabel != "" {
		appendHeader("  " + batteryLabel)
	}
	if m.killMsg != "" {
		if remaining := w - lipgloss.Width(header) - 2; remaining > 0 {
			header += "  " + lipgloss.NewStyle().Bold(true).Foreground(ui.ColorRed).
				Render(fitRunes(m.killMsg, remaining))
		}
	}
	return header
}

// collectionIsSlow reports whether the in-flight collection has been running
// long enough to be worth telling the user about.
func (m Model) collectionIsSlow() bool {
	return m.collecting && !m.collectStartedAt.IsZero() &&
		time.Since(m.collectStartedAt) > collectSlowThreshold
}
