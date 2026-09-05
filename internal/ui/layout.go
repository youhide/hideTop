package ui

import (
	"fmt"
	"strings"

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

// formatBytesCompact formats a byte count right-aligned in a fixed 5 cells
// ("999.9K", " 12.3M", "    0B"). Constant width so that two rates on the same
// line never shift each other between frames, and right-aligned so a "/s"
// suffix sits flush against the unit.
func formatBytesCompact(bytes float64) string {
	value, unit := scaleBytes(bytes)
	if unit == "B" {
		return fmt.Sprintf("%5s", fmt.Sprintf("%.0f%s", value, unit))
	}
	return fmt.Sprintf("%5s", fmt.Sprintf("%.1f%s", value, unit))
}

// formatBytes formats a byte count right-aligned in a fixed 9 cells
// ("999.9 KiB", "  1.2 MiB", "    554 B"). Same reasoning as
// formatBytesCompact.
func formatBytes(bytes float64) string {
	value, unit := scaleBytes(bytes)
	if unit == "B" {
		return fmt.Sprintf("%9s", fmt.Sprintf("%.0f %s", value, unit))
	}
	return fmt.Sprintf("%9s", fmt.Sprintf("%.1f %siB", value, unit))
}

// scaleBytes reduces a byte count to a value and its unit suffix.
func scaleBytes(bytes float64) (float64, string) {
	switch {
	case bytes >= 1<<30:
		return bytes / (1 << 30), "G"
	case bytes >= 1<<20:
		return bytes / (1 << 20), "M"
	case bytes >= 1<<10:
		return bytes / (1 << 10), "K"
	default:
		return bytes, "B"
	}
}

// padTo pads s on the right so it occupies exactly cells terminal columns,
// truncating first if it is wider.
//
// fmt's %-Ns verb counts runes, not display cells. Every call site here first
// trims to a cell budget with fitPlain and then padded with %-Ns, so a process
// named "日本語プロセス" (7 runes, 14 cells) survived the trim and then got 13
// spaces appended — a 27-cell field in a 20-cell column, which wrapped the row
// and grew the panel by a line.
func padTo(s string, cells int) string {
	if cells <= 0 {
		return ""
	}
	s = fitPlain(s, cells)
	if pad := cells - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}
