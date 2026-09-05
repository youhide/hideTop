package metrics

import (
	"context"
	"testing"
	"time"
)

func BenchmarkCollectProcesses(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	b.ResetTimer()
	for b.Loop() {
		if _, err := CollectProcesses(ctx, SortByCPU, 50); err != nil {
			b.Fatal(err)
		}
	}
}
