package gpu

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Stats holds GPU metrics. When Available is false the other fields
// are meaningless and the UI must not render a GPU panel.
type Stats struct {
	Available    bool
	Utilization  float64
	FrequencyMHz int
}

// supported reports whether the current platform can provide GPU metrics.
func supported() bool {
	return runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"
}

var (
	capOnce      sync.Once
	capAvailable bool
)

func checkCapability() bool {
	if !supported() {
		return false
	}
	if _, err := exec.LookPath("ioreg"); err != nil {
		return false
	}
	return true
}

func isAvailable() bool {
	capOnce.Do(func() {
		capAvailable = checkCapability()
	})
	return capAvailable
}

// Collect gathers GPU metrics. It is safe to call from any platform;
// on unsupported systems it immediately returns Stats{Available: false}.
func Collect(ctx context.Context) Stats {
	if !isAvailable() {
		return Stats{}
	}

	s := Stats{Available: true}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if util, ok := collectUtilization(ctx); ok {
			s.Utilization = util
		}
	}()

	go func() {
		defer wg.Done()
		if freq, ok := collectFrequency(ctx); ok {
			s.FrequencyMHz = freq
		}
	}()

	wg.Wait()
	return s
}

func collectUtilization(ctx context.Context) (float64, bool) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ioreg", "-r", "-d", "1", "-c", "AGXAccelerator").Output()
	if err != nil {
		return 0, false
	}

	return parseUtilization(out)
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

func collectFrequency(ctx context.Context) (int, bool) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ioreg", "-r", "-d", "1", "-c", "AGXAccelerator").Output()
	if err != nil {
		return 0, false
	}

	return parseFrequency(out)
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
