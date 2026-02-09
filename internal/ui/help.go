package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func RenderHelp(width int) string {
	keys := []struct{ key, desc string }{
		{"↑↓/jk", "move"},
		{"/", "search"},
		{"c/m/p", "sort"},
		{"+/-", "interval"},
		{"q", "quit"},
	}

	var line string
	for i, k := range keys {
		if i > 0 {
			line += SubtleStyle.Render("  │  ")
		}
		line += fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorTitle).Render(k.key),
			SubtleStyle.Render(k.desc),
		)
	}

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(line)
}
