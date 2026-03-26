package gpu

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NvidiaBackend provides GPU metrics via nvidia-smi.
type NvidiaBackend struct {
	once      sync.Once
	available bool
}

func (b *NvidiaBackend) Supported() bool {
	b.once.Do(func() {
		b.available = hasCommand("nvidia-smi")
	})
	return b.available
}

func (b *NvidiaBackend) Collect(ctx context.Context, cpuTotal float64) Stats {
	if !b.Supported() {
		return Stats{}
	}

	s := Stats{Available: true}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Query utilization, temperature, memory, power, clock, name
	out, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=utilization.gpu,temperature.gpu,memory.used,memory.total,clocks.current.graphics,gpu_name",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return Stats{}
	}

	// Parse CSV output: "85, 72, 4096, 8192, 1800, NVIDIA GeForce RTX 4090"
	line := strings.TrimSpace(string(out))
	// Handle multi-GPU: take first GPU
	if idx := strings.Index(line, "\n"); idx >= 0 {
		line = line[:idx]
	}

	fields := strings.Split(line, ",")
	if len(fields) < 4 {
		return Stats{}
	}
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}

	if util, err := strconv.ParseFloat(fields[0], 64); err == nil {
		s.Utilization = util
	}
	if temp, err := strconv.ParseFloat(fields[1], 64); err == nil {
		s.Temperature = temp
	}
	if len(fields) >= 5 {
		if freq, err := strconv.Atoi(fields[4]); err == nil {
			s.FrequencyMHz = freq
		}
	}
	if len(fields) >= 6 {
		s.Name = strings.TrimSpace(fields[5])
	}

	// Parse memory for additional context
	if len(fields) >= 4 {
		memUsed, err1 := strconv.ParseFloat(fields[2], 64)
		memTotal, err2 := strconv.ParseFloat(fields[3], 64)
		if err1 == nil && err2 == nil && memTotal > 0 {
			s.MemoryUsedMB = memUsed
			s.MemoryTotalMB = memTotal
			s.Engines = append(s.Engines, EngineStats{
				Name:        "VRAM",
				Utilization: memUsed / memTotal * 100,
			})
		}
	}

	// Also try to get encoder/decoder utilization
	encOut, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=utilization.encoder,utilization.decoder",
		"--format=csv,noheader,nounits",
	).Output()
	if err == nil {
		encLine := strings.TrimSpace(string(encOut))
		if idx := strings.Index(encLine, "\n"); idx >= 0 {
			encLine = encLine[:idx]
		}
		encFields := strings.Split(encLine, ",")
		if len(encFields) >= 2 {
			if enc, err := strconv.ParseFloat(strings.TrimSpace(encFields[0]), 64); err == nil && enc > 0 {
				s.Engines = append(s.Engines, EngineStats{Name: "Encoder", Utilization: enc})
			}
			if dec, err := strconv.ParseFloat(strings.TrimSpace(encFields[1]), 64); err == nil && dec > 0 {
				s.Engines = append(s.Engines, EngineStats{Name: "Decoder", Utilization: dec})
			}
		}
	}

	s.Energy = computeEnergyImpact(cpuTotal, s.Utilization, true, s.Thermal)
	return s
}

// nvidiaQueryMultiGPU returns stats for all NVIDIA GPUs. Currently unused
// but available for future multi-GPU support.
func nvidiaQueryMultiGPU(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=index,gpu_name",
		"--format=csv,noheader").Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi: %w", err)
	}
	var gpus []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			gpus = append(gpus, line)
		}
	}
	return gpus, nil
}
