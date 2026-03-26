package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/youhide/hideTop/internal/config"
	"github.com/youhide/hideTop/internal/metrics"
	"github.com/youhide/hideTop/internal/ui"
)

type tickMsg time.Time

type snapshotMsg metrics.Snapshot

type flashDoneMsg struct{}

type killMsgClearMsg struct{}

type processDetailMsg struct {
	detail ui.ProcessDetail
	err    error
}

// historySize is the max number of samples kept for sparklines.
const historySize = 60

type Model struct {
	cfg           config.Config
	snap          metrics.Snapshot
	prevSnap      metrics.Snapshot
	netDelta      metrics.NetworkDelta
	diskDelta     metrics.DiskDelta
	sortBy        metrics.SortField
	width         int
	height        int
	quitting      bool
	selectedPID   int32
	searching     bool
	searchQuery   string
	refreshFlash  bool
	collecting    bool
	collectCancel context.CancelFunc

	// Sparkline history
	cpuHistory []float64
	memHistory []float64
	gpuHistory []float64

	// UI state
	showHelp        bool
	showDetail      *ui.ProcessDetail // non-nil = showing detail overlay
	treeView        bool
	hideSystem      bool
	confirmKill     killSignal // non-zero = awaiting Y/N confirmation
	killMsg         string     // status message after kill attempt
	lastSelectedIdx int        // last known visual index for fallback
	version         string
}

func New(cfg config.Config) Model {
	return Model{
		cfg:    cfg,
		sortBy: metrics.SortByCPU,
	}
}

