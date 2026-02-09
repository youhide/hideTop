package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics"
)

func RenderCPU(cpu metrics.CPUStats, width int) string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("CPU"))
	n := len(cpu.PerCore)
	if n > 0 {
		b.WriteString(SubtleStyle.Render(fmt.Sprintf("  %d cores", n)))
	}
	b.WriteByte('\n')

	totalLabel := fmt.Sprintf("TOTAL %5.1f%%", cpu.Total)
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(renderBar(cpu.Total, totalLabel, width-4)))
	b.WriteByte('\n')

	// Two-column layout: left = cpu0..cpu4, right = cpu5..cpu9
	half := (n + 1) / 2
	colWidth := (width - 6) / 2

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
	b.WriteByte('\n')

	return PanelStyle.Width(width - 2).Render(b.String())
}

func renderBar(pct float64, label string, maxWidth int) string {
	if maxWidth < 1 {
		maxWidth = 1
	}

	labelLen := len(label) + 2
	suffixLen := 1
	barWidth := maxWidth - labelLen - suffixLen
	if barWidth < 4 {
		barWidth = 4
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
