package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

type connectionsMsg metrics.NetConnections

// historySize is the max number of samples kept for sparklines.
const historySize = 60

// mouseWheelStep is how many process rows one wheel notch moves the selection.
const mouseWheelStep = 3

// connectionsSampleEvery throttles how often listening ports / connections are
// collected (they shell out to lsof on macOS, so this is deliberately slow).
const connectionsSampleEvery = 3 * time.Second

type Model struct {
	cfg           config.Config
	snap          metrics.Snapshot
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

	intervalChanged bool // user adjusted the refresh interval this session

	// Sparkline history
	cpuHistory []float64
	memHistory []float64
	gpuHistory []float64

	// Per-panel scroll offsets (mouse wheel over the panel)
	tempScroll int
	netScroll  int

	// Listening ports / connections (collected on a throttled cadence)
	conns           metrics.NetConnections
	collectingConns bool
	connsSampleAt   time.Time
	showNetwork     bool // full-screen network/ports overlay
	overlayScroll   int  // scroll offset shared by whichever overlay is open

	// UI state
	showHelp        bool
	showDetail      *ui.ProcessDetail // non-nil = showing detail overlay
	treeView        bool
	hideSystem      bool
	paused          bool       // metrics auto-refresh paused by the user
	confirmKill     killSignal // non-zero = awaiting Y/N confirmation
	pidToKill       int32      // PID captured when the kill prompt was shown
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
		if !m.collecting && !m.paused {
			ctx, cancel := context.WithTimeout(context.Background(), m.collectionTimeout())
			m.collectCancel = cancel
			m.collecting = true
			cmds = append(cmds, collectSnapshot(ctx, m.sortBy, m.snap, m.processSampleEvery(), m.cfg.ProcLimit, metrics.CollectOptions{
				SkipGPU:  m.cfg.NoGPU,
				SkipTemp: m.cfg.NoTemp,
			}))
		}
		if !m.cfg.NoPorts && !m.collectingConns && !m.paused &&
			(m.connsSampleAt.IsZero() || time.Since(m.connsSampleAt) >= connectionsSampleEvery) {
			m.collectingConns = true
			m.connsSampleAt = time.Now()
			cmds = append(cmds, collectConnections())
		}
		return m, tea.Batch(cmds...)

	case snapshotMsg:
		if m.collectCancel != nil {
			m.collectCancel()
			m.collectCancel = nil
		}
		m.collecting = false
		newSnap := metrics.Snapshot(msg)

		// Compute network and disk deltas against the last displayed snapshot.
		if m.snap.CollectedAt.IsZero() {
			m.netDelta = metrics.NetworkDelta{}
			m.diskDelta = metrics.DiskDelta{}
		} else {
			interval := newSnap.CollectedAt.Sub(m.snap.CollectedAt).Seconds()
			m.netDelta = metrics.ComputeNetworkDelta(newSnap.Network, m.snap.Network, interval)
			m.diskDelta = metrics.ComputeDiskDelta(newSnap.Disk, m.snap.Disk, interval)
		}

		m.snap = newSnap

		// Keep per-panel scroll offsets within bounds as content changes.
		if mx := ui.TemperatureScrollMax(m.snap.Temperature); m.tempScroll > mx {
			m.tempScroll = mx
		}
		if mx := ui.NetworkScrollMax(m.netDelta, m.conns.Listening); m.netScroll > mx {
			m.netScroll = mx
		}

		// Update selection tracking with new process list
		m.resolveSelection(m.filteredProcesses())

		// Record sparkline history
		m.cpuHistory = appendHistory(m.cpuHistory, newSnap.CPU.Total)
		m.memHistory = appendHistory(m.memHistory, newSnap.Memory.Percent)
		if newSnap.GPU != nil && newSnap.GPU.Available {
			m.gpuHistory = appendHistory(m.gpuHistory, newSnap.GPU.Utilization)
		}

		return m, nil

	case connectionsMsg:
		m.collectingConns = false
		if nc := metrics.NetConnections(msg); nc.Available {
			m.conns = nc
			if mx := ui.NetworkScrollMax(m.netDelta, m.conns.Listening); m.netScroll > mx {
				m.netScroll = mx
			}
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
			m.overlayScroll = 0
		} else {
			m.killMsg = msg.err.Error()
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return killMsgClearMsg{} })
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

