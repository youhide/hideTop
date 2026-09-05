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
	innerW := contentWidth(width)
	head := HeaderStyle.Render("Disk")

	// I/O throughput
	if delta.Available {
		if innerW < 34 {
			fmt.Fprintf(&b, "  r %s/s  w %s/s",
				GreenStyle.Render(formatBytesCompact(delta.ReadSec)),
				YellowStyle.Render(formatBytesCompact(delta.WriteSec)),
			)
		} else {
			fmt.Fprintf(&b, "  read %s/s   write %s/s",
				GreenStyle.Render(formatBytes(delta.ReadSec)),
				YellowStyle.Render(formatBytes(delta.WriteSec)),
			)
		}
		b.WriteByte('\n')
	}

	// Root filesystem usage
	if disk.RootTotalGB > 0 {
		label := fmt.Sprintf("/     %5.1f%%  %.1f / %.1f GiB", disk.RootPercent, disk.RootUsedGB, disk.RootTotalGB)
		if innerW < 34 {
			label = fmt.Sprintf("/ %4.0f%% %.0f/%.0fG", disk.RootPercent, disk.RootUsedGB, disk.RootTotalGB)
		}
		b.WriteString(renderBar(disk.RootPercent, label, innerW))
		b.WriteByte('\n')
	}

	return renderPanel(head, "", b.String(), width)
}
