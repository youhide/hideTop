package ui

import "github.com/charmbracelet/lipgloss"

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
	},
}

// AvailableThemes returns the names of all built-in themes.
func AvailableThemes() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	return names
}

// ApplyTheme switches the global color variables to use the given theme.
// Falls back to "dark" if the theme name is not recognized.
func ApplyTheme(name string) {
	t, ok := themes[name]
	if !ok {
		t = themes["dark"]
	}

	ColorTitle = t.Title
	ColorGreen = t.Green
	ColorYellow = t.Yellow
	ColorRed = t.Red
	ColorSubtle = t.Subtle
	ColorBorder = t.Border
	ColorHeader = t.Header
	ColorSelectedBg = t.SelectedBg

	// Rebuild derived styles
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorTitle).
		MarginBottom(1)

	PanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorHeader)

	SubtleStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle)

	GreenStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	YellowStyle = lipgloss.NewStyle().Foreground(ColorYellow)
	RedStyle = lipgloss.NewStyle().Foreground(ColorRed)
}
