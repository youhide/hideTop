# hideTop

A modern terminal-based system monitor written in Go, inspired by `top`, `htop`, and `gotop`.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **CPU** — total + per-core utilisation bars with core count, colour-coded by load, sparkline history
- **GPU** — total + per-engine utilisation, core count, frequency, thermal pressure indicator, and heuristic energy impact score. Auto-detected at runtime; hidden on unsupported hardware. Supports **Apple Silicon** (ioreg), **NVIDIA** (nvidia-smi), and **AMD** (sysfs)
- **Memory** — used / total / available GiB with bar; conditional swap bar when swap is active; sparkline history
- **Load Average** — 1 / 5 / 15 minute
- **Temperature** — up to 6 sensors in a 2-column grid, auto-detects CPU/GPU temps, colour-coded by threshold (green < 60°C, yellow 60–80°C, red > 80°C). Disable with `--no-temp`
- **Network** — total in/out throughput (bytes/s), per-interface breakdown (up to 4 active interfaces)
- **Disk** — total read/write throughput (bytes/s), root filesystem usage
- **Ports & connections** — listening ports (proto, port, process) shown in the Network panel; press `n` for a full-screen view of all listening ports plus active connections (local/remote/state/process). Collected on a throttled cadence via `lsof`/sysfs, no sudo. Disable with `--no-ports`
- **Battery** — percentage and charging status in the header bar (macOS via `pmset`, Linux via sysfs)
- **Processes** — sortable by CPU, memory, or PID with visual sort indicators (▲/▼); columns for PID, state (R/S/Z/T), user, name, threads, CPU%, MEM%; PID-based row selection; incremental search by name, PID, or username; tree view; system process filter; process detail panel (Enter); kill / force kill with confirmation
- **Themes** — 5 built-in themes: `dark` (default), `light`, `dracula`, `nord`, `monokai`
- **Responsive layout** — two-column layout at ≥ 110 cols, single-column stacked on narrower terminals
- **Mouse support** — scroll wheel to navigate the process list, click a row to select; click a PID/CPU%/MEM% column header to sort by it; scrolling over the Temperature or Network panel scrolls that panel to reveal hidden sensors/interfaces
- **Export** — snapshot to JSON with `e`
- **Configurable** — CLI flags and `~/.config/hideTop/config.json`

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| `↑` `↓` / `j` `k` | Move process selection |
| `PgUp` `PgDn` | Jump one page up / down |
| `Home` `End` / `g` `G` | Jump to first / last process |
| `/` | Start incremental search (name, PID, or user), `Esc` to cancel |
| `Enter` | Open process detail panel |
| `c` | Sort by CPU% (descending) |
| `m` | Sort by MEM% (descending) |
| `p` | Sort by PID (ascending) |
| `Space` | Pause / resume metrics auto-refresh |
| `t` | Toggle tree view |
| `s` | Toggle system process filter |
| `z` | Reset Temperature / Network panel scroll |
| `n` | Open the network / ports view (`Esc` or `n` to close) |
| `x` | Kill selected process (SIGTERM, asks for confirmation) |
| `K` | Force kill selected process (SIGKILL, asks for confirmation) |
| `+` / `=` | Increase refresh interval (+250ms) |
| `-` / `_` | Decrease refresh interval (-250ms) |
| `e` | Export snapshot to JSON |
| `?` | Toggle help overlay |
| `Esc` | Close help / detail / cancel search |
| `q` / `Ctrl+C` | Quit |

## Installation

```bash
brew tap youhide/homebrew-youhide
brew install hidetop
```

> Recent Homebrew versions require you to trust third-party taps before the
> first install. If you see `Refusing to load formula ... from untrusted tap`,
> run `brew trust youhide/youhide` once and re-run the install.

## Quick start

```bash
go build -o hideTop ./src/
./hideTop                     # default 1s refresh
./hideTop --interval 500ms    # faster refresh
./hideTop --theme dracula     # use dracula theme
./hideTop --no-gpu --no-temp  # disable GPU and temperature panels
./hideTop --version           # print version and exit
# local build with git tag in --version:
go build -ldflags "-X main.Version=$(git describe --tags --always --dirty)" -o hideTop ./src/
```

## Configuration

CLI flags take precedence over the config file.

| Flag | Default | Description |
|------|---------|-------------|
| `--interval` | `1s` | Metrics refresh interval (min 100ms) |
| `--theme` | `dark` | Colour theme (`dark`, `light`, `dracula`, `nord`, `monokai`) |
| `--no-gpu` | `false` | Disable GPU metrics |
| `--no-temp` | `false` | Disable temperature metrics |
| `--no-ports` | `false` | Disable listening ports / connections collection |
| `--export-dir` | home | Directory for JSON snapshot exports (`e`); `~` is expanded |
| `--debug` | `false` | Enable debug logging to stderr |
| `--version` / `-v` | — | Print version and exit |

### Config file

`~/.config/hideTop/config.json`

```json
{
  "interval": "1s",
  "theme": "dracula",
  "no_gpu": false,
  "no_temp": false,
  "no_ports": false,
  "debug": false,
  "export_dir": "~/Desktop",
  "filter_users": ["root", "_windowserver", "nobody"]
}
```

The `filter_users` array controls which usernames are hidden when the system process filter (`s`) is active. Defaults to `["root", "_windowserver", "nobody"]` if not set.

## Architecture

| Layer | Package | Responsibility |
|-------|---------|---------------|
| **Entry** | `src` | Parse config, wire up Bubble Tea, enable mouse & alt screen |
| **App** | `internal/app` | Bubble Tea Model / Update / View, owns the event loop |
| **Metrics** | `internal/metrics` | CPU, memory, load, processes, temperature, network, disk, battery via gopsutil; concurrent collection with graceful degradation |
| **GPU** | `internal/metrics/gpu` | Pluggable backends: Apple Silicon (`ioreg`), NVIDIA (`nvidia-smi`), AMD (sysfs). No sudo required |
| **UI** | `internal/ui` | Pure functions: data in → styled string out. Themes, sparklines, process table, detail overlay |
| **Config** | `internal/config` | CLI flags + `~/.config/hideTop/config.json` |

Key design decisions:
- **No global mutable state** — all state lives in the Bubble Tea `Model`.
- **Async collection** — metrics are gathered in a `tea.Cmd` goroutine, so the UI never blocks.
- **Concurrent collectors** — CPU, memory, load, network, disk, battery, temperature, and processes run in parallel via `sync.WaitGroup`; GPU runs sequentially after (needs CPU total for energy calculation).
- **Graceful degradation** — if a collector fails or times out, the previous snapshot is used and a `stale` indicator appears in the header.
- **Pure rendering** — UI functions take data + width and return strings. No side effects, easy to test.
- **Runtime detection** — GPU support is detected via `runtime.GOOS` + `runtime.GOARCH` and cached with `sync.Once`. No build tags needed; the binary works on any platform.
- **No sudo** — all data sources (`ioreg`, `pmset`, `nvidia-smi`, sysfs, gopsutil) work without elevated privileges.
- **PID-based selection** — process selection tracks by PID, surviving refresh cycles, re-sorts, and search filters. Falls back to same visual position when a process disappears.
- **Responsive layout** — panels pair in two columns when the terminal is ≥ 110 columns wide, with matched heights.

## Requirements

- Go 1.25+
- macOS or Linux (GPU panel: Apple Silicon, NVIDIA with nvidia-smi, or AMD with sysfs)
