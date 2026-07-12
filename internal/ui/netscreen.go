package ui

import (
	"fmt"

	"github.com/youhide/hideTop/internal/metrics"
)

// networkOverlayLines builds the full (unscrolled) content for the network
// overlay: a listening-ports table followed by an active-connections table.
// Line count is independent of width, so it can also be used to size scrolling.
func networkOverlayLines(conns metrics.NetConnections, contentW int) []string {
	if !conns.Available {
		return []string{SubtleStyle.Render("  collecting listening ports and connections…")}
	}

	var lines []string

	lines = append(lines, HeaderStyle.Render(fmt.Sprintf("Listening ports (%d)", len(conns.Listening))))
	if len(conns.Listening) == 0 {
		lines = append(lines, SubtleStyle.Render("  none"))
	} else {
		lines = append(lines, SubtleStyle.Render(fmt.Sprintf("  %-5s %-7s %-8s %s", "PROTO", "PORT", "PID", "PROCESS")))
		for _, p := range conns.Listening {
			procW := contentW - 24
			if procW < 4 {
				procW = 4
			}
			proc := p.Process
			if proc == "" {
				proc = "-"
			}
			lines = append(lines, fmt.Sprintf("  %-5s %-7d %-8d %s",
				p.Proto, p.Port, p.PID, fitPlain(proc, procW)))
		}
	}

	lines = append(lines, "")

	lines = append(lines, HeaderStyle.Render(fmt.Sprintf("Connections (%d)", len(conns.Connections))))
	if len(conns.Connections) == 0 {
		lines = append(lines, SubtleStyle.Render("  none"))
	} else {
		lines = append(lines, SubtleStyle.Render(fmt.Sprintf("  %-5s %-22s %-22s %-12s %s",
			"PROTO", "LOCAL", "REMOTE", "STATE", "PROCESS")))
		for _, c := range conns.Connections {
			procW := contentW - 66
			if procW < 4 {
				procW = 4
			}
			proc := c.Process
			if proc == "" {
				proc = "-"
			}
			lines = append(lines, fmt.Sprintf("  %-5s %-22s %-22s %-12s %s",
				c.Proto, fitPlain(c.Laddr, 22), fitPlain(c.Raddr, 22),
				fitPlain(c.Status, 12), fitPlain(proc, procW)))
		}
	}

	return lines
}

// NetworkOverlayMaxScroll returns the maximum scroll offset for the network
// overlay at the given terminal size.
func NetworkOverlayMaxScroll(conns metrics.NetConnections, width, height int) int {
	return OverlayMaxScroll(len(networkOverlayLines(conns, overlayInnerWidth(width))), height)
}

// RenderNetworkOverlay renders the network/ports view inside the standard
// overlay box.
func RenderNetworkOverlay(conns metrics.NetConnections, collecting bool, width, height, scroll int) string {
	title := "Network — ports & connections"
	if collecting {
		title += "  (collecting)"
	}
	return RenderOverlay(title, func(cw int) []string {
		return networkOverlayLines(conns, cw)
	}, width, height, scroll)
}
