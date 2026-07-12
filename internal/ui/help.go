package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderHelp renders the single-line hint bar at the bottom of the screen.
// It is always exactly one line (MaxHeight 1) so it can never wrap and get
// clipped by the height guard; `?` opens the full help overlay with everything.
func RenderHelp(width int) string {
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorTitle)
	seg := func(k, d string) string {
		return keyStyle.Render(k) + " " + SubtleStyle.Render(d)
	}

	var keys []struct{ key, desc string }
	switch {
	case width < 44:
		keys = []struct{ key, desc string }{
			{"?", "help"}, {"q", "quit"},
		}
	case width < 80:
		keys = []struct{ key, desc string }{
			{"↑↓", "move"}, {"/", "search"}, {"n", "net"}, {"?", "help"}, {"q", "quit"},
		}
	default:
		keys = []struct{ key, desc string }{
			{"↑↓/jk", "move"}, {"/", "search"}, {"c/m/p", "sort"},
			{"n", "net"}, {"?", "more"}, {"q", "quit"},
		}
	}

	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = seg(k.key, k.desc)
	}
	line := strings.Join(parts, SubtleStyle.Render("   "))

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		MaxHeight(1).
		Render(line)
}

// RenderHelpOverlay renders the full-screen help overlay (non-scrolled).
func RenderHelpOverlay(width, height int, version string) string {
	return RenderHelpOverlayScroll(width, height, version, 0)
}

// RenderHelpOverlayScroll renders the help overlay at a scroll offset.
func RenderHelpOverlayScroll(width, height int, version string, scroll int) string {
	title := "hideTop Help"
	if version != "" {
		title += "  " + version
	}
	return RenderOverlay(title, helpOverlayLines, width, height, scroll)
}

// HelpOverlayMaxScroll returns the max scroll offset for the help overlay.
func HelpOverlayMaxScroll(width, height int) int {
	return OverlayMaxScroll(len(helpOverlayLines(overlayInnerWidth(width))), height)
}

// helpOverlayLines builds the help content as one string per line, fitted to
// the given inner content width.
func helpOverlayLines(cw int) []string {
	sections := []struct {
		title string
		keys  []struct{ key, desc string }
	}{
		{
			title: "Navigation",
			keys: []struct{ key, desc string }{
				{"↑ / k", "Move up in process list"},
				{"↓ / j", "Move down in process list"},
				{"PgUp/PgDn", "Jump one page up / down"},
				{"Home/End", "Jump to first / last (g / G)"},
				{"Wheel", "Scroll process list; over Temp/Net panels scrolls them"},
				{"/", "Start incremental search"},
				{"Esc", "Cancel search / close help / close detail"},
				{"Enter", "Open process detail panel"},
			},
		},
		{
			title: "Sorting",
			keys: []struct{ key, desc string }{
				{"c", "Sort by CPU% (descending)"},
				{"m", "Sort by MEM% (descending)"},
				{"p", "Sort by PID (ascending)"},
				{"Click", "Click a PID/CPU%/MEM% column header to sort"},
			},
		},
		{
			title: "Process Actions",
			keys: []struct{ key, desc string }{
				{"t", "Toggle tree view"},
				{"s", "Toggle system process filter"},
				{"x", "Kill selected process (SIGTERM)"},
				{"K", "Force kill (SIGKILL)"},
			},
		},
		{
			title: "Display",
			keys: []struct{ key, desc string }{
				{"n", "Open network / ports view"},
				{"Space", "Pause / resume auto-refresh"},
				{"z", "Reset Temp/Network panel scroll"},
				{"+/=", "Increase refresh interval (+250ms)"},
				{"-/_", "Decrease refresh interval (-250ms)"},
				{"e", "Export snapshot to JSON"},
				{"?", "Toggle this help overlay"},
				{"q / Ctrl+C", "Quit"},
			},
		},
	}

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorTitle)
	var lines []string
	for si, section := range sections {
		if si > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, HeaderStyle.Render(fitPlain(section.title, cw)))
		descW := cw - 16
		if descW < 1 {
			descW = 1
		}
		for _, k := range section.keys {
			lines = append(lines, "  "+keyStyle.Width(12).Render(k.key)+"  "+
				SubtleStyle.Render(fitPlain(k.desc, descW)))
		}
	}
	return lines
}
