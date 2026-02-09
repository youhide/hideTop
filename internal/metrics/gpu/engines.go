package gpu

import (
	"regexp"
	"strconv"
	"strings"
)

// ioregEngineRe matches ioreg PerformanceStatistics entries like:
//
//	"Tiler Utilization %" = 4
//	"Renderer Utilization %" = 5
//
// Captures the engine name and the integer percentage.
var ioregEngineRe = regexp.MustCompile(`"(\w+)\s+Utilization\s+%"\s*=\s*(\d+)`)

// parseEnginesFromIOReg extracts per-engine GPU utilization from ioreg output.
// This works without sudo on Apple Silicon — the data is in PerformanceStatistics.
// "Device Utilization %" is excluded since it represents the total.
func parseEnginesFromIOReg(data []byte) []EngineStats {
	matches := ioregEngineRe.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return nil
	}

	var engines []EngineStats
	for _, m := range matches {
		name := string(m[1])
		// Skip "Device" — that's the total, not an engine.
		if strings.EqualFold(name, "Device") {
			continue
		}
		v, err := strconv.ParseFloat(string(m[2]), 64)
		if err != nil {
			continue
		}
		engines = append(engines, EngineStats{
			Name:        name,
			Utilization: v,
		})
	}
	return engines
}
