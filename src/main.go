package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/youhide/hideTop/internal/app"
	"github.com/youhide/hideTop/internal/config"
)

func main() {
	cfg := config.Parse()
	m := app.New(cfg)

	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "hideTop: %v\n", err)
		os.Exit(1)
	}
}
