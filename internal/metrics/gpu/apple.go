package gpu

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// AppleBackend provides GPU metrics for Apple Silicon Macs via ioreg.
type AppleBackend struct {
	once      sync.Once
	available bool
}

func (b *AppleBackend) Supported() bool {
	b.once.Do(func() {
		b.available = runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" && hasCommand("ioreg")
	})
	return b.available
}

func (b *AppleBackend) Collect(ctx context.Context, cpuTotal float64) Stats {
	if !b.Supported() {
		return Stats{}
	}

	s := Stats{Available: true}

	var ioregData []byte
	func() {
		ioCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ioCtx, "ioreg", "-r", "-c", "AGXAccelerator").Output()
		if err == nil {
			ioregData = out
		}
	}()

	if len(ioregData) > 0 {
		if util, ok := parseUtilization(ioregData); ok {
			s.Utilization = util
		}
		if freq, ok := parseFrequency(ioregData); ok {
			s.FrequencyMHz = freq
		}
		if engines := parseEnginesFromIOReg(ioregData); len(engines) > 0 {
			s.Engines = engines
		}
		if cores, ok := parseCoreCount(ioregData); ok {
			s.CoreCount = cores
		}
	}

	if state, ok := collectThermal(ctx); ok {
		s.Thermal = state
		s.ThermalOK = true
	}

	s.Energy = computeEnergyImpact(cpuTotal, s.Utilization, true, s.Thermal)
	return s
}

// hasCommand checks if a command exists in PATH.
func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// parseGPUTemp extracts GPU temperature from ioreg data if present.
func parseGPUTemp(data []byte) (float64, bool) {
	// Some Apple Silicon ioreg outputs include temperature data under
	// "GPU Temperature" or similar keys. This is best-effort.
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		s := string(line)
		_ = s
		// Apple Silicon typically does not expose GPU temp through ioreg
		// Temperature comes from SMC/IOKit sensors (handled by gopsutil)
	}
	return 0, false
}
