package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics"
)

// ProcessViewState holds pure rendering state for the process panel.
type ProcessViewState struct {
	SortBy      metrics.SortField
	SelectedIdx int // -1 = no selection
	SearchQuery string
	Searching   bool
}

func columnHeader(label string, width int, align lipgloss.Position, sortBy, target metrics.SortField) string {
	indicator := ""
	if sortBy == target {
		switch target {
		case metrics.SortByPID:
			indicator = " ▲"
		default:
			indicator = " ▼"
		}
	}
	text := label + indicator
	style := lipgloss.NewStyle().Bold(true).Foreground(ColorHeader).Width(width).Align(align)
	if sortBy == target {
		style = style.Underline(true)
	}
	return style.Render(text)
}

func RenderProcesses(procs []metrics.ProcessInfo, state ProcessViewState, width, maxRows int) string {
	var b strings.Builder

	// Header with optional search indicator
	b.WriteString(HeaderStyle.Render("Processes"))
	if state.SearchQuery != "" || state.Searching {
		cursor := ""
		if state.Searching {
			cursor = "█"
		}
		b.WriteString(SubtleStyle.Render("  /" + state.SearchQuery + cursor))
	}
	b.WriteByte('\n')

	// Column headers with sort direction + underline on active column
	hdr := "  " +
		columnHeader("PID", 7, lipgloss.Left, state.SortBy, metrics.SortByPID) + " " +
		columnHeader("NAME", 24, lipgloss.Left, state.SortBy, metrics.SortField(-1)) + " " +
		columnHeader("CPU%", 8, lipgloss.Right, state.SortBy, metrics.SortByCPU) + " " +
		columnHeader("MEM%", 8, lipgloss.Right, state.SortBy, metrics.SortByMem)
	b.WriteString(hdr)
	b.WriteByte('\n')

	sep := SubtleStyle.Render(strings.Repeat("─", width-4))
	b.WriteString(sep)
	b.WriteByte('\n')

	// Compute visible window that keeps selection on screen
	n := len(procs)
	start := 0
	if maxRows > 0 && state.SelectedIdx >= maxRows {
		start = state.SelectedIdx - maxRows + 1
	}
	end := n
	if maxRows > 0 {
		end = start + maxRows
	}
	if end > n {
		end = n
		if maxRows > 0 {
			start = end - maxRows
			if start < 0 {
				start = 0
			}
		}
	}

	innerW := width - 4
	for i := start; i < end; i++ {
		p := procs[i]
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

		if state.SelectedIdx >= 0 && i == state.SelectedIdx {
			visible := lipgloss.Width(line)
			if visible < innerW {
				line += strings.Repeat(" ", innerW-visible)
			}
			line = lipgloss.NewStyle().
				Background(ColorSelectedBg).
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Render("▎" + line[1:])
		}

		b.WriteString(line)
		b.WriteByte('\n')
	}

	return PanelStyle.Width(width - 2).Render(b.String())
}
