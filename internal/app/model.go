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

type Model struct {
	cfg      config.Config
	snap     metrics.Snapshot
	sortBy   metrics.SortField
	width    int
	height   int
	quitting bool
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

	header := ui.TitleStyle.Render("hideTop") +
		ui.SubtleStyle.Render(fmt.Sprintf("  refresh %s", m.cfg.RefreshInterval))

	cpuPanel := ui.RenderCPU(m.snap.CPU, w)
	memPanel := ui.RenderMemory(m.snap.Memory, m.snap.Load, w)

	// Count actual lines used by fixed panels
	usedLines := strings.Count(header, "\n") + 1
	usedLines += strings.Count(cpuPanel, "\n") + 1
	usedLines += strings.Count(memPanel, "\n") + 1
	usedLines += 1 // help bar

	// Render a dummy process panel with 0 rows to measure its overhead
	emptyProc := ui.RenderProcesses(nil, m.sortBy, w, 0)
	procOverhead := strings.Count(emptyProc, "\n") + 1

	procRows := h - usedLines - procOverhead
	if procRows < 3 {
		procRows = 3
	}
	procPanel := ui.RenderProcesses(m.snap.Processes, m.sortBy, w, procRows)
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
	case "-", "_":
		if m.cfg.RefreshInterval > 250*time.Millisecond {
			m.cfg.RefreshInterval -= 250 * time.Millisecond
		}
	}

	return m, nil
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
