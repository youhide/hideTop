package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func panelWidth(width int) int {
	w := width - 2
	if w < 1 {
		return 1
	}
	return w
}

func contentWidth(width int) int {
	w := width - 4
	if w < 1 {
		return 1
	}
	return w
}

func fitPlain(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return fitPlainRunes(s, maxWidth)
	}
	return fitPlainRunes(s, maxWidth-3) + "..."
}

func fitPlainRunes(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	var out []rune
	width := 0
	for _, r := range s {
		next := lipgloss.Width(string(r))
		if width+next > maxWidth {
			break
		}
		out = append(out, r)
		width += next
	}
	return string(out)
}

func wrapPlain(s string, maxWidth int) []string {
	if maxWidth <= 0 {
		return nil
	}
	if s == "" {
		return nil
	}

	var lines []string
	var line []rune
	width := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if width > 0 && width+rw > maxWidth {
			lines = append(lines, string(line))
			line = line[:0]
			width = 0
		}
		line = append(line, r)
		width += rw
	}
	if len(line) > 0 {
		lines = append(lines, string(line))
	}
	return lines
}

func formatBytesCompact(bytes float64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1fG", bytes/(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1fM", bytes/(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1fK", bytes/(1<<10))
	default:
		return fmt.Sprintf("%.0fB", bytes)
	}
}
