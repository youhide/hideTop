package ui

import (
	"fmt"
	"strings"

	"github.com/youhide/hideTop/internal/metrics"
)

// RenderNetwork renders the network panel. Returns empty if no data.
func RenderNetwork(delta metrics.NetworkDelta, width int) string {
	if !delta.Available {
		return ""
	}

	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Network"))
	b.WriteByte('\n')

	// Total throughput
	b.WriteString(fmt.Sprintf("  ▼ %s/s   ▲ %s/s",
		GreenStyle.Render(formatBytes(delta.TotalInSec)),
		YellowStyle.Render(formatBytes(delta.TotalOutSec)),
	))
	b.WriteByte('\n')

	// Per-interface (limit to top 4, skip inactive)
	maxIfaces := 4
	shown := 0
	for _, iface := range delta.Interfaces {
		if shown >= maxIfaces {
			break
		}
		// Skip interfaces with zero traffic
		if iface.InSec == 0 && iface.OutSec == 0 {
			continue
		}
		b.WriteString(SubtleStyle.Render(fmt.Sprintf("  %-10s", truncateStr(iface.Name, 10))))
		b.WriteString(fmt.Sprintf("  ▼ %s/s  ▲ %s/s",
			formatBytes(iface.InSec),
			formatBytes(iface.OutSec),
		))
		b.WriteByte('\n')
		shown++
	}

	return PanelStyle.Width(width - 2).Render(b.String())
}

// formatBytes formats bytes into human-readable format.
func formatBytes(bytes float64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GiB", bytes/(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MiB", bytes/(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KiB", bytes/(1<<10))
	default:
		return fmt.Sprintf("%.0f B", bytes)
	}
}

// truncateStr truncates a string to maxLen runes, preserving valid UTF-8.
func truncateStr(s string, maxLen int) string {
	return truncateRunes(s, maxLen)
}
