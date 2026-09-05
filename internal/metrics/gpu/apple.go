package gpu

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
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

	s := Stats{Available: true, Name: appleSoCName()}

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
		// Best effort. Apple Silicon does not publish a GPU clock through
		// AGXAccelerator — its PerformanceStatistics carry utilisation and
		// memory only, and the clock lives behind the private IOReport API,
		// which needs elevated privileges this tool deliberately avoids. The
		// parse stays for any model that does expose one; the panel simply
		// omits the row when it comes back zero.
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

	s.Energy = ComputeEnergyImpact(cpuTotal, s.Utilization, true, s.Thermal)
	return s
}

// appleSoCName returns the marketing name of the SoC ("Apple M5"), which is
// also the GPU's identity on Apple Silicon: AGXAccelerator itself exposes no
// product name, so the GPU panel had no label at all. The value is fixed for
// the life of the machine, so it is read once.
var appleSoCName = sync.OnceValue(func() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sysctl", "-n", "machdep.cpu.brand_string").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
})

// hasCommand checks if a command exists in PATH.
func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
