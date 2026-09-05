package ui

import (
	"fmt"
	"strings"

	"github.com/youhide/hideTop/internal/metrics"
)

// Network panel sizing. Like the temperature panel, the visible row count
// adapts to the terminal height instead of being fixed.
const (
	netVisibleRows = 4 // default when the caller does not constrain height
	NetMinRows     = 4
	NetMaxRows     = 16
)

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
	return NetworkScrollMaxRows(delta, ports, netVisibleRows)
}

// NetworkScrollMaxRows is NetworkScrollMax for a given visible row count.
func NetworkScrollMaxRows(delta metrics.NetworkDelta, ports []metrics.PortInfo, rows int) int {
	return max(0, networkBodyLen(delta, ports)-rows)
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
	return RenderNetworkScrollRows(delta, ports, width, scroll, netVisibleRows)
}

// RenderNetworkScrollRows renders the network panel with at most rows body
// rows. The body is always padded to exactly rows lines so that interfaces
// going idle (and dropping out of the list) never change the panel height and
// shift everything below it.
func RenderNetworkScrollRows(delta metrics.NetworkDelta, ports []metrics.PortInfo, width, scroll, rows int) (string, int) {
	if !delta.Available && len(ports) == 0 {
		return "", 0
	}
	rows = min(max(rows, NetMinRows), NetMaxRows)
	innerW := contentWidth(width)

	// Build the scrollable body: active interfaces, then listening ports.
	var body []string
	if delta.Available {
		for _, j := range activeInterfaceIndexes(delta) {
			iface := delta.Interfaces[j]
			if innerW < 34 {
				body = append(body, SubtleStyle.Render("  "+padTo(iface.Name, 6))+
					fmt.Sprintf(" ↓%s ↑%s", formatBytesCompact(iface.InSec), formatBytesCompact(iface.OutSec)))
			} else {
				body = append(body, SubtleStyle.Render("  "+padTo(iface.Name, 10))+
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

	maxScroll := max(0, len(body)-rows)
	scroll = min(max(scroll, 0), maxScroll)
	end := min(scroll+rows, len(body))
	window := body[scroll:end]

	var b strings.Builder
	if delta.Available {
		if innerW < 34 {
			fmt.Fprintf(&b, "  ↓ %s/s  ↑ %s/s",
				GreenStyle.Render(formatBytesCompact(delta.TotalInSec)),
				YellowStyle.Render(formatBytesCompact(delta.TotalOutSec)),
			)
		} else {
			fmt.Fprintf(&b, "  ▼ %s/s   ▲ %s/s",
				GreenStyle.Render(formatBytes(delta.TotalInSec)),
				YellowStyle.Render(formatBytes(delta.TotalOutSec)),
			)
		}
		b.WriteByte('\n')
	}

	for _, row := range window {
		b.WriteString(row)
		b.WriteByte('\n')
	}
	// Pad to a constant number of body rows. Without this, an interface going
	// idle removes a line and everything below the panel jumps up.
	for i := len(window); i < rows; i++ {
		b.WriteByte('\n')
	}

	return renderPanel(HeaderStyle.Render("Network"), scrollStatus(scroll+1, end, len(body), maxScroll > 0),
		b.String(), width), maxScroll
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
		return fmt.Sprintf("  %5d %s %s", p.Port, padTo(p.Proto, 3), padTo(proc, pw))
	}
	pw := innerW - 16
	if pw < 1 {
		pw = 1
	}
	return fmt.Sprintf("  %s %5d  %s", padTo(p.Proto, 3), p.Port, padTo(proc, pw))
}
