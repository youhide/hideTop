package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/youhide/hideTop/internal/config"
	"github.com/youhide/hideTop/internal/metrics"
	"github.com/youhide/hideTop/internal/ui"
)

type tickMsg time.Time

type snapshotMsg metrics.Snapshot

type flashDoneMsg struct{}

type Model struct {
	cfg          config.Config
	snap         metrics.Snapshot
	sortBy       metrics.SortField
	width        int
	height       int
	quitting     bool
	selectedPID  int32
	searching    bool
	searchQuery  string
	refreshFlash bool
}

func New(cfg config.Config) Model {
	return Model{
		cfg:    cfg,
		sortBy: metrics.SortByCPU,
	}
}

func (m Model) Init() tea.Cmd {
	return tick(m.cfg.RefreshInterval)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		return m, tea.Batch(
			collectSnapshot(m.sortBy),
			tick(m.cfg.RefreshInterval),
		)

	case snapshotMsg:
		m.snap = metrics.Snapshot(msg)
		return m, nil

	case flashDoneMsg:
		m.refreshFlash = false
		return m, nil
	}

	return m, nil
}

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

	// Header with refresh interval; briefly highlighted on change
	refreshLabel := fmt.Sprintf("  refresh %s", m.cfg.RefreshInterval)
	var header string
	if m.refreshFlash {
		header = ui.TitleStyle.Render("hideTop") +
			lipgloss.NewStyle().Bold(true).Foreground(ui.ColorTitle).Render(refreshLabel)
	} else {
		header = ui.TitleStyle.Render("hideTop") +
			ui.SubtleStyle.Render(refreshLabel)
	}

	cpuPanel := ui.RenderCPU(m.snap.CPU, w)
	memPanel := ui.RenderMemory(m.snap.Memory, m.snap.Load, w)

	// Filter processes and resolve PID-based selection
	procs := m.filteredProcesses()
	selectedIdx := m.findSelectionIndex(procs)

	procState := ui.ProcessViewState{
		SortBy:      m.sortBy,
		SelectedIdx: selectedIdx,
		SearchQuery: m.searchQuery,
		Searching:   m.searching,
	}

	// Count actual lines used by fixed panels
	usedLines := strings.Count(header, "\n") + 1
	usedLines += strings.Count(cpuPanel, "\n") + 1
	usedLines += strings.Count(memPanel, "\n") + 1
	usedLines += 1 // help bar

	// Render a dummy process panel with 0 rows to measure its overhead
	emptyProc := ui.RenderProcesses(nil, procState, w, 0)
	procOverhead := strings.Count(emptyProc, "\n") + 1

	procRows := h - usedLines - procOverhead
	if procRows < 3 {
		procRows = 3
	}
	procPanel := ui.RenderProcesses(procs, procState, w, procRows)
	helpBar := ui.RenderHelp(w)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		cpuPanel,
		memPanel,
		procPanel,
		helpBar,
	)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		return m.handleSearchKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "c":
		m.sortBy = metrics.SortByCPU
	case "m":
		m.sortBy = metrics.SortByMem
	case "p":
		m.sortBy = metrics.SortByPID
	case "+", "=":
		m.cfg.RefreshInterval += 250 * time.Millisecond
		m.refreshFlash = true
		return m, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return flashDoneMsg{} })
	case "-", "_":
		if m.cfg.RefreshInterval > 250*time.Millisecond {
			m.cfg.RefreshInterval -= 250 * time.Millisecond
		}
		m.refreshFlash = true
		return m, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return flashDoneMsg{} })
	case "j", "down":
		procs := m.filteredProcesses()
		if len(procs) > 0 {
			idx := m.findSelectionIndex(procs)
			if idx < 0 {
				m.selectedPID = procs[0].PID
			} else if idx < len(procs)-1 {
				m.selectedPID = procs[idx+1].PID
			}
		}
	case "k", "up":
		procs := m.filteredProcesses()
		if len(procs) > 0 {
			idx := m.findSelectionIndex(procs)
			if idx < 0 {
				m.selectedPID = procs[len(procs)-1].PID
			} else if idx > 0 {
				m.selectedPID = procs[idx-1].PID
			}
		}
	case "/":
		m.searching = true
	}

	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.searching = false
		m.searchQuery = ""
	case tea.KeyEnter:
		m.searching = false
	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
	case tea.KeyUp:
		procs := m.filteredProcesses()
		if len(procs) > 0 {
			idx := m.findSelectionIndex(procs)
			if idx < 0 {
				m.selectedPID = procs[len(procs)-1].PID
			} else if idx > 0 {
				m.selectedPID = procs[idx-1].PID
			}
		}
	case tea.KeyDown:
		procs := m.filteredProcesses()
		if len(procs) > 0 {
			idx := m.findSelectionIndex(procs)
			if idx < 0 {
				m.selectedPID = procs[0].PID
			} else if idx < len(procs)-1 {
				m.selectedPID = procs[idx+1].PID
			}
		}
	case tea.KeyRunes:
		m.searchQuery += string(msg.Runes)
	}
	return m, nil
}

// filteredProcesses returns processes matching the current search query.
func (m Model) filteredProcesses() []metrics.ProcessInfo {
	if m.searchQuery == "" {
		return m.snap.Processes
	}
	query := strings.ToLower(m.searchQuery)
	var result []metrics.ProcessInfo
	for _, p := range m.snap.Processes {
		if strings.Contains(strings.ToLower(p.Name), query) {
			result = append(result, p)
		}
	}
	return result
}

// findSelectionIndex resolves selectedPID to an index in the given slice.
// Returns -1 when no selection, 0 as fallback when PID disappears.
func (m Model) findSelectionIndex(procs []metrics.ProcessInfo) int {
	if m.selectedPID == 0 {
		return -1
	}
	for i, p := range procs {
		if p.PID == m.selectedPID {
			return i
		}
	}
	if len(procs) == 0 {
		return -1
	}
	return 0
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func collectSnapshot(sortBy metrics.SortField) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		snap := metrics.Collect(ctx, 200*time.Millisecond, sortBy, 50)
		return snapshotMsg(snap)
	}
}
