package metrics

import (
	"context"
	"strings"

	"github.com/shirou/gopsutil/v4/sensors"
)

// SensorReading represents a single temperature sensor reading.
type SensorReading struct {
	Label       string
	Temperature float64 // Celsius
}

// TemperatureStats holds system temperature readings.
type TemperatureStats struct {
	Available bool
	Sensors   []SensorReading
	CPUTemp   float64 // best-effort CPU temperature (highest "core"/"cpu" sensor)
	GPUTemp   float64 // best-effort GPU temperature (highest "gpu" sensor)
}

// CollectTemperature gathers temperature readings from available sensors.
// Works on Linux (/sys/class/hwmon) and macOS (IOKit) via gopsutil.
// Returns TemperatureStats with Available=false if no sensors found.
func CollectTemperature(ctx context.Context) (TemperatureStats, error) {
	temps, err := sensors.TemperaturesWithContext(ctx)
	if err != nil {
		// gopsutil may return an error alongside partial results on some platforms.
		if len(temps) == 0 {
			return TemperatureStats{}, nil
		}
	}

	if len(temps) == 0 {
		return TemperatureStats{}, nil
	}

	stats := TemperatureStats{Available: true}
	var maxCPU, maxGPU float64

	for _, t := range temps {
		if t.Temperature <= 0 || t.Temperature > 150 {
			continue // skip invalid readings
		}

		stats.Sensors = append(stats.Sensors, SensorReading{
			Label:       normalizeSensorLabel(t.SensorKey),
			Temperature: t.Temperature,
		})

		lower := strings.ToLower(t.SensorKey)
		if isCPUSensor(lower) && t.Temperature > maxCPU {
			maxCPU = t.Temperature
		}
		if isGPUSensor(lower) && t.Temperature > maxGPU {
			maxGPU = t.Temperature
		}
	}

	if len(stats.Sensors) == 0 {
		stats.Available = false
		return stats, nil
	}

	stats.CPUTemp = maxCPU
	stats.GPUTemp = maxGPU
	return stats, nil
}

// isCPUSensor checks if a sensor key likely represents a CPU temperature.
func isCPUSensor(lower string) bool {
	cpuKeywords := []string{"core", "cpu", "coretemp", "k10temp", "tctl", "tdie", "package"}
	for _, kw := range cpuKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// isGPUSensor checks if a sensor key likely represents a GPU temperature.
func isGPUSensor(lower string) bool {
	gpuKeywords := []string{"gpu", "amdgpu", "nouveau", "nvidia", "radeon"}
	for _, kw := range gpuKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// normalizeSensorLabel cleans up sensor labels for display.
func normalizeSensorLabel(key string) string {
	// Some sensors have prefixes like "coretemp_packageid0_input"
	// Try to make them more readable.
	label := key
	for _, sep := range []string{"_input", "_temp"} {
		label = strings.TrimSuffix(label, sep)
	}
	label = strings.ReplaceAll(label, "_", " ")
	return label
}
