package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderHelp(width int) string {
	keys := []struct{ key, desc string }{
		{"↑↓/jk", "move"},
		{"/", "search"},
		{"c/m/p", "sort"},
		{"t", "tree"},
		{"s", "sys filter"},
		{"Enter", "detail"},
		{"x/K", "kill"},
		{"+/-", "interval"},
		{"e", "export"},
		{"?", "help"},
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

// RenderHelpOverlay renders a full-screen help overlay.
func RenderHelpOverlay(width, height int, version string) string {
	sections := []struct {
		title string
		keys  []struct{ key, desc string }
	}{
		{
			title: "Navigation",
			keys: []struct{ key, desc string }{
				{"↑ / k", "Move up in process list"},
				{"↓ / j", "Move down in process list"},
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
				{"+/=", "Increase refresh interval (+250ms)"},
				{"-/_", "Decrease refresh interval (-250ms)"},
				{"e", "Export snapshot to JSON"},
				{"?", "Toggle this help overlay"},
				{"q / Ctrl+C", "Quit"},
			},
		},
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorTitle).Render("hideTop Help"))
	if version != "" {
		b.WriteString(SubtleStyle.Render("  " + version))
	}
	b.WriteString("\n\n")

	for _, section := range sections {
		b.WriteString(HeaderStyle.Render(section.title))
		b.WriteString("\n")
		for _, k := range section.keys {
			b.WriteString(fmt.Sprintf("  %s  %s\n",
				lipgloss.NewStyle().Bold(true).Foreground(ColorTitle).Width(12).Render(k.key),
				SubtleStyle.Render(k.desc),
			))
		}
		b.WriteString("\n")
	}

	b.WriteString(SubtleStyle.Render("Press ? or Esc to close"))

	content := b.String()
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorTitle).
		Padding(1, 2).
		Width(width - 4).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
