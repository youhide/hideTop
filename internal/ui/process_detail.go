package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/youhide/hideTop/internal/metrics"
)

// ProcessDetail holds extended info for the process detail overlay.
type ProcessDetail struct {
	metrics.ProcessInfo
	Cmdline    string
	NumFDs     int32
	RSS        uint64 // bytes
	VMS        uint64 // bytes
	CreateTime int64  // milliseconds since epoch
}

// RenderProcessDetail renders the process detail overlay (non-scrolled).
func RenderProcessDetail(d ProcessDetail, width, height int) string {
	return RenderProcessDetailScroll(d, width, height, 0)
}

// RenderProcessDetailScroll renders the process detail overlay at a scroll
// offset, inside the standard overlay box.
func RenderProcessDetailScroll(d ProcessDetail, width, height, scroll int) string {
	title := fmt.Sprintf("Process %d — %s", d.PID, d.Name)
	return RenderOverlay(title, func(cw int) []string {
		return processDetailLines(d, cw)
	}, width, height, scroll)
}

// ProcessDetailMaxScroll returns the max scroll offset for the detail overlay.
func ProcessDetailMaxScroll(d ProcessDetail, width, height int) int {
	return OverlayMaxScroll(len(processDetailLines(d, overlayInnerWidth(width))), height)
}

// processDetailLines builds the detail content as one string per line.
func processDetailLines(d ProcessDetail, cw int) []string {
	var lines []string

	field := func(label, value string) {
		valueW := cw - 18
		if valueW < 1 {
			valueW = 1
		}
		lines = append(lines, "  "+
			lipgloss.NewStyle().Bold(true).Foreground(ColorHeader).Width(14).Render(label)+"  "+
			SubtleStyle.Render(fitPlain(value, valueW)))
	}

	field("PID", fmt.Sprintf("%d", d.PID))
	field("PPID", fmt.Sprintf("%d", d.PPID))
	field("User", d.User)
	field("State", stateLabel(d.State)+" ("+d.State+")")
	field("Threads", fmt.Sprintf("%d", d.NumThreads))
	field("CPU%", fmt.Sprintf("%.1f%%", d.CPUPercent))
	field("MEM%", fmt.Sprintf("%.1f%%", d.MemPercent))

	if d.RSS > 0 {
		field("RSS", formatBytes(float64(d.RSS)))
	}
	if d.VMS > 0 {
		field("VMS", formatBytes(float64(d.VMS)))
	}
	if d.NumFDs > 0 {
		field("Open FDs", fmt.Sprintf("%d", d.NumFDs))
	}

	if d.Cmdline != "" {
		lines = append(lines, "")
		lines = append(lines, HeaderStyle.Render("  Command Line"))
		maxW := cw - 2
		if maxW < 1 {
			maxW = 1
		}
		for _, line := range wrapPlain(d.Cmdline, maxW) {
			lines = append(lines, "  "+SubtleStyle.Render(line))
		}
	}

	return lines
}
