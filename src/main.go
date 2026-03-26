package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/youhide/hideTop/internal/app"
	"github.com/youhide/hideTop/internal/config"
	"github.com/youhide/hideTop/internal/ui"
)

var Version = "dev"

func main() {
	cfg := config.Parse()
	if cfg.ShowVersion {
		fmt.Printf("hideTop %s\n", Version)
		return
	}

	// Setup structured logging
	if cfg.Debug {
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
		slog.SetDefault(slog.New(handler))
		slog.Debug("debug mode enabled", "version", Version)
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}

	m := app.New(cfg)
	m.SetVersion(Version)

	if cfg.Theme != "" {
		ui.ApplyTheme(cfg.Theme)
	}

	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "hideTop: %v\n", err)
		os.Exit(1)
	}
}
