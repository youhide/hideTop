package app

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/youhide/hideTop/internal/metrics"
	"github.com/youhide/hideTop/internal/ui"
)

func (m Model) collectionTimeout() time.Duration {
	timeout := m.cfg.RefreshInterval * 2
	if timeout < time.Second {
		timeout = time.Second
	}
	if timeout > 5*time.Second {
		timeout = 5 * time.Second
	}
	return timeout
}

func (m Model) processSampleEvery() time.Duration {
	sampleEvery := 2 * time.Second
	if m.cfg.RefreshInterval > sampleEvery {
		return m.cfg.RefreshInterval
	}
	return sampleEvery
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func collectSnapshot(ctx context.Context, sortBy metrics.SortField, previous metrics.Snapshot, processSampleEvery time.Duration, procLimit int, opts metrics.CollectOptions) tea.Cmd {
	return func() tea.Msg {
		snap := metrics.Collect(ctx, 200*time.Millisecond, sortBy, procLimit, processSampleEvery, previous, opts)
		return snapshotMsg(snap)
	}
}

// collectConnections gathers listening ports and active connections in the
// background. On failure it returns an unavailable result so the previous data
// is kept by the update loop.
func collectConnections() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		conns, err := metrics.CollectConnections(ctx)
		if err != nil {
			return connectionsMsg(metrics.NetConnections{})
		}
		return connectionsMsg(conns)
	}
}

func appendHistory(h []float64, v float64) []float64 {
	h = append(h, v)
	if len(h) > historySize {
		h = h[len(h)-historySize:]
	}
	return h
}

func fetchProcessDetail(pid int32, procs []metrics.ProcessInfo) tea.Cmd {
	return func() tea.Msg {
		// Find base info from snapshot
		var base metrics.ProcessInfo
		for _, p := range procs {
			if p.PID == pid {
				base = p
				break
			}
		}

		detail := ui.ProcessDetail{ProcessInfo: base}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		proc, err := process.NewProcessWithContext(ctx, pid)
		if err != nil {
			return processDetailMsg{detail: detail, err: fmt.Errorf("process %d no longer available", pid)}
		}

		if cmd, err := proc.CmdlineWithContext(ctx); err == nil {
			detail.Cmdline = cmd
		}
		if fds, err := proc.NumFDsWithContext(ctx); err == nil {
			detail.NumFDs = fds
		}
		if mem, err := proc.MemoryInfoWithContext(ctx); err == nil && mem != nil {
			detail.RSS = mem.RSS
			detail.VMS = mem.VMS
		}
		if ct, err := proc.CreateTimeWithContext(ctx); err == nil {
			detail.CreateTime = ct
		}

		return processDetailMsg{detail: detail}
	}
}
