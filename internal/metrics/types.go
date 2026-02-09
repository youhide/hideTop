package metrics

type CPUStats struct {
	PerCore []float64
	Total   float64
}

type MemoryStats struct {
	TotalGB     float64
	UsedGB      float64
	AvailableGB float64
	Percent     float64
	SwapTotalGB float64
	SwapUsedGB  float64
	SwapPercent float64
}

type LoadAvg struct {
	Load1  float64
	Load5  float64
	Load15 float64
}

type ProcessInfo struct {
	PID        int32
	Name       string
	CPUPercent float64
	MemPercent float32
}

type Snapshot struct {
	CPU       CPUStats
	Memory    MemoryStats
	Load      LoadAvg
	Processes []ProcessInfo
	GPU       interface{} // *gpu.Stats when available; avoids import cycle
}
