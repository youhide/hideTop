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

	label := func(b Binding) string {
		if b.HintKeys != "" {
			return b.HintKeys
		}
		return b.Display
	}
	segment := func(b Binding) string {
		return keyStyle.Render(label(b)) + " " + SubtleStyle.Render(b.Hint)
	}

	// Two orders are at play. HintPri decides what to drop as the bar narrows
	// (help and quit outlast the rest), but the bar itself reads in table
	// order, so the keys do not reshuffle as the terminal is resized.
	var candidates []int
	for i, b := range Bindings {
		if b.Hint != "" {
			candidates = append(candidates, i)
		}
	}
	byPriority := slices.Clone(candidates)
	slices.SortFunc(byPriority, func(a, b int) int { return Bindings[a].HintPri - Bindings[b].HintPri })

	keep := map[int]bool{}
	used := 0
	for _, i := range byPriority {
		cost := lipgloss.Width(segment(Bindings[i]))
		if len(keep) > 0 {
			cost += lipgloss.Width(sep)
		}
		if used+cost > width {
			continue
		}
		keep[i] = true
		used += cost
	}

	var parts []string
	for _, i := range candidates {
		if keep[i] {
			parts = append(parts, segment(Bindings[i]))
		}
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
