package app

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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

// collectGracePeriod is how long past its own deadline a collection is given
// before the watchdog abandons it.
const collectGracePeriod = 2 * time.Second

type Model struct {
	// Configuration. Immutable for the session except RefreshInterval, which
	// the +/- keys adjust and which is persisted on quit.
	cfg             config.Config
	version         string
	intervalChanged bool // user adjusted the refresh interval this session
	panelsChanged   bool // user toggled panel visibility this session

	// Metric panels the user has hidden with the number keys, keyed by
	// panelName. Persisted to the config file on quit.
	hiddenPanels map[string]bool

	// Collected data and derived history.
	snap       metrics.Snapshot
	netDelta   metrics.NetworkDelta
	diskDelta  metrics.DiskDelta
	conns      metrics.NetConnections // listening ports, throttled cadence
	cpuHistory []float64              // sparkline history
	memHistory []float64
	gpuHistory []float64

	// Collection lifecycle. collecting is true while a snapshot command is in
	// flight; collectCancel bounds it and is cleared when the result arrives.
	collecting       bool
	collectStartedAt time.Time
	collectCancel    context.CancelFunc
	collectingConns  bool
	connsSampleAt    time.Time
	connsGen         uint32 // bumped on each connectionsMsg, part of the layout key
	paused           bool   // metrics auto-refresh paused by the user

	// Terminal geometry and the memoised metrics-section render. layout is a
	// pointer so View, which has a value receiver, can fill it.
	width  int
	height int
	layout *layoutCache

	// Process list view and selection.
	sortBy          metrics.SortField
	selectedPID     int32
	lastSelectedIdx int // last known visual index, used as a fallback
	treeView        bool
	hideSystem      bool
	searching       bool
	searchQuery     string

	// Per-panel scroll offsets (mouse wheel over the panel).
	tempScroll int
	netScroll  int

	// Overlays and transient UI state.
	showHelp      bool
	showNetwork   bool              // full-screen network/ports overlay
	showDetail    *ui.ProcessDetail // non-nil = showing detail overlay
	overlayScroll int               // scroll offset shared by whichever overlay is open
	refreshFlash  bool
	confirmKill   killSignal // non-zero = awaiting Y/N confirmation
	pidToKill     int32      // PID captured when the kill prompt was shown
	killMsg       string     // status message after kill attempt
	quitting      bool
	saveErr       error // config write failure, reported after the TUI exits
}

func New(cfg config.Config) Model {
	hidden := make(map[string]bool, len(cfg.HiddenPanels))
	for _, n := range cfg.HiddenPanels {
		hidden[n] = true
	}
	return Model{
		cfg:          cfg,
		sortBy:       metrics.SortByCPU,
		hiddenPanels: hidden,
		layout:       &layoutCache{},
	}
}

// SetVersion sets the version string for display in the help overlay.
// SaveError returns any config write failure from the session. main reports it
// after the alt-screen is torn down, where the user can actually see it.
func (m Model) SaveError() error {
	return m.saveErr
}

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
		// Watchdog: if a collection overruns its own deadline the result is
		// never coming (several gopsutil darwin calls ignore the context), and
		// collecting would otherwise stay true forever with the app silently
		// frozen. Drop it and let the next tick start a fresh one.
		if m.collecting && time.Since(m.collectStartedAt) > m.collectionTimeout()+collectGracePeriod {
			if m.collectCancel != nil {
				m.collectCancel()
				m.collectCancel = nil
			}
			m.collecting = false
		}
		if !m.collecting && !m.paused {
			ctx, cancel := context.WithTimeout(context.Background(), m.collectionTimeout())
			m.collectCancel = cancel
			m.collecting = true
			m.collectStartedAt = time.Now()
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
		if mx := m.tempScrollMax(); m.tempScroll > mx {
			m.tempScroll = mx
		}
		if mx := m.netScrollMax(); m.netScroll > mx {
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
			m.connsGen++
			if mx := m.netScrollMax(); m.netScroll > mx {
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
