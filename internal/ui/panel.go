package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderPanel wraps a panel body in the standard bordered frame.
//
// head is the already-styled title line (panels put their own chips on it).
// status is an optional right-aligned note on that line, e.g. the scroll
// position. Putting it there rather than on its own row saves a line per
// scrollable panel without reintroducing the height changes that a
// conditionally-rendered status row caused.
//
// Exactly one trailing newline is trimmed from body: lipgloss renders it as an
// extra blank line inside the border, which every scrollable panel was paying
// for. Only one, so that deliberate blank-line padding (the network panel pads
// its body to a constant row count) survives.
func renderPanel(head, status, body string, width int) string {
	innerW := contentWidth(width)

	if status != "" {
		gap := innerW - lipgloss.Width(head) - lipgloss.Width(status)
		if gap >= 1 {
			head += strings.Repeat(" ", gap) + SubtleStyle.Render(status)
		}
	}

	var b strings.Builder
	b.WriteString(head)
	if body != "" {
		b.WriteByte('\n')
		b.WriteString(strings.TrimSuffix(body, "\n"))
	}
	return PanelStyle.Width(panelWidth(width)).Render(b.String())
}

// scrollStatus formats the "3-8/40" indicator shown on a scrollable panel's
// title line. Returns empty when everything fits.
func scrollStatus(first, last, total int, scrollable bool) string {
	if !scrollable {
		return ""
	}
	return fmt.Sprintf("%d-%d/%d", first, last, total)
}
