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

	label := fmt.Sprintf("used  %5.1f%%  %.1f / %.1f GiB", mem.Percent, mem.UsedGB, mem.TotalGB)
	b.WriteString(renderBar(mem.Percent, label, width-4))
	b.WriteByte('\n')
	b.WriteByte('\n')

	b.WriteString(HeaderStyle.Render("Load Average"))
	b.WriteByte('\n')
	b.WriteString(SubtleStyle.Render(
		fmt.Sprintf("  1m: %.2f   5m: %.2f   15m: %.2f", load.Load1, load.Load5, load.Load15),
	))
	b.WriteByte('\n')

	return PanelStyle.Width(width).Render(b.String())
}
