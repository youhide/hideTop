package ui

import (
	"fmt"
	"strings"

	"github.com/youhide/hideTop/internal/metrics"
)

// RenderDisk renders the disk panel with I/O throughput and usage.
func RenderDisk(delta metrics.DiskDelta, disk metrics.DiskStats, width int) string {
	if !disk.Available {
		return ""
	}

	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Disk"))
	b.WriteByte('\n')

	// I/O throughput
	if delta.Available {
		b.WriteString(fmt.Sprintf("  read %s/s   write %s/s",
			GreenStyle.Render(formatBytes(delta.ReadSec)),
			YellowStyle.Render(formatBytes(delta.WriteSec)),
		))
		b.WriteByte('\n')
	}

	// Root filesystem usage
	if disk.RootTotalGB > 0 {
		label := fmt.Sprintf("/     %5.1f%%  %.1f / %.1f GiB", disk.RootPercent, disk.RootUsedGB, disk.RootTotalGB)
		b.WriteString(renderBar(disk.RootPercent, label, width-4))
		b.WriteByte('\n')
	}

	return PanelStyle.Width(width - 2).Render(b.String())
}
