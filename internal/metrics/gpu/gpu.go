package gpu

import (
	"bytes"
	"context"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Stats holds GPU metrics. When Available is false the other fields
// are meaningless and the UI must not render a GPU panel.
type Stats struct {
	Available     bool
	Utilization   float64
	FrequencyMHz  int
	CoreCount     int
	Engines       []EngineStats
	Thermal       ThermalState
	ThermalOK     bool
	Energy        EnergyImpact
	Temperature   float64 // GPU temperature in Celsius (0 if unavailable)
	Name          string  // GPU model name (e.g. "NVIDIA GeForce RTX 4090")
	MemoryUsedMB  float64 // VRAM used in MiB
	MemoryTotalMB float64 // VRAM total in MiB
}

// EngineStats represents a single GPU engine's utilization.
type EngineStats struct {
	Name        string
	Utilization float64 // 0-100
}

// ThermalState represents the system thermal pressure level.
type ThermalState int

const (
	ThermalNominal  ThermalState = iota // normal
	ThermalFair                         // moderate pressure
	ThermalSerious                      // heavy pressure
	ThermalCritical                     // critical / throttling
)

// String returns a human-readable label for the thermal state.
func (t ThermalState) String() string {
	switch t {
	case ThermalFair:
		return "fair"
	case ThermalSerious:
		return "serious"
	case ThermalCritical:
		return "critical"
	default:
		return "nominal"
	}
}

// EnergyImpact holds an approximate energy impact score.
// This is a heuristic inspired by macOS Activity Monitor,
// NOT an official Apple metric.
type EnergyImpact struct {
	Score     float64 // 0-100
	Available bool
}

// supported reports whether the current platform can provide GPU metrics
// via any backend.
func supported() bool {
	return runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"
}

var (
	backendOnce   sync.Once
	activeBackend Backend
)

// initBackend discovers the first available GPU backend.
// Order: Apple Silicon → NVIDIA → AMD → none.
func initBackend() {
	backends := []Backend{
		&AppleBackend{},
		&NvidiaBackend{},
		&AMDBackend{},
	}
	for _, b := range backends {
		if b.Supported() {
			activeBackend = b
			return
		}
	}
}

// Collect gathers GPU metrics. It is safe to call from any platform;
// on unsupported systems it immediately returns Stats{Available: false}.
func Collect(ctx context.Context, cpuTotal float64) Stats {
	backendOnce.Do(initBackend)
	if activeBackend == nil {
		return Stats{}
	}
	return activeBackend.Collect(ctx, cpuTotal)
}

// BackendName returns the name of the active GPU backend, or "none".
func BackendName() string {
	backendOnce.Do(initBackend)
	if activeBackend == nil {
		return "none"
	}
	switch activeBackend.(type) {
	case *AppleBackend:
		return "apple"
	case *NvidiaBackend:
		return "nvidia"
	case *AMDBackend:
		return "amd"
	default:
		return "unknown"
	}
}

var utilRe = regexp.MustCompile(`(?i)(?:device utilization|gpu[- ]utilization)[^"]*"?\s*(?:%\s*)?=\s*(\d+)`)

func parseUtilization(data []byte) (float64, bool) {
	m := utilRe.FindSubmatch(data)
	if m == nil {
		return 0, false
	}
	v, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

var coreCountRe = regexp.MustCompile(`"gpu-core-count"\s*=\s*(\d+)`)

func parseCoreCount(data []byte) (int, bool) {
	m := coreCountRe.FindSubmatch(data)
	if m == nil {
		return 0, false
	}
	v, err := strconv.Atoi(string(m[1]))
	if err != nil {
		return 0, false
	}
	return v, true
}

var freqPatterns = []string{
	"gpu-core-clock",
	"gpuCoreClockMHz",
	"gpu-freq",
	"GPUClockFrequency",
}

func parseFrequency(data []byte) (int, bool) {
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		s := string(line)
		lower := strings.ToLower(s)
		for _, pat := range freqPatterns {
			if !strings.Contains(lower, strings.ToLower(pat)) {
				continue
			}
			re := regexp.MustCompile(`=\s*(\d+)`)
			m := re.FindStringSubmatch(s)
			if m == nil {
				continue
			}
			v, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			if v > 100000 {
				v = v / 1000000
			}
			return v, true
		}
	}
	return 0, false
}
