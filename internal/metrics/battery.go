package metrics

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// BatteryStats holds battery information.
type BatteryStats struct {
	Available bool
	Percent   float64 // 0-100
	Charging  bool
	Status    string // "Charging", "Discharging", "Full", "Not charging"
}

// CollectBattery gathers battery information.
// Works on Linux (sysfs) and macOS (pmset).
func CollectBattery(ctx context.Context) (BatteryStats, error) {
	switch runtime.GOOS {
	case "linux":
		return collectBatteryLinux()
	case "darwin":
		return collectBatteryDarwin(ctx)
	default:
		return BatteryStats{}, nil
	}
}

// collectBatteryLinux reads battery info from /sys/class/power_supply/.
func collectBatteryLinux() (BatteryStats, error) {
	base := "/sys/class/power_supply"
	entries, err := os.ReadDir(base)
	if err != nil {
		return BatteryStats{}, nil
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "BAT") {
			continue
		}

		batPath := filepath.Join(base, name)
		stats := BatteryStats{Available: true}

		// Read capacity
		if data, err := os.ReadFile(filepath.Join(batPath, "capacity")); err == nil {
			if v, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
				stats.Percent = v
			}
		}

		// Read status
		if data, err := os.ReadFile(filepath.Join(batPath, "status")); err == nil {
			stats.Status = strings.TrimSpace(string(data))
			stats.Charging = strings.EqualFold(stats.Status, "Charging")
		}

		return stats, nil
	}

	return BatteryStats{}, nil
}

// collectBatteryDarwin reads battery info via pmset on macOS.
func collectBatteryDarwin(_ context.Context) (BatteryStats, error) {
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return BatteryStats{}, nil
	}

	s := string(out)
	stats := BatteryStats{}
	onAC := strings.Contains(s, "'AC Power'")

	// Parse percentage: "InternalBattery-0 (id=...)	85%; charging; ..."
	for _, line := range strings.Split(s, "\n") {
		if !strings.Contains(line, "InternalBattery") {
			continue
		}
		stats.Available = true
		// Extract percentage
		if idx := strings.Index(line, "%"); idx > 0 {
			start := idx - 1
			for start >= 0 && (line[start] >= '0' && line[start] <= '9') {
				start--
			}
			start++
			if start < idx {
				if v, err := strconv.ParseFloat(line[start:idx], 64); err == nil {
					stats.Percent = v
				}
			}
		}
		// Extract status
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "charging") && !strings.Contains(lower, "not charging") && !strings.Contains(lower, "discharging"):
			stats.Status = "Charging"
			stats.Charging = true
		case strings.Contains(lower, "discharging"):
			stats.Status = "Discharging"
			stats.Charging = false
		case strings.Contains(lower, "not charging"):
			stats.Status = "Not charging"
			stats.Charging = false
		case strings.Contains(lower, "charged"):
			stats.Status = "Full"
			stats.Charging = false
		default:
			// No explicit status found; infer from power source
			stats.Charging = onAC
		}
		break
	}

	return stats, nil
}
