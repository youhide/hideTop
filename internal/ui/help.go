package ui

import (
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderHelp renders the single-line hint bar at the bottom of the screen.
// It is always exactly one line (MaxHeight 1) so it can never wrap and get
// clipped by the height guard; `?` opens the full help overlay with everything.
func RenderHelp(width int) string {
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorTitle)
	sep := SubtleStyle.Render("   ")

	// Greedily fill the bar in priority order, measuring in display cells so a
	// narrow terminal degrades instead of wrapping.
	hints := make([]Binding, 0, len(Bindings))
	for _, b := range Bindings {
		if b.Hint != "" {
			hints = append(hints, b)
		}
	}
	slices.SortFunc(hints, func(a, b Binding) int { return a.HintPri - b.HintPri })

	var parts []string
	used := 0
	for _, h := range hints {
		seg := keyStyle.Render(h.Display) + " " + SubtleStyle.Render(h.Hint)
		cost := lipgloss.Width(seg)
		if len(parts) > 0 {
			cost += lipgloss.Width(sep)
		}
		if used+cost > width {
			continue
		}
		parts = append(parts, seg)
		used += cost
	}

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		MaxHeight(1).
		Render(strings.Join(parts, sep))
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
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorTitle)

	var lines []string
	for si, section := range sections() {
		if si > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, HeaderStyle.Render(section))
		for _, b := range Bindings {
			if b.Section != section {
				continue
			}
			key := keyStyle.Width(14).Render(b.Display)
			descW := cw - 14
			if descW < 1 {
				descW = 1
			}
			wrapped := wrapPlain(b.Desc, descW)
			if len(wrapped) == 0 {
				wrapped = []string{""}
			}
			lines = append(lines, key+SubtleStyle.Render(wrapped[0]))
			for _, cont := range wrapped[1:] {
				lines = append(lines, strings.Repeat(" ", 14)+SubtleStyle.Render(cont))
			}
		}
	}
	return lines
}
