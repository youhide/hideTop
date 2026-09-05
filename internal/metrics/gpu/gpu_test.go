package gpu

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestParseFrequency(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
		ok   bool
	}{
		{"MHz value", `"gpu-core-clock" = 800`, 800, true},
		{"kHz value", `"gpu-core-clock" = 500000`, 500, true},
		{"Hz value", `"gpu-core-clock" = 1300000000`, 1300, true},
		{"kHz just above the old threshold", `"gpu-freq" = 150000`, 150, true},
		{"case insensitive key", `"GPUCLOCKFREQUENCY" = 1200`, 1200, true},
		{"no match", `"something-else" = 42`, 0, false},
		{"no value", `"gpu-core-clock" = abc`, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseFrequency([]byte(tt.in))
			if got != tt.want || ok != tt.ok {
				t.Errorf("parseFrequency(%q) = (%d, %v), want (%d, %v)", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestComputeEnergyImpactClamps(t *testing.T) {
	tests := []struct {
		name          string
		cpu, gpu      float64
		thermal       ThermalState
		wantAvailable bool
		wantInRange   bool
	}{
		{"idle", 0, 0, ThermalNominal, true, true},
		{"pegged", 100, 100, ThermalNominal, true, true},
		{"over range inputs are clamped", 500, 500, ThermalCritical, true, true},
		{"negative inputs are clamped", -50, -50, ThermalNominal, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ComputeEnergyImpact(tt.cpu, tt.gpu, true, tt.thermal)
			if e.Available != tt.wantAvailable {
				t.Errorf("Available = %v, want %v", e.Available, tt.wantAvailable)
			}
			if e.Score < 0 || e.Score > 100 {
				t.Errorf("Score = %v, want within [0,100]", e.Score)
			}
		})
	}
}

// TestAppleSoCNameIsCached pins that the SoC lookup is a one-shot: it shells
// out to sysctl, and the value cannot change while the process runs.
func TestAppleSoCNameIsCached(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sysctl machdep.cpu.brand_string is darwin-only")
	}
	first := appleSoCName()
	if first == "" {
		t.Skip("SoC name unavailable on this machine")
	}
	if second := appleSoCName(); second != first {
		t.Errorf("appleSoCName() returned %q then %q; it must be stable", first, second)
	}
}

// TestAppleBackendLabelsTheGPU pins the fix for an unnamed GPU panel:
// AGXAccelerator exposes no product name, so the panel had no label at all.
func TestAppleBackendLabelsTheGPU(t *testing.T) {
	var b AppleBackend
	if !b.Supported() {
		t.Skip("not an Apple Silicon Mac")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s := b.Collect(ctx, 0)
	if !s.Available {
		// Virtualised Macs (the CI runners among them) have no AGXAccelerator.
		// Reporting unavailable is the correct outcome there; what must not
		// happen is claiming availability with nothing behind it.
		if s.Name != "" || s.CoreCount != 0 || s.Utilization != 0 || len(s.Engines) != 0 {
			t.Errorf("unavailable GPU returned populated stats: %+v", s)
		}
		t.Skip("no Apple GPU on this machine")
	}
	if s.Name == "" {
		t.Error("Apple GPU is available but has no name")
	}
	if s.CoreCount <= 0 {
		t.Errorf("CoreCount = %d, want a positive core count", s.CoreCount)
	}
}
