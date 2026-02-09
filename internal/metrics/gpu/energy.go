package gpu

// computeEnergyImpact calculates a lightweight approximation of system
// energy impact, inspired by macOS Activity Monitor.
//
// This is NOT an official Apple metric. It is a heuristic based on:
//   - CPU utilization (dominant factor)
//   - GPU utilization (when available)
//   - Thermal state (elevated states imply higher power draw)
//
// The score is normalized to roughly 0-100, where:
//   - 0 means idle
//   - 50 means moderate workload
//   - 100 means sustained heavy load
//
// The formula intentionally over-weights CPU because it is available
// on all platforms and is the most reliable signal.
func computeEnergyImpact(cpuTotal float64, gpuUtil float64, gpuAvailable bool, thermal ThermalState) EnergyImpact {
	score := cpuTotal * 0.70

	if gpuAvailable {
		score += gpuUtil * 0.25
	}

	switch thermal {
	case ThermalFair:
		score += 2
	case ThermalSerious:
		score += 4
	case ThermalCritical:
		score += 5
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return EnergyImpact{
		Score:     score,
		Available: true,
	}
}