// pageSize returns the number of process rows visible in the current viewport,
// used for PageUp/PageDown jumps.
func (m Model) pageSize() int {
	h := m.height
	if h == 0 {
		h = 24
	}
	emptyProc := ui.RenderProcesses(nil, ui.ProcessViewState{}, m.width, 0)
	procOverhead := strings.Count(emptyProc, "\n") + 1
	p := h - m.computeUsedLines() - procOverhead
	if p < 1 {
		p = 1
	}
	return p
}

// computeUsedLines returns lines used by header + metrics + help bar.
func (m Model) computeUsedLines() int {
	w := m.width
	if w == 0 {
		w = 80
	}

	twoCol := w >= 110
	var colL, colR int
	if twoCol {
		colL = w / 2
		colR = w - colL
	} else {
		colL = w
		colR = w
	}

	metricsSection := m.buildMetricsSection(colL, colR, twoCol)

	// Header is always 1 line
	usedLines := 1
	usedLines += strings.Count(metricsSection, "\n") + 1
	usedLines += 1 // help bar

	return usedLines
}

// metricPanel is a rendered metric panel with its assigned column and height.
type metricPanel struct {
	name string
	s    string
	h    int
	col  int // 0 = left column, 1 = right column (always 0 in single-column mode)
}

// panelRect is the on-screen rectangle of a metric panel, in view rows.
// Row 0 is the header line; metric panels start at row 1.
type panelRect struct {
	name   string
	col    int
	y0, y1 int // [y0, y1)
}

// layoutMetrics renders every metric panel and assigns it to a column. In
// two-column mode panels are packed greedily into the shorter column so the
// two columns stay balanced in height regardless of which panels are present
// (e.g. when there is no GPU). This is the single source of truth for panel
// placement, shared by rendering, line counting and mouse hit-testing.
func (m Model) layoutMetrics(colL int, twoCol bool) []metricPanel {
	cw := colL // all panels render at the left-column width for a stable layout

	tempPanel, _ := ui.RenderTemperatureScroll(m.snap.Temperature, cw, m.tempScroll)
	netPanel, _ := ui.RenderNetworkScroll(m.netDelta, m.conns.Listening, cw, m.netScroll)

	order := []struct {
		name string
		s    string
	}{
		{"cpu", ui.RenderCPU(m.snap.CPU, cw, m.cpuHistory)},
		{"gpu", ui.RenderGPU(m.snap.GPU, cw, m.gpuHistory)},
		{"mem", ui.RenderMemory(m.snap.Memory, m.snap.Load, cw, m.memHistory)},
		{"temp", tempPanel},
		{"net", netPanel},
		{"disk", diskPanel(m, cw)},
	}

	var panels []metricPanel
	var lh, rh int
	for _, c := range order {
		if c.s == "" {
			continue
		}
		h := strings.Count(c.s, "\n") + 1
		col := 0
		if twoCol && rh < lh {
			col = 1
		}
		if col == 0 {
			lh += h
		} else {
			rh += h
		}
		panels = append(panels, metricPanel{name: c.name, s: c.s, h: h, col: col})
	}
	return panels
}

// diskPanel is a tiny helper so the panel order table stays readable.
func diskPanel(m Model, cw int) string {
	return ui.RenderDisk(m.diskDelta, m.snap.Disk, cw)
}

// buildMetricsSection renders all metric panels and joins them into a single
// string. Shared by View() and computeUsedLines() to avoid duplication.
func (m Model) buildMetricsSection(colL, colR int, twoCol bool) string {
	panels := m.layoutMetrics(colL, twoCol)

	if !twoCol {
		rows := make([]string, 0, len(panels))
		for _, p := range panels {
			rows = append(rows, p.s)
		}
		return lipgloss.JoinVertical(lipgloss.Left, rows...)
	}

	var left, right []string
	for _, p := range panels {
		if p.col == 0 {
			left = append(left, p.s)
		} else {
			right = append(right, p.s)
		}
	}
	leftCol := lipgloss.NewStyle().Width(colL).Render(lipgloss.JoinVertical(lipgloss.Left, left...))
	rightCol := lipgloss.NewStyle().Width(colR).Render(lipgloss.JoinVertical(lipgloss.Left, right...))
	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
}

