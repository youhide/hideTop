package ui

import "github.com/charmbracelet/lipgloss"

var (
	ColorTitle      = lipgloss.Color("#7D56F4")
	ColorGreen      = lipgloss.Color("#04B575")
	ColorYellow     = lipgloss.Color("#FBBF24")
	ColorRed        = lipgloss.Color("#EF4444")
	ColorSubtle     = lipgloss.Color("#6B7280")
	ColorBorder     = lipgloss.Color("#3F3F46")
	ColorHeader     = lipgloss.Color("#D4D4D8")
	ColorSelectedBg = lipgloss.Color("#2D2D3D")
)

var (
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

	GreenStyle  = lipgloss.NewStyle().Foreground(ColorGreen)
	YellowStyle = lipgloss.NewStyle().Foreground(ColorYellow)
	RedStyle    = lipgloss.NewStyle().Foreground(ColorRed)
)

func BarColor(pct float64) lipgloss.Color {
	switch {
	case pct > 80:
		return ColorRed
	case pct > 50:
		return ColorYellow
	default:
		return ColorGreen
	}
}
