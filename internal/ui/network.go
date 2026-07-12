package ui

import (
	"fmt"
	"strings"

	"github.com/youhide/hideTop/internal/metrics"
)

// netVisibleRows is how many body rows (interfaces + ports) are shown at once.
const netVisibleRows = 4

// activeInterfaceIndexes returns the indexes of interfaces with non-zero
// traffic, in original order.
func activeInterfaceIndexes(delta metrics.NetworkDelta) []int {
	var idx []int
	for i, iface := range delta.Interfaces {
		if iface.InSec == 0 && iface.OutSec == 0 {
			continue
		}
		idx = append(idx, i)
	}
	return idx
}

// networkBodyLen returns the number of scrollable body rows (active interfaces
// plus a listening-ports sub-section). Independent of width.
func networkBodyLen(delta metrics.NetworkDelta, ports []metrics.PortInfo) int {
	n := 0
	if delta.Available {
		n += len(activeInterfaceIndexes(delta))
	}
	if len(ports) > 0 {
		n += 1 + len(ports) // sub-header + one row per port
	}
	return n
}

// NetworkScrollMax returns the maximum scroll offset for the network panel.
func NetworkScrollMax(delta metrics.NetworkDelta, ports []metrics.PortInfo) int {
	m := networkBodyLen(delta, ports) - netVisibleRows
	if m < 0 {
		return 0
	}
	return m
}

// RenderNetwork renders the network panel. Returns empty if no data.
func RenderNetwork(delta metrics.NetworkDelta, width int) string {
	s, _ := RenderNetworkScroll(delta, nil, width, 0)
	return s
}

// RenderNetworkScroll renders the network panel: total throughput plus a
// scrollable body of active interfaces and listening ports. Returns the panel
// and the clamped maximum scroll offset.
func RenderNetworkScroll(delta metrics.NetworkDelta, ports []metrics.PortInfo, width, scroll int) (string, int) {
	if !delta.Available && len(ports) == 0 {
		return "", 0
	}
	innerW := contentWidth(width)

	// Build the scrollable body: active interfaces, then listening ports.
	var body []string
	if delta.Available {
		for _, j := range activeInterfaceIndexes(delta) {
			iface := delta.Interfaces[j]
			if innerW < 34 {
				body = append(body, SubtleStyle.Render(fmt.Sprintf("  %-6s", fitPlain(iface.Name, 6)))+
					fmt.Sprintf(" ↓%s ↑%s", formatBytesCompact(iface.InSec), formatBytesCompact(iface.OutSec)))
			} else {
				body = append(body, SubtleStyle.Render(fmt.Sprintf("  %-10s", truncateStr(iface.Name, 10)))+
					fmt.Sprintf("  ▼ %s/s  ▲ %s/s", formatBytes(iface.InSec), formatBytes(iface.OutSec)))
			}
		}
	}
	if len(ports) > 0 {
		body = append(body, SubtleStyle.Render(fitPlain(fmt.Sprintf("  Listening (%d)", len(ports)), innerW)))
		for _, p := range ports {
			body = append(body, portRow(p, innerW))
		}
	}

	maxScroll := len(body) - netVisibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	end := scroll + netVisibleRows
	if end > len(body) {
		end = len(body)
	}
	window := body[scroll:end]

	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Network"))
	b.WriteByte('\n')

	if delta.Available {
		if innerW < 34 {
			b.WriteString(fmt.Sprintf("  ↓ %s/s  ↑ %s/s",
				GreenStyle.Render(formatBytesCompact(delta.TotalInSec)),
				YellowStyle.Render(formatBytesCompact(delta.TotalOutSec)),
			))
		} else {
			b.WriteString(fmt.Sprintf("  ▼ %s/s   ▲ %s/s",
				GreenStyle.Render(formatBytes(delta.TotalInSec)),
				YellowStyle.Render(formatBytes(delta.TotalOutSec)),
			))
		}
		b.WriteByte('\n')
	}

	for _, row := range window {
		b.WriteString(row)
		b.WriteByte('\n')
	}

	// Constant-height scroll status so scrolling never changes the panel height.
	if maxScroll > 0 {
		status := fmt.Sprintf("  %d-%d of %d  (scroll)", scroll+1, end, len(body))
		b.WriteString(SubtleStyle.Render(fitPlain(status, innerW)))
		b.WriteByte('\n')
	}

	return PanelStyle.Width(panelWidth(width)).Render(b.String()), maxScroll
}

// portRow renders one listening-port line for the network panel.
func portRow(p metrics.PortInfo, innerW int) string {
	proc := p.Process
	if proc == "" {
		proc = fmt.Sprintf("pid %d", p.PID)
	}
	if innerW < 34 {
		pw := innerW - 12
		if pw < 1 {
			pw = 1
		}
		return fmt.Sprintf("  %5d %-3s %s", p.Port, p.Proto, fitPlain(proc, pw))
	}
	pw := innerW - 16
	if pw < 1 {
		pw = 1
	}
	return fmt.Sprintf("  %-3s %5d  %s", p.Proto, p.Port, fitPlain(proc, pw))
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
