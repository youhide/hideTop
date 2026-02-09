package gpu

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func collectThermal(ctx context.Context) (ThermalState, bool) {
	if !supported() {
		return ThermalNominal, false
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "pmset", "-g", "therm").Output()
	if err != nil {
		return ThermalNominal, false
	}
	return parseThermal(out)
}

var thermalRe = regexp.MustCompile(`(?i)thermal\s+pressure\s+is\s+(\w+)`)
var speedLimitRe = regexp.MustCompile(`(?i)CPU_Speed_Limit\s*=\s*(\d+)`)

func parseThermal(data []byte) (ThermalState, bool) {
	s := string(data)

	if m := thermalRe.FindStringSubmatch(s); m != nil {
		switch strings.ToLower(m[1]) {
		case "moderate", "fair":
			return ThermalFair, true
		case "heavy", "serious":
			return ThermalSerious, true
		case "critical", "trapping":
			return ThermalCritical, true
		default:
			return ThermalFair, true
		}
	}

	if m := speedLimitRe.FindStringSubmatch(s); m != nil {
		if m[1] != "100" {
			return ThermalFair, true
		}
		return ThermalNominal, true
	}

	if strings.Contains(strings.ToLower(s), "no thermal warning") {
		return ThermalNominal, true
	}

	return ThermalNominal, false
}
