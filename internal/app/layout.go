package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/youhide/hideTop/internal/ui"
)

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
