# hideTop

A modern terminal-based system monitor written in Go, inspired by `top`, `htop`, and `gotop`.

## Features

- **CPU usage** — per-core bars + total, colour-coded by load
- **Memory usage** — bar with GiB breakdown
- **Load average** — 1 / 5 / 15 minute
- **Process list** — sortable by CPU, memory, or PID
- **Keyboard controls** — `c` sort CPU · `m` sort mem · `p` sort PID · `+/-` adjust refresh · `q` quit
- **Configurable refresh** via `--interval` flag

## Quick start

```bash
go build -o hideTop ./cmd/hidetop/
./hideTop                     # default 1s refresh
./hideTop --interval 500ms    # faster refresh
```

## Project structure

```
hideTop/
├── cmd/hidetop/          # Entry point
│   └── main.go
├── internal/
│   ├── app/              # Bubble Tea model, update loop, view
│   │   └── model.go
│   ├── config/           # CLI flags & configuration
│   │   └── config.go
│   ├── metrics/          # System metrics collectors (gopsutil)
│   │   ├── types.go      # Shared data types (Snapshot, CPUStats, …)
│   │   ├── cpu.go
│   │   ├── memory.go
│   │   ├── processes.go
│   │   └── collector.go  # Concurrent aggregation of all metrics
│   └── ui/               # Pure rendering functions (Lip Gloss)
│       ├── styles.go     # Colour palette & shared styles
│       ├── cpu.go
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
| **Entry** | `cmd/hidetop` | Parse config, wire up Bubble Tea |
| **App** | `internal/app` | Bubble Tea Model/Update/View, owns the event loop |
| **Metrics** | `internal/metrics` | Data collection via gopsutil, fully async |
| **UI** | `internal/ui` | Pure functions: data in → styled string out |
| **Config** | `internal/config` | CLI flags, future file-based config |

Key design decisions:
- **No global mutable state** — all state lives in the Bubble Tea `Model`.
- **Async collection** — metrics are gathered in a `tea.Cmd` goroutine, so the UI never blocks.
- **Concurrent collectors** — CPU, memory, load, and processes run in parallel via `sync.WaitGroup`.
- **Pure rendering** — UI functions take data + width and return strings. No side effects, easy to test.
- **Extensible** — adding a new widget means writing a render function in `ui/` and a collector in `metrics/`.

## Requirements

- Go 1.21+
- macOS or Linux
