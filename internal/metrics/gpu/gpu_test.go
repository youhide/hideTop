package gpu

import "testing"

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