// SetVersion sets the version string for display in the help overlay.
func (m *Model) SetVersion(v string) {
	m.version = v
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

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tickMsg:
		cmds := []tea.Cmd{tick(m.cfg.RefreshInterval)}
		if !m.collecting {
			ctx, cancel := context.WithTimeout(context.Background(), m.collectionTimeout())
			m.collectCancel = cancel
			m.collecting = true
			cmds = append(cmds, collectSnapshot(ctx, m.sortBy, m.snap, m.processSampleEvery(), metrics.CollectOptions{
				SkipGPU:  m.cfg.NoGPU,
				SkipTemp: m.cfg.NoTemp,
			}))
		}
		return m, tea.Batch(cmds...)

	case snapshotMsg:
		if m.collectCancel != nil {
			m.collectCancel()
			m.collectCancel = nil
		}
		m.collecting = false
		newSnap := metrics.Snapshot(msg)

		// Compute network and disk deltas
		if m.prevSnap.CollectedAt.IsZero() {
			m.netDelta = metrics.NetworkDelta{}
			m.diskDelta = metrics.DiskDelta{}
		} else {
			interval := newSnap.CollectedAt.Sub(m.prevSnap.CollectedAt).Seconds()
			m.netDelta = metrics.ComputeNetworkDelta(newSnap.Network, m.prevSnap.Network, interval)
			m.diskDelta = metrics.ComputeDiskDelta(newSnap.Disk, m.prevSnap.Disk, interval)
		}

		m.prevSnap = m.snap
		m.snap = newSnap

		// Update selection tracking with new process list
		m.resolveSelection(m.filteredProcesses())

		// Record sparkline history
		m.cpuHistory = appendHistory(m.cpuHistory, newSnap.CPU.Total)
		m.memHistory = appendHistory(m.memHistory, newSnap.Memory.Percent)
		if newSnap.GPU != nil && newSnap.GPU.Available {
			m.gpuHistory = appendHistory(m.gpuHistory, newSnap.GPU.Utilization)
		}

		return m, nil

	case flashDoneMsg:
		m.refreshFlash = false
		return m, nil

	case killMsgClearMsg:
		m.killMsg = ""
		return m, nil

	case processDetailMsg:
		if msg.err == nil {
			m.showDetail = &msg.detail
		}
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

	if m.showHelp {
		return ui.RenderHelpOverlay(w, h, m.version)
	}

	if m.showDetail != nil {
		return ui.RenderProcessDetail(*m.showDetail, w, h)
	}

	// Header
	batteryLabel := ui.RenderBattery(m.snap.Battery)
	refreshLabel := fmt.Sprintf("  refresh %s", m.cfg.RefreshInterval)
	var header string
	if m.refreshFlash {
		header = ui.TitleStyle.Render("hideTop") +
			lipgloss.NewStyle().Bold(true).Foreground(ui.ColorTitle).Render(refreshLabel)
	} else {
		header = ui.TitleStyle.Render("hideTop") +
			ui.SubtleStyle.Render(refreshLabel)
	}
	if m.collecting {
		header += ui.SubtleStyle.Render("  collecting")
	}
	if stale := m.snap.Status.StaleMetrics(); len(stale) > 0 {
		header += "  " + lipgloss.NewStyle().Bold(true).Foreground(ui.ColorYellow).Render("stale:"+strings.Join(stale, ","))
	}
	if batteryLabel != "" {
		header += "  " + batteryLabel
	}
	if m.killMsg != "" {
		header += "  " + lipgloss.NewStyle().Bold(true).Foreground(ui.ColorRed).Render(m.killMsg)
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
	cpuPanel := ui.RenderCPU(m.snap.CPU, colL, m.cpuHistory)
	gpuPanel := ui.RenderGPU(m.snap.GPU, colR, m.gpuHistory)
	tempPanel := ui.RenderTemperature(m.snap.Temperature, colR)
	netPanel := ui.RenderNetwork(m.netDelta, colL)
	diskPanel := ui.RenderDisk(m.diskDelta, m.snap.Disk, colR)

	// Memory panel width depends on whether GPU is present (left vs right column)
	var memPanel string
	if twoCol && gpuPanel != "" {
		memPanel = ui.RenderMemory(m.snap.Memory, m.snap.Load, colL, m.memHistory)
	} else if twoCol {
		// No GPU: memory goes to the right of CPU
		memPanel = ui.RenderMemory(m.snap.Memory, m.snap.Load, colR, m.memHistory)
	} else {
		memPanel = ui.RenderMemory(m.snap.Memory, m.snap.Load, colL, m.memHistory)
	}

	// Filter processes and resolve PID-based selection.
	procs := m.filteredProcesses()
	selectedIdx, _ := findSelectionIndex(m.selectedPID, procs, m.lastSelectedIdx)

	procState := ui.ProcessViewState{
		SortBy:      m.sortBy,
		SelectedIdx: selectedIdx,
		SearchQuery: m.searchQuery,
		Searching:   m.searching,
		TreeView:    m.treeView,
		HideSystem:  m.hideSystem,
		TotalProcs:  len(m.snap.Processes),
	}

	// Build the metric panels section
	var metricRows []string

	if twoCol {
		// Two-column layout: pair panels side by side with matched heights
		// Row 1: CPU | GPU (or Memory if no GPU)
		if gpuPanel != "" {
			metricRows = append(metricRows, joinPanelRow(cpuPanel, gpuPanel, colL, colR))
		} else {
			metricRows = append(metricRows, joinPanelRow(cpuPanel, memPanel, colL, colR))
		}

		// Row 2: Memory | Temperature (only if GPU existed in row 1)
		if gpuPanel != "" {
			if tempPanel != "" {
				metricRows = append(metricRows, joinPanelRow(memPanel, tempPanel, colL, colR))
			} else {
				metricRows = append(metricRows, memPanel)
			}
		} else if tempPanel != "" {
			metricRows = append(metricRows, tempPanel)
		}

		// Row 3: Network | Disk
		if netPanel != "" && diskPanel != "" {
			metricRows = append(metricRows, joinPanelRow(netPanel, diskPanel, colL, colR))
		} else if netPanel != "" {
			metricRows = append(metricRows, netPanel)
		} else if diskPanel != "" {
			metricRows = append(metricRows, diskPanel)
		}
	} else {
		// Single-column stacked layout
		metricRows = append(metricRows, cpuPanel)
		if gpuPanel != "" {
			metricRows = append(metricRows, gpuPanel)
		}
		metricRows = append(metricRows, memPanel)
		if tempPanel != "" {
			metricRows = append(metricRows, tempPanel)
		}
		if netPanel != "" {
			metricRows = append(metricRows, netPanel)
		}
		if diskPanel != "" {
			metricRows = append(metricRows, diskPanel)
		}
	}

	metricsSection := lipgloss.JoinVertical(lipgloss.Left, metricRows...)

	// Count lines used by fixed panels to size the process panel.
	usedLines := strings.Count(header, "\n") + 1
	usedLines += strings.Count(metricsSection, "\n") + 1
	usedLines += 1 // help bar

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
		metricsSection,
		procPanel,
		helpBar,
	)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle kill confirmation first
	if m.confirmKill != 0 {
		switch msg.String() {
		case "y", "Y":
			m.killMsg = m.killSelectedProcess(m.confirmKill)
			m.confirmKill = 0
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return killMsgClearMsg{} })
		default:
			m.confirmKill = 0
			m.killMsg = ""
		}
		return m, nil
	}

	// Close detail overlay on Esc or Enter
	if m.showDetail != nil {
		switch msg.String() {
		case "esc", "enter", "q":
			m.showDetail = nil
		}
		return m, nil
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
		return m, tea.Quit
	case "c":
		if m.sortBy != metrics.SortByCPU {
			m.sortBy = metrics.SortByCPU
			m.treeView = false
		}
	case "m":
		if m.sortBy != metrics.SortByMem {
			m.sortBy = metrics.SortByMem
			m.treeView = false
		}
	case "p":
		if m.sortBy != metrics.SortByPID {
			m.sortBy = metrics.SortByPID
			m.treeView = false
		}
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
			idx := m.resolveSelection(procs)
			if idx < 0 {
				m.selectedPID = procs[0].PID
			} else if idx < len(procs)-1 {
				m.selectedPID = procs[idx+1].PID
			}
		}
	case "k", "up":
		procs := m.filteredProcesses()
		if len(procs) > 0 {
			idx := m.resolveSelection(procs)
			if idx < 0 {
				m.selectedPID = procs[len(procs)-1].PID
			} else if idx > 0 {
				m.selectedPID = procs[idx-1].PID
			}
		}
	case "/":
		m.searching = true
	case "?":
		m.showHelp = !m.showHelp
	case "t":
		m.treeView = !m.treeView
	case "s":
		m.hideSystem = !m.hideSystem
	case "K":
		if m.selectedPID > 0 {
			m.confirmKill = signalKill
			m.killMsg = fmt.Sprintf("SIGKILL PID %d? (y/N)", m.selectedPID)
		}
	case "x":
		if m.selectedPID > 0 {
			m.confirmKill = signalTerm
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

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.showHelp || m.showDetail != nil || m.confirmKill != 0 {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		procs := m.filteredProcesses()
		if len(procs) > 0 {
			idx := m.resolveSelection(procs)
			if idx <= 0 {
				m.selectedPID = procs[0].PID
			} else {
				m.selectedPID = procs[idx-1].PID
			}
		}
	case tea.MouseButtonWheelDown:
		procs := m.filteredProcesses()
		if len(procs) > 0 {
			idx := m.resolveSelection(procs)
			if idx < 0 {
				m.selectedPID = procs[0].PID
			} else if idx < len(procs)-1 {
				m.selectedPID = procs[idx+1].PID
			}
		}
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionRelease {
			return m, nil
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
		selectedIdx, _ := findSelectionIndex(m.selectedPID, procs, m.lastSelectedIdx)
		h := m.height
		if h == 0 {
			h = 24
		}
		usedLines := m.computeUsedLines()
		emptyProc := ui.RenderProcesses(nil, ui.ProcessViewState{}, m.width, 0)
		procOverhead := strings.Count(emptyProc, "\n") + 1
		maxRows := h - usedLines - procOverhead
		if maxRows < 3 {
			maxRows = 3
		}
		viewStart := 0
		if maxRows > 0 && selectedIdx >= maxRows {
			viewStart = selectedIdx - maxRows + 1
		}

		clickedIdx := viewStart + (msg.Y - procDataY)
		if clickedIdx >= 0 && clickedIdx < len(procs) {
			m.selectedPID = procs[clickedIdx].PID
		}
	}
	return m, nil
}

// computeUsedLines returns lines used by header + metrics + help bar.
func (m Model) computeUsedLines() int {
	w := m.width
	if w == 0 {
		w = 80
	}

	// Header is always 1 line
	usedLines := 1

	twoCol := w >= 110
	var colL, colR int
	if twoCol {
		colL = w / 2
		colR = w - colL
	} else {
		colL = w
		colR = w
	}

	cpuPanel := ui.RenderCPU(m.snap.CPU, colL, m.cpuHistory)
	gpuPanel := ui.RenderGPU(m.snap.GPU, colR, m.gpuHistory)
	tempPanel := ui.RenderTemperature(m.snap.Temperature, colR)
	netPanel := ui.RenderNetwork(m.netDelta, colL)
	diskPanel := ui.RenderDisk(m.diskDelta, m.snap.Disk, colR)

	var memPanel string
	if twoCol && gpuPanel != "" {
		memPanel = ui.RenderMemory(m.snap.Memory, m.snap.Load, colL, m.memHistory)
	} else if twoCol {
		memPanel = ui.RenderMemory(m.snap.Memory, m.snap.Load, colR, m.memHistory)
	} else {
		memPanel = ui.RenderMemory(m.snap.Memory, m.snap.Load, colL, m.memHistory)
	}

	var metricRows []string
	if twoCol {
		if gpuPanel != "" {
			metricRows = append(metricRows, joinPanelRow(cpuPanel, gpuPanel, colL, colR))
		} else {
			metricRows = append(metricRows, joinPanelRow(cpuPanel, memPanel, colL, colR))
		}
		if gpuPanel != "" {
			if tempPanel != "" {
				metricRows = append(metricRows, joinPanelRow(memPanel, tempPanel, colL, colR))
			} else {
				metricRows = append(metricRows, memPanel)
			}
		} else if tempPanel != "" {
			metricRows = append(metricRows, tempPanel)
		}
		if netPanel != "" && diskPanel != "" {
			metricRows = append(metricRows, joinPanelRow(netPanel, diskPanel, colL, colR))
		} else if netPanel != "" {
			metricRows = append(metricRows, netPanel)
		} else if diskPanel != "" {
			metricRows = append(metricRows, diskPanel)
		}
	} else {
		metricRows = append(metricRows, cpuPanel)
		if gpuPanel != "" {
			metricRows = append(metricRows, gpuPanel)
		}
		metricRows = append(metricRows, memPanel)
		if tempPanel != "" {
			metricRows = append(metricRows, tempPanel)
		}
		if netPanel != "" {
			metricRows = append(metricRows, netPanel)
		}
		if diskPanel != "" {
			metricRows = append(metricRows, diskPanel)
		}
	}

	metricsSection := lipgloss.JoinVertical(lipgloss.Left, metricRows...)
	usedLines += strings.Count(metricsSection, "\n") + 1
	usedLines += 1 // help bar

	return usedLines
}

// computeProcDataY returns the Y line where process data rows begin on screen.
func (m Model) computeProcDataY() int {
	// usedLines (header + metrics + help bar) + proc panel overhead:
	// border-top(1) + title row(1) + column header(1) + separator(1) = 4
	return m.computeUsedLines() - 1 + 4 // -1 because help bar is below processes
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.searching = false
		m.searchQuery = ""
	case tea.KeyEnter:
		m.searching = false
		// After confirming search, open detail if a process is selected
		if m.selectedPID > 0 {
			return m, fetchProcessDetail(m.selectedPID, m.snap.Processes)
		}
	case tea.KeyBackspace:
		r := []rune(m.searchQuery)
		if len(r) > 0 {
			m.searchQuery = string(r[:len(r)-1])
		}
	case tea.KeyUp:
		procs := m.filteredProcesses()
		if len(procs) > 0 {
			idx := m.resolveSelection(procs)
			if idx < 0 {
				m.selectedPID = procs[len(procs)-1].PID
			} else if idx > 0 {
				m.selectedPID = procs[idx-1].PID
			}
		}
	case tea.KeyDown:
		procs := m.filteredProcesses()
		if len(procs) > 0 {
			idx := m.resolveSelection(procs)
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

// findSelectionIndex resolves selectedPID to an index in the given slice.
// Returns -1 when no selection. When PID disappears, stays at same visual
// position (clamped to list bounds) rather than jumping to 0.
func findSelectionIndex(selectedPID int32, procs []metrics.ProcessInfo, lastIdx int) (int, int32) {
	if selectedPID == 0 {
		return -1, 0
	}
	for i, p := range procs {
		if p.PID == selectedPID {
			return i, selectedPID
		}
	}
	if len(procs) == 0 {
		return -1, 0
	}
	// PID disappeared — pick closest position
	idx := lastIdx
	if idx >= len(procs) {
		idx = len(procs) - 1
	}
	if idx < 0 {
		idx = 0
	}
	return idx, procs[idx].PID
}

// resolveSelection updates m.selectedPID and m.lastSelectedIdx and returns
// the current index. Safe to call from Update handlers (value receiver + return).
func (m *Model) resolveSelection(procs []metrics.ProcessInfo) int {
	idx, pid := findSelectionIndex(m.selectedPID, procs, m.lastSelectedIdx)
	m.selectedPID = pid
	m.lastSelectedIdx = idx
	return idx
}

func (m Model) collectionTimeout() time.Duration {
	timeout := m.cfg.RefreshInterval * 2
	if timeout < time.Second {
		timeout = time.Second
	}
	if timeout > 5*time.Second {
		timeout = 5 * time.Second
	}
	return timeout
}

func (m Model) processSampleEvery() time.Duration {
	sampleEvery := 2 * time.Second
	if m.cfg.RefreshInterval > sampleEvery {
		return m.cfg.RefreshInterval
	}
	return sampleEvery
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func collectSnapshot(ctx context.Context, sortBy metrics.SortField, previous metrics.Snapshot, processSampleEvery time.Duration, opts metrics.CollectOptions) tea.Cmd {
	return func() tea.Msg {
		snap := metrics.Collect(ctx, 200*time.Millisecond, sortBy, 50, processSampleEvery, previous, opts)
		return snapshotMsg(snap)
	}
}

// joinPanelRow places two panels side by side with matched heights.
// Each panel is placed in a fixed-width container so columns stay aligned
// regardless of content width.
func joinPanelRow(left, right string, leftW, rightW int) string {
	lh := strings.Count(left, "\n") + 1
	rh := strings.Count(right, "\n") + 1
	h := lh
	if rh > h {
		h = rh
	}
	l := lipgloss.NewStyle().Width(leftW).Height(h).Render(left)
	r := lipgloss.NewStyle().Width(rightW).Height(h).Render(right)
	return lipgloss.JoinHorizontal(lipgloss.Top, l, r)
}

func appendHistory(h []float64, v float64) []float64 {
	h = append(h, v)
	if len(h) > historySize {
		h = h[len(h)-historySize:]
	}
	return h
}

func (m Model) killSelectedProcess(sig killSignal) string {
	if m.selectedPID <= 0 {
		return ""
	}
	err := killProcess(int(m.selectedPID), sig)
	if err != nil {
		return fmt.Sprintf("kill %d: %v", m.selectedPID, err)
	}
	return fmt.Sprintf("sent signal %d to PID %d", sig, m.selectedPID)
}

func (m Model) exportSnapshot() string {
	filename := fmt.Sprintf("hideTop_%s.json", m.snap.CollectedAt.Format("20060102_150405"))
	data, err := json.MarshalIndent(m.snap, "", "  ")
	if err != nil {
		return fmt.Sprintf("export error: %v", err)
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return fmt.Sprintf("export error: %v", err)
	}
	return fmt.Sprintf("exported to %s", filename)
}

func fetchProcessDetail(pid int32, procs []metrics.ProcessInfo) tea.Cmd {
	return func() tea.Msg {
		// Find base info from snapshot
		var base metrics.ProcessInfo
		for _, p := range procs {
			if p.PID == pid {
				base = p
				break
			}
		}

		detail := ui.ProcessDetail{ProcessInfo: base}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		proc, err := process.NewProcessWithContext(ctx, pid)
		if err != nil {
			return processDetailMsg{detail: detail, err: nil}
		}

		if cmd, err := proc.CmdlineWithContext(ctx); err == nil {
			detail.Cmdline = cmd
		}
		if fds, err := proc.NumFDsWithContext(ctx); err == nil {
			detail.NumFDs = fds
		}
		if mem, err := proc.MemoryInfoWithContext(ctx); err == nil && mem != nil {
			detail.RSS = mem.RSS
			detail.VMS = mem.VMS
		}
		if ct, err := proc.CreateTimeWithContext(ctx); err == nil {
			detail.CreateTime = ct
		}

		return processDetailMsg{detail: detail}
	}
}