// metricRects returns the on-screen rectangle of each metric panel, used for
// mouse hit-testing. Left and right columns stack independently below the
// header row.
func (m Model) metricRects(colL int, twoCol bool) []panelRect {
	panels := m.layoutMetrics(colL, twoCol)
	const headerRows = 1
	ly, ry := headerRows, headerRows
	rects := make([]panelRect, 0, len(panels))
	for _, p := range panels {
		if twoCol && p.col == 1 {
			rects = append(rects, panelRect{name: p.name, col: 1, y0: ry, y1: ry + p.h})
			ry += p.h
		} else {
			rects = append(rects, panelRect{name: p.name, col: 0, y0: ly, y1: ly + p.h})
			ly += p.h
		}
	}
	return rects
}

// metricPanelAt returns the name of the metric panel under the given screen
// coordinates, or "" if none (e.g. the pointer is over the process list).
func (m Model) metricPanelAt(x, y int) string {
	w := m.width
	if w == 0 {
		w = 80
	}
	twoCol := w >= 110
	colL := w
	if twoCol {
		colL = w / 2
	}
	col := 0
	if twoCol && x >= colL {
		col = 1
	}
	for _, r := range m.metricRects(colL, twoCol) {
		if r.col == col && y >= r.y0 && y < r.y1 {
			return r.name
		}
	}
	return ""
}

// computeProcDataY returns the Y line where process data rows begin on screen.
func (m Model) computeProcDataY() int {
	// usedLines (header + metrics + help bar) + proc panel overhead:
	// border-top(1) + title row(1) + column header(1) + separator(1) = 4
	return m.computeUsedLines() - 1 + 4 // -1 because help bar is below processes
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

func collectSnapshot(ctx context.Context, sortBy metrics.SortField, previous metrics.Snapshot, processSampleEvery time.Duration, procLimit int, opts metrics.CollectOptions) tea.Cmd {
	return func() tea.Msg {
		snap := metrics.Collect(ctx, 200*time.Millisecond, sortBy, procLimit, processSampleEvery, previous, opts)
		return snapshotMsg(snap)
	}
}

// collectConnections gathers listening ports and active connections in the
// background. On failure it returns an unavailable result so the previous data
// is kept by the update loop.
func collectConnections() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		conns, err := metrics.CollectConnections(ctx)
		if err != nil {
			return connectionsMsg(metrics.NetConnections{})
		}
		return connectionsMsg(conns)
	}
}

func appendHistory(h []float64, v float64) []float64 {
	h = append(h, v)
	if len(h) > historySize {
		h = h[len(h)-historySize:]
	}
	return h
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

func (m Model) killSelectedProcess(sig killSignal) string {
	if m.pidToKill <= 0 {
		return ""
	}
	err := killProcess(int(m.pidToKill), sig)
	if err != nil {
		return fmt.Sprintf("kill %d: %v", m.pidToKill, err)
	}
	return fmt.Sprintf("sent signal %d to PID %d", sig, m.pidToKill)
}

func (m Model) exportSnapshot() string {
	basename := fmt.Sprintf("hideTop_%s.json", m.snap.CollectedAt.Format("20060102_150405"))

	// Destination: --export-dir / config export_dir, falling back to home.
	dir := strings.TrimSpace(m.cfg.ExportDir)
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = home
	} else if strings.HasPrefix(dir, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, strings.TrimPrefix(dir, "~"))
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Sprintf("export error: %v", err)
	}
	filename := filepath.Join(dir, basename)
	data, err := json.MarshalIndent(m.snap, "", "  ")
	if err != nil {
		return fmt.Sprintf("export error: %v", err)
	}
	if err := os.WriteFile(filename, data, 0o600); err != nil {
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
			return processDetailMsg{detail: detail, err: fmt.Errorf("process %d no longer available", pid)}
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
