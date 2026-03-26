package gpu

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// AMDBackend provides GPU metrics for AMD GPUs on Linux via sysfs.
type AMDBackend struct {
	once      sync.Once
	available bool
	cardPath  string // e.g. /sys/class/drm/card0/device
}

func (b *AMDBackend) Supported() bool {
	b.once.Do(func() {
		b.cardPath, b.available = findAMDGPU()
	})
	return b.available
}

func (b *AMDBackend) Collect(ctx context.Context, cpuTotal float64) Stats {
	if !b.Supported() {
		return Stats{}
	}

	s := Stats{Available: true}

	// GPU utilization: /sys/class/drm/card*/device/gpu_busy_percent
	if val, ok := readSysfsInt(filepath.Join(b.cardPath, "gpu_busy_percent")); ok {
		s.Utilization = float64(val)
	}

	// GPU temperature: look in hwmon subdirectory
	if temp, ok := readAMDTemp(b.cardPath); ok {
		s.Temperature = temp
	}

	// GPU frequency (sclk): /sys/class/drm/card*/device/pp_dpm_sclk
	if freq, ok := readAMDFrequency(b.cardPath); ok {
		s.FrequencyMHz = freq
	}

	// VRAM usage
	vramUsed, ok1 := readSysfsInt(filepath.Join(b.cardPath, "mem_info_vram_used"))
	vramTotal, ok2 := readSysfsInt(filepath.Join(b.cardPath, "mem_info_vram_total"))
	if ok1 && ok2 && vramTotal > 0 {
		s.MemoryUsedMB = float64(vramUsed) / (1024 * 1024)
		s.MemoryTotalMB = float64(vramTotal) / (1024 * 1024)
		s.Engines = append(s.Engines, EngineStats{
			Name:        "VRAM",
			Utilization: float64(vramUsed) / float64(vramTotal) * 100,
		})
	}

	// GPU name from marketing name
	if name, err := os.ReadFile(filepath.Join(b.cardPath, "product_name")); err == nil {
		s.Name = strings.TrimSpace(string(name))
	}

	s.Energy = ComputeEnergyImpact(cpuTotal, s.Utilization, true, s.Thermal)
	return s
}

// findAMDGPU scans /sys/class/drm/ for an AMD GPU card.
func findAMDGPU() (string, bool) {
	matches, err := filepath.Glob("/sys/class/drm/card[0-9]*/device/vendor")
	if err != nil {
		return "", false
	}
	for _, vendorPath := range matches {
		data, err := os.ReadFile(vendorPath)
		if err != nil {
			continue
		}
		vendor := strings.TrimSpace(string(data))
		// AMD vendor ID
		if vendor == "0x1002" {
			devicePath := filepath.Dir(vendorPath)
			// Verify gpu_busy_percent exists (amdgpu driver)
			if _, err := os.Stat(filepath.Join(devicePath, "gpu_busy_percent")); err == nil {
				return devicePath, true
			}
		}
	}
	return "", false
}

// readSysfsInt reads an integer from a sysfs file.
func readSysfsInt(path string) (int64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// readAMDTemp reads GPU temperature from hwmon under the device path.
func readAMDTemp(devicePath string) (float64, bool) {
	hwmonBase := filepath.Join(devicePath, "hwmon")
	entries, err := os.ReadDir(hwmonBase)
	if err != nil {
		return 0, false
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "hwmon") {
			continue
		}
		tempPath := filepath.Join(hwmonBase, entry.Name(), "temp1_input")
		if val, ok := readSysfsInt(tempPath); ok {
			// sysfs reports temperature in millidegrees Celsius
			return float64(val) / 1000.0, true
		}
	}
	return 0, false
}

// readAMDFrequency reads current GPU clock from pp_dpm_sclk.
// The active frequency line is marked with *.
func readAMDFrequency(devicePath string) (int, bool) {
	data, err := os.ReadFile(filepath.Join(devicePath, "pp_dpm_sclk"))
	if err != nil {
		return 0, false
	}
	// Format: "0: 300Mhz *\n1: 800Mhz\n2: 1800Mhz"
	// The active one has "*" at the end
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, "*") {
			continue
		}
		// Extract MHz value
		parts := strings.Fields(line)
		for _, part := range parts {
			lower := strings.ToLower(part)
			if strings.HasSuffix(lower, "mhz") {
				numStr := strings.TrimSuffix(lower, "mhz")
				if freq, err := strconv.Atoi(numStr); err == nil {
					return freq, true
				}
			}
		}
	}
	return 0, false
}

// FormatVRAM formats VRAM usage for display.
func FormatVRAM(usedMB, totalMB float64) string {
	if totalMB <= 0 {
		return ""
	}
	if totalMB >= 1024 {
		return fmt.Sprintf("%.1f / %.1f GiB", usedMB/1024, totalMB/1024)
	}
	return fmt.Sprintf("%.0f / %.0f MiB", usedMB, totalMB)
}
