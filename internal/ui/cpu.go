package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics"
)

// RenderCPU renders the CPU panel with per-core bars.
func RenderCPU(cpu metrics.CPUStats, width int, history []float64) string {
	return RenderCPUCompact(cpu, width, history, false)
}

// RenderCPUCompact renders the CPU panel, optionally hiding the per-core bars.
// On a short terminal the per-core rows are what stands between the process
// list and a usable number of rows, so the caller can drop them and keep the
// total plus the sparkline.
func RenderCPUCompact(cpu metrics.CPUStats, width int, history []float64, compact bool) string {
	var b strings.Builder
	innerW := contentWidth(width)

	head := HeaderStyle.Render("CPU")
	n := len(cpu.PerCore)
	if n > 0 {
		head += SubtleStyle.Render(fmt.Sprintf("  %d cores", n))
	}

	totalLabel := fmt.Sprintf("TOTAL %5.1f%%", cpu.Total)
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(renderBar(cpu.Total, totalLabel, innerW)))
	b.WriteByte('\n')

	switch {
	case compact:
		// Per-core bars omitted; TOTAL and the sparkline carry the signal.
	case innerW < 36:
		for i := 0; i < n; i++ {
			label := fmt.Sprintf("c%d %4.0f%%", i, cpu.PerCore[i])
			b.WriteString(renderBar(cpu.PerCore[i], label, innerW))
			if i < n-1 {
				b.WriteByte('\n')
			}
		}
	default:
		// Two-column layout: left = cpu0..cpu4, right = cpu5..cpu9
		half := (n + 1) / 2
		colWidth := (innerW - 2) / 2

		var leftCol, rightCol strings.Builder
		for i := 0; i < half; i++ {
			label := fmt.Sprintf("cpu%-2d %5.1f%%", i, cpu.PerCore[i])
			leftCol.WriteString(renderBar(cpu.PerCore[i], label, colWidth))
			if i < half-1 {
				leftCol.WriteByte('\n')
			}
		}
		for i := half; i < n; i++ {
			label := fmt.Sprintf("cpu%-2d %5.1f%%", i, cpu.PerCore[i])
			rightCol.WriteString(renderBar(cpu.PerCore[i], label, colWidth))
			if i < n-1 {
				rightCol.WriteByte('\n')
			}
		}

		cols := lipgloss.JoinHorizontal(lipgloss.Top,
			leftCol.String(), "  ", rightCol.String(),
		)
		b.WriteString(cols)
	}

	// Sparkline history
	if len(history) > 1 {
		b.WriteByte('\n')
		b.WriteString(RenderSparklineCompact("cpu", history, innerW))
	}

	return renderPanel(head, "", b.String(), width)
}

func renderBar(pct float64, label string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if maxWidth < 8 {
		return fitPlain(fmt.Sprintf("%.0f%%", pct), maxWidth)
	}

	const minBarWidth = 3
	label = fitPlain(label, maxWidth-minBarWidth-3)

	barWidth := maxWidth - lipgloss.Width(label) - 3
	if barWidth < 1 {
		barWidth = 1
	}

	filled := int(pct / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	color := BarColor(pct)
	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorBorder)

	return fmt.Sprintf("%s [%s%s]",
		label,
		filledStyle.Render(strings.Repeat("█", filled)),
		emptyStyle.Render(strings.Repeat("░", empty)),
	)
}
