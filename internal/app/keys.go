package app

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/youhide/hideTop/internal/config"
	"github.com/youhide/hideTop/internal/metrics"
	"github.com/youhide/hideTop/internal/ui"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle kill confirmation first
	if m.confirmKill != 0 {
		switch msg.String() {
		case "y", "Y":
			m.killMsg = m.killSelectedProcess(m.confirmKill)
			m.confirmKill = 0
			m.pidToKill = 0
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return killMsgClearMsg{} })
		default:
			m.confirmKill = 0
			m.pidToKill = 0
			m.killMsg = ""
		}
		return m, nil
	}

	// Any full-screen overlay (help / detail / network) is scrollable and
	// shares one handler for a consistent feel.
	if m.showHelp || m.showDetail != nil || m.showNetwork {
		return m.handleOverlayKey(msg)
	}

	if m.searching {
		return m.handleSearchKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		if m.collectCancel != nil {
			m.collectCancel()
			m.collectCancel = nil
		}
		if m.intervalChanged {
			if err := config.SaveInterval(m.cfg.RefreshInterval); err != nil {
				fmt.Fprintf(os.Stderr, "hideTop: failed to save refresh interval: %v\n", err)
			}
		}
		return m, tea.Quit
	case "c":
		if m.sortBy != metrics.SortByCPU {
			m.sortBy = metrics.SortByCPU
			m.treeView = false
			m.resolveSelection(m.filteredProcesses())
		}
	case "m":
		if m.sortBy != metrics.SortByMem {
			m.sortBy = metrics.SortByMem
			m.treeView = false
			m.resolveSelection(m.filteredProcesses())
		}
	case "p":
		if m.sortBy != metrics.SortByPID {
			m.sortBy = metrics.SortByPID
			m.treeView = false
			m.resolveSelection(m.filteredProcesses())
		}
	case "+", "=":
		m.cfg.RefreshInterval += 250 * time.Millisecond
		m.intervalChanged = true
		m.refreshFlash = true
		return m, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return flashDoneMsg{} })
	case "-", "_":
		if m.cfg.RefreshInterval > 250*time.Millisecond {
			m.cfg.RefreshInterval -= 250 * time.Millisecond
			m.intervalChanged = true
		}
		m.refreshFlash = true
		return m, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return flashDoneMsg{} })
	case "j", "down":
		m.moveSelection(1)
	case "k", "up":
		m.moveSelection(-1)
	case "pgdown", "ctrl+f":
		m.moveSelection(m.pageSize())
	case "pgup", "ctrl+b":
		m.moveSelection(-m.pageSize())
	case "home", "g":
		m.moveToStart()
	case "end", "G":
		m.moveToEnd()
	case " ":
		m.paused = !m.paused
	case "z":
		m.tempScroll = 0
		m.netScroll = 0
	case "n":
		m.showNetwork = true
		m.overlayScroll = 0
	case "/":
		m.searching = true
	case "?":
		m.showHelp = true
		m.overlayScroll = 0
	case "t":
		m.treeView = !m.treeView
		m.resolveSelection(m.filteredProcesses())
	case "s":
		m.hideSystem = !m.hideSystem
		m.resolveSelection(m.filteredProcesses())
	case "K":
		if m.selectedPID > 0 {
			m.confirmKill = signalKill
			m.pidToKill = m.selectedPID
			m.killMsg = fmt.Sprintf("SIGKILL PID %d? (y/N)", m.selectedPID)
		}
	case "x":
		if m.selectedPID > 0 {
			m.confirmKill = signalTerm
			m.pidToKill = m.selectedPID
			m.killMsg = fmt.Sprintf("Kill PID %d? (y/N)", m.selectedPID)
		}
	case "e":
		msg := m.exportSnapshot()
		m.killMsg = msg // reuse the status area
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return killMsgClearMsg{} })
	case "enter":
		if m.selectedPID > 0 {
			return m, fetchProcessDetail(m.selectedPID, m.snap.Processes)
		}
	}

	return m, nil
}

// handleOverlayKey drives scrolling and closing for whichever full-screen
// overlay is currently open (help, process detail, or network/ports).
func (m Model) handleOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showHelp, m.showNetwork, m.showDetail = false, false, nil
		m.overlayScroll = 0
		return m, nil
	case "q":
		// Close the overlay rather than quitting the app.
		m.showHelp, m.showNetwork, m.showDetail = false, false, nil
		m.overlayScroll = 0
		return m, nil
	case "?":
		if m.showHelp {
			m.showHelp = false
			m.overlayScroll = 0
			return m, nil
		}
	case "n":
		if m.showNetwork {
			m.showNetwork = false
			m.overlayScroll = 0
			return m, nil
		}
	case "enter":
		if m.showDetail != nil {
			m.showDetail = nil
			m.overlayScroll = 0
			return m, nil
		}
	}

	switch msg.String() {
	case "j", "down":
		m.overlayScroll++
	case "k", "up":
		if m.overlayScroll > 0 {
			m.overlayScroll--
		}
	case "pgdown", "ctrl+f":
		m.overlayScroll += 10
	case "pgup", "ctrl+b":
		m.overlayScroll -= 10
		if m.overlayScroll < 0 {
			m.overlayScroll = 0
		}
	case "g", "home":
		m.overlayScroll = 0
	case "end", "G":
		m.overlayScroll = m.overlayMaxScroll()
	}
	if mx := m.overlayMaxScroll(); m.overlayScroll > mx {
		m.overlayScroll = mx
	}
	return m, nil
}

// overlayMaxScroll returns the max scroll offset for the active overlay.
func (m Model) overlayMaxScroll() int {
	switch {
	case m.showHelp:
		return ui.HelpOverlayMaxScroll(m.width, m.height)
	case m.showNetwork:
		return ui.NetworkOverlayMaxScroll(m.conns, m.width, m.height)
	case m.showDetail != nil:
		return ui.ProcessDetailMaxScroll(*m.showDetail, m.width, m.height)
	}
	return 0
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		if m.collectCancel != nil {
			m.collectCancel()
			m.collectCancel = nil
		}
		return m, tea.Quit
	case tea.KeyEscape:
		m.searching = false
		m.searchQuery = ""
		m.resolveSelection(m.filteredProcesses())
	case tea.KeyEnter:
		m.searching = false
		// After confirming search, open detail if a process is selected
		m.resolveSelection(m.filteredProcesses())
		if m.selectedPID > 0 {
			return m, fetchProcessDetail(m.selectedPID, m.snap.Processes)
		}
	case tea.KeyBackspace:
		r := []rune(m.searchQuery)
		if len(r) > 0 {
			m.searchQuery = string(r[:len(r)-1])
			m.resolveSelection(m.filteredProcesses())
		}
	case tea.KeyUp:
		m.moveSelection(-1)
	case tea.KeyDown:
		m.moveSelection(1)
	case tea.KeyRunes:
		m.searchQuery += string(msg.Runes)
		m.resolveSelection(m.filteredProcesses())
	}
	return m, nil
}
