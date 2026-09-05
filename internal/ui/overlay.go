package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Overlay box geometry. A lipgloss style with Padding(1,2) + rounded border
// renders as: outer = Width+2 (border) wide, and content+2 (padding) + 2
// (border) tall. We size everything from the terminal so the box always fits
// and its top border is never scrolled off-screen.

// overlayInnerWidth is the width available for content lines inside the box.
func overlayInnerWidth(width int) int {
	if width < 20 {
		width = 20
	}
	cw := width - 8 // 2 border + 4 padding + 2 outer margin
	if cw < 10 {
		cw = 10
	}
	return cw
}

// overlayContentRows is how many scrollable content rows fit inside the box.
func overlayContentRows(height int) int {
	if height < 6 {
		height = 6
	}
	bh := height - 6 // 2 border + 2 padding + 2 outer margin
	if bh < 3 {
		bh = 3
	}
	cr := bh - 3 // title row + blank row + footer row
	if cr < 1 {
		cr = 1
	}
	return cr
}

// OverlayMaxScroll returns the maximum scroll offset for an overlay whose
// content has n lines, at the given terminal height.
func OverlayMaxScroll(n, height int) int {
	m := n - overlayContentRows(height)
	if m < 0 {
		return 0
	}
	return m
}

// RenderOverlay draws a centered, rounded-border box that always fits within
// the terminal (so the top border is never clipped) and scrolls its content
// when it does not fit. build receives the inner content width so lines can be
// sized to fit. This is the shared standard used by every full-screen overlay.
func RenderOverlay(title string, build func(cw int) []string, width, height, scroll int) string {
	cw := overlayInnerWidth(width)
	contentRows := overlayContentRows(height)
	content := build(cw)

	maxScroll := len(content) - contentRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	end := scroll + contentRows
	if end > len(content) {
		end = len(content)
	}
	window := content[scroll:end]

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorTitle).Render(fitPlain(title, cw)))
	b.WriteByte('\n')
	b.WriteByte('\n')
	for i := 0; i < contentRows; i++ {
		if i < len(window) {
			b.WriteString(window[i])
		}
		b.WriteByte('\n')
	}
	pos := scroll + 1
	if pos > len(content) {
		pos = len(content)
	}
	hint := "Esc close"
	if maxScroll > 0 {
		hint = fmt.Sprintf("↑↓ scroll · %d-%d of %d · Esc close", pos, end, len(content))
	}
	b.WriteString(SubtleStyle.Render(fitPlain(hint, cw)))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorTitle).
		Padding(1, 2).
		Width(cw + 4). // content width + horizontal padding; border adds 2 more
		Render(b.String())

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// RenderConfirm draws a centred confirmation dialog over a dimmed background.
//
// The kill prompt used to be a chip appended to the header, which
// appendHeaderMessage drops silently when the line is full — so on a narrow
// terminal the app could sit waiting for a keypress that sends SIGKILL with
// nothing on screen saying so.
func RenderConfirm(title, body, hint string, width, height int) string {
	inner := max(20, min(60, width-8))

	var lines []string
	lines = append(lines, RedStyle.Bold(true).Render(fitPlain(title, inner)), "")
	lines = append(lines, wrapPlain(body, inner)...)
	lines = append(lines, "", SubtleStyle.Render(fitPlain(hint, inner)))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorRed).
		Padding(1, 2).
		Width(inner).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
