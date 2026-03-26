package ui

import (
	"fmt"
	"strings"

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

// RenderProcessDetail renders a full-screen overlay with extended process info.
func RenderProcessDetail(d ProcessDetail, width, height int) string {
	var b strings.Builder

	title := fmt.Sprintf("Process %d — %s", d.PID, d.Name)
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorTitle).Render(title))
	b.WriteString("\n\n")

	field := func(label, value string) {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			lipgloss.NewStyle().Bold(true).Foreground(ColorHeader).Width(14).Render(label),
			SubtleStyle.Render(value),
		))
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

	b.WriteString("\n")
	if d.Cmdline != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHeader).Render("  Command Line"))
		b.WriteString("\n")
		// Wrap long command lines
		cmd := d.Cmdline
		maxW := width - 12
		if maxW < 40 {
			maxW = 40
		}
		for len(cmd) > maxW {
			b.WriteString("  " + SubtleStyle.Render(cmd[:maxW]) + "\n")
			cmd = cmd[maxW:]
		}
		if cmd != "" {
			b.WriteString("  " + SubtleStyle.Render(cmd) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(SubtleStyle.Render("  Press Esc or Enter to close"))

	content := b.String()
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorTitle).
		Padding(1, 2).
		Width(width - 4).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
