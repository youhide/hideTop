package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/youhide/hideTop/internal/config"
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

// saveSettings persists session changes the user made with the keyboard.
// Errors are recorded on the model rather than printed: stderr is invisible
// while the alt-screen is still up, so a message written here would be lost.
func (m *Model) saveSettings() {
	if m.intervalChanged {
		if err := config.SaveInterval(m.cfg.RefreshInterval); err != nil {
			m.saveErr = err
		}
	}
	if m.panelsChanged {
		if err := config.SaveHiddenPanels(m.hiddenPanelList()); err != nil {
			m.saveErr = err
		}
	}
}

// confirmTitle and confirmBody describe the pending kill for the modal.
func (m Model) confirmTitle() string {
	if m.confirmKill == signalKill {
		return "Force kill process?"
	}
	return "Terminate process?"
}

func (m Model) confirmBody() string {
	name := "process"
	for _, p := range m.snap.Processes {
		if p.PID == m.pidToKill {
			name = p.Name
			break
		}
	}
	sig := "SIGTERM"
	if m.confirmKill == signalKill {
		sig = "SIGKILL"
	}
	return fmt.Sprintf("Send %s to PID %d (%s)?", sig, m.pidToKill, name)
}
