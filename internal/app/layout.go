package app

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/youhide/hideTop/internal/ui"
)

// pageSize returns the number of process rows visible in the current viewport,
// used for PageUp/PageDown jumps.
// procViewport returns how many process rows are visible and the screen row
// where the first of them is drawn.
//
// View, pageSize and the mouse hit-test each used to compute this separately,
// and disagreed: View clamped the row count at 1 while the mouse handler
// clamped it at 3, so on a short terminal a click selected the wrong PID.
func (m Model) procViewport() (rows, dataY int) {
	_, h := m.screen()
	used := m.computeUsedLines()
	rows = max(1, h-used-ui.ProcessPanelChrome)
	// -1 because the help bar counted in used sits below the process panel.
	dataY = used - 1 + ui.ProcessPanelHeaderRows
	return rows, dataY
}

// pageSize is how far PageUp/PageDown jump: one screenful of process rows.
func (m Model) pageSize() int {
	rows, _ := m.procViewport()
	return rows
}

// computeUsedLines returns lines used by header + metrics + help bar.
// computeUsedLines is the number of screen lines consumed by everything that
// is not a process row: the header, the metrics section and the help bar.
func (m Model) computeUsedLines() int {
	return headerLines + m.metricsLayout().lines + helpLines
}

// columns returns the per-column widths and whether the two-column layout is
// in use. Single source of truth for the width breakpoint, which View,
// computeUsedLines and metricPanelAt each used to spell out separately.
func (m Model) columns() (colL, colR int, twoCol bool) {
	w, _ := m.screen()
	if w < twoColumnMinWidth {
		return w, w, false
	}
	colL = w / 2
	return colL, w - colL, true // right column absorbs the odd column
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
// Height budget. The metrics panels used to take whatever they wanted, which
// on a typical 33-row terminal left the process list — the main content of a
// top clone — with three rows. The two scrollable panels (temperature and
// network) now shrink toward their minimum so the process list keeps at least
// minProcRows, and grow into any leftover space on a tall terminal.
const (
	minProcRows = 8
	headerLines = 1
	helpLines   = 1

	// twoColumnMinWidth is the terminal width at which the metric panels split
	// into two columns.
	twoColumnMinWidth = 110

	// metricsHeightPercent caps the metrics section's share of the screen so
	// surplus height is split with the process list rather than all going to
	// the scrollable panels.
	metricsHeightPercent = 55
)

// ceilDiv divides a by b, rounding up.
func ceilDiv(a, b int) int { return (a + b - 1) / b }

// sectionHeight is the rendered height of a laid-out metrics section: the
// taller of the two columns.
func sectionHeight(panels []metricPanel) int {
	var lh, rh int
	for _, p := range panels {
		if p.col == 0 {
			lh += p.h
		} else {
			rh += p.h
		}
	}
	return max(lh, rh)
}

// metricsBudget is the tallest the metrics section may be.
//
// Two limits apply. The first guarantees the process list minProcRows on a
// short terminal. The second caps the panels at metricsHeightPercent of the
// screen so that extra height on a tall terminal is shared rather than being
// swallowed entirely by the scrollable panels growing to their maximum.
func (m Model) metricsBudget() int {
	_, h := m.screen()
	floor := h - headerLines - helpLines - ui.ProcessPanelChrome - minProcRows
	return max(0, min(floor, h*metricsHeightPercent/100))
}

// layoutKey identifies everything the metrics layout depends on. Comparable
// with ==, so a cache hit is one comparison.
type layoutKey struct {
	width      int
	height     int
	snapAt     time.Time // identity of the whole snapshot
	connsGen   uint32    // bumped on each connectionsMsg
	tempScroll int
	netScroll  int
	hidden     string // hidden panels, joined
}

// layoutCache memoises one metrics-section render.
//
// It hangs off Model by pointer so View, which has a value receiver and cannot
// mutate the model, can still fill it. Bubble Tea runs Update and View on one
// goroutine, so no synchronisation is needed.
//
// Without it the section was rendered two to four times per input event: once
// for View, once for computeUsedLines (via pageSize and the mouse handler) and
// once more for metricPanelAt, which hit-tests every mouse wheel notch by
// re-rendering all the panels to find their rectangles.
type layoutCache struct {
	key     layoutKey
	valid   bool
	panels  []metricPanel
	section string
	lines   int
	// Body rows the scrollable panels were actually rendered with. The scroll
	// clamps must use these, not the defaults: the panels size to the height
	// available, so clamping against a fixed count either stranded the last
	// sensors out of reach or let the user scroll past the end into nothing.
	tempRows int
	netRows  int
}

// currentLayoutKey captures the inputs of the current frame.
//
// Snapshot identity is CollectedAt. Any code that mutates m.snap in place,
// rather than replacing it from a snapshotMsg, must call invalidateLayout.
func (m Model) currentLayoutKey() layoutKey {
	w, h := m.screen()
	return layoutKey{
		width:      w,
		height:     h,
		snapAt:     m.snap.CollectedAt,
		connsGen:   m.connsGen,
		tempScroll: m.tempScroll,
		netScroll:  m.netScroll,
		hidden:     strings.Join(m.hiddenPanelList(), ","),
	}
}

// metricsLayout is the single entry point for the laid-out metrics section.
func (m Model) metricsLayout() *layoutCache {
	c := m.layout
	if c == nil {
		c = &layoutCache{} // zero-value Model: correct, just uncached
	}
	if k := m.currentLayoutKey(); c.valid && c.key == k {
		return c
	} else {
		c.key = k
	}

	colL, colR, twoCol := m.columns()
	c.panels, c.tempRows, c.netRows = m.layoutMetrics(colL, twoCol)
	c.section = joinMetricsSection(c.panels, colL, colR, twoCol)
	c.lines = strings.Count(c.section, "\n") + 1
	c.valid = true
	return c
}

// invalidateLayout drops the cached section. Call it after mutating m.snap in
// place rather than replacing it.
func (m Model) invalidateLayout() {
	if m.layout != nil {
		m.layout.valid = false
	}
}

// layoutMetrics renders the metric panels and assigns each to a column.
//
// It runs in two passes: the first renders the scrollable panels at their
// minimum height to learn the column assignments and what the fixed panels
// cost, the second grows temperature and network into whatever slack their own
// column has. Panel height is linear in body rows, so one growth step suffices.
//
// The second pass reuses the first pass's column assignment. Re-running the
// greedy packing on the grown panels would move them between columns and blow
// the budget the growth was computed against.
func (m Model) layoutMetrics(colL int, twoCol bool) (panels []metricPanel, tempRows, netRows int) {
	cw := colL // all panels render at the left-column width for a stable layout

	// Detail rows (per-core CPU bars, GPU engines) are dropped only when
	// keeping them would starve the process list, rather than at a fixed
	// height threshold: a tall terminal keeps the detail and still has room.
	panels = m.renderPanels(cw, twoCol, false, ui.TempMinRows, ui.NetMinRows, nil)
	budget := m.metricsBudget()
	compact := sectionHeight(panels) > budget
	if compact {
		panels = m.renderPanels(cw, twoCol, true, ui.TempMinRows, ui.NetMinRows, nil)
	}

	assign := make(map[string]int, len(panels))
	colHeight := map[int]int{}
	for _, p := range panels {
		assign[p.name] = p.col
		colHeight[p.col] += p.h
	}
	// Slack is consumed, not shared: temperature and network can land in the
	// same column (always, in single-column mode), and letting each grow into
	// the full remaining slack would overshoot the budget by that much.
	takeSlack := func(name string, want int) int {
		col, ok := assign[name]
		if !ok {
			return 0
		}
		got := max(0, min(want, budget-colHeight[col]))
		colHeight[col] += got
		return got
	}

	// Sensors render two per line when the panel is wide enough, so one line
	// of slack buys two more sensors.
	sensorsPerLine := 1
	if ui.TemperatureTwoColumn(cw) {
		sensorsPerLine = 2
	}
	tempGrow := takeSlack(string(panelTemp), ceilDiv(ui.TempMaxRows-ui.TempMinRows, sensorsPerLine))
	netGrow := takeSlack(string(panelNet), ui.NetMaxRows-ui.NetMinRows)

	tempRows = min(ui.TempMinRows+tempGrow*sensorsPerLine, ui.TempMaxRows)
	netRows = min(ui.NetMinRows+netGrow, ui.NetMaxRows)

	if tempRows == ui.TempMinRows && netRows == ui.NetMinRows {
		return panels, tempRows, netRows
	}
	return m.renderPanels(cw, twoCol, compact, tempRows, netRows, assign), tempRows, netRows
}

// tempScrollMax and netScrollMax bound the per-panel scroll offsets using the
// row counts the layout actually rendered with.
func (m Model) tempScrollMax() int {
	return ui.TemperatureScrollMaxRows(m.snap.Temperature, m.metricsLayout().tempRows)
}

func (m Model) netScrollMax() int {
	return ui.NetworkScrollMaxRows(m.netDelta, m.conns.Listening, m.metricsLayout().netRows)
}

// renderPanels renders every visible metric panel and assigns it to a column.
// When assign is nil the panels are packed greedily into the shorter column;
// otherwise the given assignment is reused verbatim.
//
// Panel heights are stable frame to frame (the network body is padded to a
// constant row count), so the greedy packing no longer makes panels jump
// between columns as traffic comes and goes.
func (m Model) renderPanels(cw int, twoCol, compact bool, tempRows, netRows int, assign map[string]int) []metricPanel {

	render := func(name panelName) string {
		if !m.panelVisible(name) {
			return ""
		}
		switch name {
		case panelCPU:
			return ui.RenderCPUCompact(m.snap.CPU, cw, m.cpuHistory, compact)
		case panelGPU:
			return ui.RenderGPUCompact(m.snap.GPU, cw, m.gpuHistory, compact)
		case panelMem:
			return ui.RenderMemory(m.snap.Memory, m.snap.Load, cw, m.memHistory)
		case panelTemp:
			s, _ := ui.RenderTemperatureScrollRows(m.snap.Temperature, cw, m.tempScroll, tempRows)
			return s
		case panelNet:
			s, _ := ui.RenderNetworkScrollRows(m.netDelta, m.conns.Listening, cw, m.netScroll, netRows)
			return s
		case panelDisk:
			return ui.RenderDisk(m.diskDelta, m.snap.Disk, cw)
		}
		return ""
	}

	var panels []metricPanel
	var lh, rh int
	for _, name := range panelOrder {
		body := render(name)
		if body == "" {
			continue
		}
		h := strings.Count(body, "\n") + 1
		col := 0
		switch {
		case !twoCol:
			col = 0
		case assign != nil:
			col = assign[string(name)]
		case rh < lh:
			col = 1
		}
		if col == 0 {
			lh += h
		} else {
			rh += h
		}
		panels = append(panels, metricPanel{name: string(name), s: body, h: h, col: col})
	}
	return panels
}

// buildMetricsSection renders all metric panels and joins them into a single
// string. Shared by View() and computeUsedLines() to avoid duplication.
// joinMetricsSection joins laid-out panels into the rendered section.
func joinMetricsSection(panels []metricPanel, colL, colR int, twoCol bool) string {
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
// metricRects maps each laid-out panel to the screen rows it occupies, for
// mouse hit-testing.
func (m Model) metricRects(twoCol bool) []panelRect {
	panels := m.metricsLayout().panels
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
// metricPanelAt returns the name of the metric panel under the given screen
// position, or empty when there is none.
func (m Model) metricPanelAt(x, y int) string {
	colL, _, twoCol := m.columns()
	col := 0
	if twoCol && x >= colL {
		col = 1
	}
	for _, r := range m.metricRects(twoCol) {
		if r.col == col && y >= r.y0 && y < r.y1 {
			return r.name
		}
	}
	return ""
}

// computeProcDataY returns the Y line where process data rows begin on screen.
// computeProcDataY is the screen row of the first process row.
func (m Model) computeProcDataY() int {
	_, dataY := m.procViewport()
	return dataY
}

// screen returns the terminal size, applying the 80x24 fallback used before
// the first WindowSizeMsg arrives.
func (m Model) screen() (w, h int) {
	w, h = m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}
	return w, h
}
