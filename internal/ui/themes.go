package ui

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines a color palette for the UI.
type Theme struct {
	Name       string
	Title      lipgloss.Color
	Green      lipgloss.Color
	Yellow     lipgloss.Color
	Red        lipgloss.Color
	Subtle     lipgloss.Color
	Border     lipgloss.Color
	Header     lipgloss.Color
	SelectedBg lipgloss.Color
	// SelectedFg is the text color on the selected row. The row used to
	// hardcode #FFFFFF, which on the light theme's pale SelectedBg gave a
	// contrast ratio around 1.15:1 — effectively invisible.
	SelectedFg lipgloss.Color
}

var themes = map[string]Theme{
	"dark": {
		Name:       "dark",
		Title:      lipgloss.Color("#7D56F4"),
		Green:      lipgloss.Color("#04B575"),
		Yellow:     lipgloss.Color("#FBBF24"),
		Red:        lipgloss.Color("#EF4444"),
		Subtle:     lipgloss.Color("#6B7280"),
		Border:     lipgloss.Color("#3F3F46"),
		Header:     lipgloss.Color("#D4D4D8"),
		SelectedBg: lipgloss.Color("#3B3B5C"),
		SelectedFg: lipgloss.Color("#FFFFFF"),
	},
	"light": {
		Name:       "light",
		Title:      lipgloss.Color("#6D28D9"),
		Green:      lipgloss.Color("#059669"),
		Yellow:     lipgloss.Color("#D97706"),
		Red:        lipgloss.Color("#DC2626"),
		Subtle:     lipgloss.Color("#9CA3AF"),
		Border:     lipgloss.Color("#D1D5DB"),
		Header:     lipgloss.Color("#374151"),
		SelectedBg: lipgloss.Color("#E0E7FF"),
		SelectedFg: lipgloss.Color("#111827"),
	},
	"dracula": {
		Name:       "dracula",
		Title:      lipgloss.Color("#BD93F9"),
		Green:      lipgloss.Color("#50FA7B"),
		Yellow:     lipgloss.Color("#F1FA8C"),
		Red:        lipgloss.Color("#FF5555"),
		Subtle:     lipgloss.Color("#6272A4"),
		Border:     lipgloss.Color("#44475A"),
		Header:     lipgloss.Color("#F8F8F2"),
		SelectedBg: lipgloss.Color("#44475A"),
		SelectedFg: lipgloss.Color("#F8F8F2"),
	},
	"nord": {
		Name:       "nord",
		Title:      lipgloss.Color("#88C0D0"),
		Green:      lipgloss.Color("#A3BE8C"),
		Yellow:     lipgloss.Color("#EBCB8B"),
		Red:        lipgloss.Color("#BF616A"),
		Subtle:     lipgloss.Color("#4C566A"),
		Border:     lipgloss.Color("#3B4252"),
		Header:     lipgloss.Color("#ECEFF4"),
		SelectedBg: lipgloss.Color("#3B4252"),
		SelectedFg: lipgloss.Color("#ECEFF4"),
	},
	"monokai": {
		Name:       "monokai",
		Title:      lipgloss.Color("#AE81FF"),
		Green:      lipgloss.Color("#A6E22E"),
		Yellow:     lipgloss.Color("#E6DB74"),
		Red:        lipgloss.Color("#F92672"),
		Subtle:     lipgloss.Color("#75715E"),
		Border:     lipgloss.Color("#49483E"),
		Header:     lipgloss.Color("#F8F8F2"),
		SelectedBg: lipgloss.Color("#49483E"),
		SelectedFg: lipgloss.Color("#F8F8F2"),
	},
}

// AvailableThemes returns the names of all built-in themes.
func AvailableThemes() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	slices.Sort(names) // map iteration order made the list random per run
	return names
}

// ApplyTheme switches the palette to the named theme, falling back to "dark".
//
// It mutates package-level styles, so it must be called before the program
// starts rendering (main does, before tea.Run). If a runtime theme-switch key
// is ever added, the theme has to move into the model and be threaded through
// the render functions instead.
func ApplyTheme(name string) {
	t, ok := themes[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "hideTop: unknown theme %q, falling back to \"dark\" (available: %s)\n",
			name, strings.Join(AvailableThemes(), ", "))
		t = themes["dark"]
	}
	rebuildStyles(t)
}
