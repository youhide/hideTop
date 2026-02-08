package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics"
)

func SortIndicator(current, target metrics.SortField) string {
	if current == target {
		return " ▼"
	}
	return ""
}

func RenderProcesses(procs []metrics.ProcessInfo, sortBy metrics.SortField, width, maxRows int) string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Processes"))
	b.WriteByte('\n')

	hdr := fmt.Sprintf("  %-7s %-24s %8s %8s",
		"PID",
		"NAME",
		"CPU%"+SortIndicator(sortBy, metrics.SortByCPU),
		"MEM%"+SortIndicator(sortBy, metrics.SortByMem),
	)
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHeader).Render(hdr))
	b.WriteByte('\n')

	sep := SubtleStyle.Render(strings.Repeat("─", width-4))
	b.WriteString(sep)
	b.WriteByte('\n')

	rows := procs
	if maxRows > 0 && maxRows < len(rows) {
		rows = rows[:maxRows]
	}

	for _, p := range rows {
		name := p.Name
		if len(name) > 24 {
			name = name[:21] + "..."
		}

		cpuColor := BarColor(p.CPUPercent)
		memColor := BarColor(float64(p.MemPercent))

		line := fmt.Sprintf("  %-7d %-24s %s %s",
			p.PID,
			name,
			lipgloss.NewStyle().Foreground(cpuColor).Width(8).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f", p.CPUPercent)),
			lipgloss.NewStyle().Foreground(memColor).Width(8).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f", p.MemPercent)),
		)
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return PanelStyle.Width(width).Render(b.String())
}
