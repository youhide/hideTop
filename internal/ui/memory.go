package ui

import (
	"fmt"
	"strings"

	"github.com/youhide/hideTop/internal/metrics"
)

func RenderMemory(mem metrics.MemoryStats, load metrics.LoadAvg, width int) string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Memory"))
	b.WriteByte('\n')

	label := fmt.Sprintf("used  %5.1f%%  %.1f / %.1f GiB   avail %.1f GiB", mem.Percent, mem.UsedGB, mem.TotalGB, mem.AvailableGB)
	b.WriteString(renderBar(mem.Percent, label, width-4))
	b.WriteByte('\n')
	if mem.SwapTotalGB > 0 {
		swapLabel := fmt.Sprintf("swap  %5.1f%%  %.1f / %.1f GiB", mem.SwapPercent, mem.SwapUsedGB, mem.SwapTotalGB)
		b.WriteString(renderBar(mem.SwapPercent, swapLabel, width-4))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	b.WriteString(HeaderStyle.Render("Load Average"))
	b.WriteByte('\n')
	b.WriteString(SubtleStyle.Render(
		fmt.Sprintf("  1m: %.2f   5m: %.2f   15m: %.2f", load.Load1, load.Load5, load.Load15),
	))
	b.WriteByte('\n')

	return PanelStyle.Width(width - 2).Render(b.String())
}
