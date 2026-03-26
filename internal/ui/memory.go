package ui

import (
	"fmt"
	"strings"

	"github.com/youhide/hideTop/internal/metrics"
)

func RenderMemory(mem metrics.MemoryStats, load metrics.LoadAvg, width int, history []float64) string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Memory"))
	b.WriteByte('\n')

	label := fmt.Sprintf("used %5.1f%%  %.1f/%.1f GiB", mem.Percent, mem.UsedGB, mem.TotalGB)
	b.WriteString(renderBar(mem.Percent, label, width-4))
	b.WriteByte('\n')
	if mem.SwapTotalGB > 0 {
		swapLabel := fmt.Sprintf("swap %5.1f%%  %.1f/%.1f GiB", mem.SwapPercent, mem.SwapUsedGB, mem.SwapTotalGB)
		b.WriteString(renderBar(mem.SwapPercent, swapLabel, width-4))
		b.WriteByte('\n')
	}

	b.WriteString(SubtleStyle.Render(
		fmt.Sprintf("  load: %.2f  %.2f  %.2f", load.Load1, load.Load5, load.Load15),
	))

	// Sparkline history
	if len(history) > 1 {
		b.WriteByte('\n')
		b.WriteString(RenderSparklineCompact("mem", history, width-4))
	}

	return PanelStyle.Width(width - 2).Render(b.String())
}
