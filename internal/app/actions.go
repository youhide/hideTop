package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m Model) killSelectedProcess(sig killSignal) string {
	if m.pidToKill <= 0 {
		return ""
	}
	err := killProcess(int(m.pidToKill), sig)
	if err != nil {
		return fmt.Sprintf("kill %d: %v", m.pidToKill, err)
	}
	return fmt.Sprintf("sent signal %d to PID %d", sig, m.pidToKill)
}

func (m Model) exportSnapshot() string {
	basename := fmt.Sprintf("hideTop_%s.json", m.snap.CollectedAt.Format("20060102_150405"))

	// Destination: --export-dir / config export_dir, falling back to home.
	dir := strings.TrimSpace(m.cfg.ExportDir)
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = home
	} else if strings.HasPrefix(dir, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, strings.TrimPrefix(dir, "~"))
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Sprintf("export error: %v", err)
	}
	filename := filepath.Join(dir, basename)
	data, err := json.MarshalIndent(m.snap, "", "  ")
	if err != nil {
		return fmt.Sprintf("export error: %v", err)
	}
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		return fmt.Sprintf("export error: %v", err)
	}
	return fmt.Sprintf("exported to %s", filename)
}
