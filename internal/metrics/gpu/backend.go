package gpu

import "context"

// Backend is the interface that GPU metric providers must implement.
// Collect returns GPU stats; if the backend is unavailable it returns
// Stats{Available: false}.
type Backend interface {
	// Supported reports whether this backend can provide GPU metrics
	// on the current platform.
	Supported() bool

	// Collect gathers GPU metrics. cpuTotal is passed for energy
	// impact calculation.
	Collect(ctx context.Context, cpuTotal float64) Stats
}
