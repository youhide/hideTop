# hideTop

A modern terminal-based system monitor written in Go, inspired by `top`, `htop`, and `gotop`.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **CPU** — total + per-core utilisation bars with core count, colour-coded by load
- **GPU** *(Apple Silicon)* — total + per-engine (Tiler / Renderer) utilisation, core count, frequency, thermal pressure indicator, and heuristic energy impact score. Auto-detected at runtime; hidden on unsupported hardware
- **Memory** — used / total / available GiB with bar; conditional swap bar when swap is active
- **Load Average** — 1 / 5 / 15 minute
- **Processes** — sortable by CPU, memory, or PID with visual sort indicators (▲/▼); PID-based row selection (↑↓ / j/k); incremental search (`/`); auto-scrolling viewport
- **Refresh control** — `+`/`-` to adjust interval with visual flash feedback
- **Configurable** — `--interval` flag (default 1s, minimum 100ms)

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| `↑` `↓` / `j` `k` | Move process selection |
| `/` | Start incremental search, `Esc` to cancel |
| `c` | Sort by CPU% (descending) |
| `m` | Sort by MEM% (descending) |
| `p` | Sort by PID (ascending) |
| `+` `-` | Increase / decrease refresh interval |
| `q` / `Ctrl+C` | Quit |

## Quick start

```bash
go build -o hideTop ./src/
./hideTop                     # default 1s refresh
./hideTop --interval 500ms    # faster refresh
```

## Project structure

```
hideTop/
├── src/                      # Entry point
│   └── main.go
├── internal/
│   ├── app/                  # Bubble Tea model, update loop, view
│   │   └── model.go
│   ├── config/               # CLI flags & configuration
│   │   └── config.go
│   ├── metrics/              # System metrics collectors
│   │   ├── types.go          # Shared data types (Snapshot, CPUStats, …)
│   │   ├── collector.go      # Concurrent aggregation of all metrics
│   │   ├── cpu.go
│   │   ├── memory.go
│   │   ├── processes.go
│   │   └── gpu/              # Apple Silicon GPU metrics (ioreg-based)
│   │       ├── gpu.go        # Main collector, types & capability detection
│   │       ├── engines.go    # Per-engine utilisation parser (Tiler, Renderer)
│   │       ├── thermal.go    # Thermal pressure via pmset
│   │       └── energy.go     # Heuristic energy impact calculator
│   └── ui/                   # Pure rendering functions (Lip Gloss)
│       ├── styles.go         # Colour palette & shared styles
│       ├── cpu.go
│       ├── gpu.go
│       ├── memory.go
│       ├── processes.go
│       └── help.go
├── go.mod
├── go.sum
└── README.md
```

## Architecture

| Layer | Package | Responsibility |
|-------|---------|---------------|
| **Entry** | `src` | Parse config, wire up Bubble Tea |
| **App** | `internal/app` | Bubble Tea Model / Update / View, owns the event loop |
| **Metrics** | `internal/metrics` | CPU, memory, load, processes via gopsutil; concurrent collection |
| **GPU** | `internal/metrics/gpu` | Apple Silicon GPU via `ioreg` + `pmset` (no sudo required) |
| **UI** | `internal/ui` | Pure functions: data in → styled string out |
| **Config** | `internal/config` | CLI flags, future file-based config |

Key design decisions:
- **No global mutable state** — all state lives in the Bubble Tea `Model`.
- **Async collection** — metrics are gathered in a `tea.Cmd` goroutine, so the UI never blocks.
- **Concurrent collectors** — CPU, memory, load, and processes run in parallel via `sync.WaitGroup`; GPU runs sequentially after (needs CPU total for energy calculation).
- **Pure rendering** — UI functions take data + width and return strings. No side effects, easy to test.
- **Runtime detection** — GPU support is detected via `runtime.GOOS` + `runtime.GOARCH` and cached with `sync.Once`. No build tags needed; the binary works on any platform.
- **No sudo** — all data sources (`ioreg`, `pmset`, gopsutil) work without elevated privileges.
- **PID-based selection** — process selection tracks by PID, surviving refresh cycles, re-sorts, and search filters.

## Requirements

- Go 1.25+
- macOS or Linux (GPU panel requires Apple Silicon)
