package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

	logPath, closeLog := setupLogging(cfg.Debug)
	defer closeLog()

	if err := applyTheme(cfg.Theme); err != nil {
		fmt.Fprintf(os.Stderr, "hideTop: %v\n", err)
		os.Exit(2)
	}

	m := app.New(cfg)
	m.SetVersion(Version)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	final, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "hideTop: %v\n", err)
		os.Exit(1)
	}

	// Reported here rather than inside Update: writing to stderr while the
	// alt-screen is still up paints into a buffer the user never sees.
	if fm, ok := final.(app.Model); ok && fm.SaveError() != nil {
		fmt.Fprintf(os.Stderr, "hideTop: failed to save settings: %v\n", fm.SaveError())
	}
	if logPath != "" {
		fmt.Fprintf(os.Stderr, "hideTop: debug log written to %s\n", logPath)
	}
}

// applyTheme validates the theme name before the alt-screen takes over.
// ui.ApplyTheme used to warn on stderr from inside the TUI, where the message
// was painted into a buffer the user never sees.
func applyTheme(name string) error {
	if name == "" {
		// Pick a palette that suits the terminal instead of always starting
		// dark, which left light-terminal users with an unreadable UI until
		// they discovered --theme.
		if lipgloss.HasDarkBackground() {
			return nil
		}
		ui.ApplyTheme("light")
		return nil
	}
	if !slices.Contains(ui.AvailableThemes(), name) {
		return fmt.Errorf("unknown theme %q (available: %s)", name, strings.Join(ui.AvailableThemes(), ", "))
	}
	ui.ApplyTheme(name)
	return nil
}

// setupLogging routes slog to a file. The app runs in the alt-screen, so debug
// output on stderr was invisible; the path is printed on exit instead.
func setupLogging(debug bool) (path string, closeFn func()) {
	if !debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
		return "", func() {}
	}

	dir, err := config.StateDir()
	if err == nil {
		err = os.MkdirAll(dir, 0o755)
	}
	if err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
		return "", func() {}
	}

	path = filepath.Join(dir, "debug.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
		return "", func() {}
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})))
	slog.Debug("debug mode enabled", "version", Version)
	return path, func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "hideTop: closing debug log: %v\n", err)
		}
	}
}
