package ui

import (
	"fmt"
	"strings"

	"github.com/youhide/hideTop/internal/metrics"
)

func RenderMemory(mem metrics.MemoryStats, load metrics.LoadAvg, width int, history []float64) string {
	var b strings.Builder
	innerW := contentWidth(width)

	b.WriteString(HeaderStyle.Render("Memory"))
	b.WriteByte('\n')

	label := fmt.Sprintf("used %5.1f%%  %.1f/%.1f GiB", mem.Percent, mem.UsedGB, mem.TotalGB)
	b.WriteString(renderBar(mem.Percent, label, innerW))
	b.WriteByte('\n')
	if mem.SwapTotalGB > 0 {
		swapLabel := fmt.Sprintf("swap %5.1f%%  %.1f/%.1f GiB", mem.SwapPercent, mem.SwapUsedGB, mem.SwapTotalGB)
		b.WriteString(renderBar(mem.SwapPercent, swapLabel, innerW))
		b.WriteByte('\n')
	}

	loadLine := fmt.Sprintf("  load: %.2f  %.2f  %.2f", load.Load1, load.Load5, load.Load15)
	b.WriteString(SubtleStyle.Render(fitPlain(loadLine, innerW)))

	// Sparkline history
	if len(history) > 1 {
		b.WriteByte('\n')
		b.WriteString(RenderSparklineCompact("mem", history, innerW))
	}

	return PanelStyle.Width(panelWidth(width)).Render(b.String())
}
