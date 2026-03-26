package config

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	RefreshInterval time.Duration
	ShowVersion     bool
	Debug           bool
	Theme           string
	NoGPU           bool
	NoTemp          bool
	FilterUsers     []string
}

// DefaultFilterUsers is used when no custom filter is configured.
var DefaultFilterUsers = []string{"root", "_windowserver", "nobody"}

// fileConfig matches the JSON config file format.
type fileConfig struct {
	Interval    string   `json:"interval"`
	Theme       string   `json:"theme"`
	NoGPU       bool     `json:"no_gpu"`
	NoTemp      bool     `json:"no_temp"`
	Debug       bool     `json:"debug"`
	FilterUsers []string `json:"filter_users"`
}

func Parse() Config {
	interval := flag.Duration("interval", 1*time.Second,
		"metrics refresh interval (e.g. 500ms, 1s, 2s)")
	showVersion := flag.Bool("version", false, "print version and exit")
	showVersionShort := flag.Bool("v", false, "print version and exit")
	debug := flag.Bool("debug", false, "enable debug logging to stderr")
	theme := flag.String("theme", "", "color theme (dark, light, dracula, nord, monokai)")
	noGPU := flag.Bool("no-gpu", false, "disable GPU metrics")
	noTemp := flag.Bool("no-temp", false, "disable temperature metrics")
	flag.Parse()

	cfg := Config{
		RefreshInterval: *interval,
		ShowVersion:     *showVersion || *showVersionShort,
		Debug:           *debug,
		Theme:           *theme,
		NoGPU:           *noGPU,
		NoTemp:          *noTemp,
	}

	// Load config file (flags take precedence)
	fc := loadConfigFile()
	if fc != nil {
		if cfg.Theme == "" && fc.Theme != "" {
			cfg.Theme = fc.Theme
		}
		if !cfg.Debug && fc.Debug {
			cfg.Debug = true
		}
		if !cfg.NoGPU && fc.NoGPU {
			cfg.NoGPU = true
		}
		if !cfg.NoTemp && fc.NoTemp {
			cfg.NoTemp = true
		}
		if fc.Interval != "" && *interval == 1*time.Second {
			if d, err := time.ParseDuration(fc.Interval); err == nil {
				cfg.RefreshInterval = d
			}
		}
	}

	if cfg.RefreshInterval < 100*time.Millisecond {
		cfg.RefreshInterval = 100 * time.Millisecond
	}

	// Apply filter_users from config file; default if not set
	if fc != nil && len(fc.FilterUsers) > 0 {
		cfg.FilterUsers = fc.FilterUsers
	}
	if len(cfg.FilterUsers) == 0 {
		cfg.FilterUsers = DefaultFilterUsers
	}

	return cfg
}

func loadConfigFile() *fileConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(home, ".config", "hideTop", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil
	}
	return &fc
}
