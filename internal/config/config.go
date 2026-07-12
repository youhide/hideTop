package config

import (
	"encoding/json"
	"flag"
	"fmt"
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
	NoPorts         bool
	FilterUsers     []string
	ProcLimit       int
	ExportDir       string
}

// DefaultFilterUsers is used when no custom filter is configured.
var DefaultFilterUsers = []string{"root", "_windowserver", "nobody"}

// fileConfig matches the JSON config file format.
type fileConfig struct {
	Interval    string   `json:"interval"`
	Theme       string   `json:"theme"`
	NoGPU       bool     `json:"no_gpu"`
	NoTemp      bool     `json:"no_temp"`
	NoPorts     bool     `json:"no_ports"`
	Debug       bool     `json:"debug"`
	FilterUsers []string `json:"filter_users"`
	ProcLimit   int      `json:"proc_limit"`
	ExportDir   string   `json:"export_dir"`
}

func Parse() Config {
	return parse(flag.CommandLine, os.Args[1:], loadConfigFile)
}

func parse(fs *flag.FlagSet, args []string, loadFile func() *fileConfig) Config {
	interval := fs.Duration("interval", 1*time.Second,
		"metrics refresh interval (e.g. 500ms, 1s, 2s)")
	showVersion := fs.Bool("version", false, "print version and exit")
	showVersionShort := fs.Bool("v", false, "print version and exit")
	debug := fs.Bool("debug", false, "enable debug logging to stderr")
	theme := fs.String("theme", "", "color theme (dark, light, dracula, nord, monokai)")
	noGPU := fs.Bool("no-gpu", false, "disable GPU metrics")
	noTemp := fs.Bool("no-temp", false, "disable temperature metrics")
	noPorts := fs.Bool("no-ports", false, "disable listening ports / connections collection")
	procLimit := fs.Int("proc-limit", 0, "max number of processes to display (0 = 50)")
	exportDir := fs.String("export-dir", "", "directory for JSON snapshot exports (default: home directory)")
	_ = fs.Parse(args)
	setFlags := visitedFlags(fs)

	cfg := Config{
		RefreshInterval: *interval,
		ShowVersion:     *showVersion || *showVersionShort,
		Debug:           *debug,
		Theme:           *theme,
		NoGPU:           *noGPU,
		NoTemp:          *noTemp,
		NoPorts:         *noPorts,
		ProcLimit:       *procLimit,
		ExportDir:       *exportDir,
	}

	// Load config file (flags take precedence)
	fc := loadFile()
	if fc != nil {
		if !setFlags["theme"] && fc.Theme != "" {
			cfg.Theme = fc.Theme
		}
		if !setFlags["debug"] && fc.Debug {
			cfg.Debug = true
		}
		if !setFlags["no-gpu"] && fc.NoGPU {
			cfg.NoGPU = true
		}
		if !setFlags["no-temp"] && fc.NoTemp {
			cfg.NoTemp = true
		}
		if !setFlags["no-ports"] && fc.NoPorts {
			cfg.NoPorts = true
		}
		if !setFlags["export-dir"] && fc.ExportDir != "" {
			cfg.ExportDir = fc.ExportDir
		}
		if !setFlags["interval"] && fc.Interval != "" {
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

	// Apply proc_limit from config file if not set via CLI
	if fc != nil && !setFlags["proc-limit"] && fc.ProcLimit > 0 {
		cfg.ProcLimit = fc.ProcLimit
	}
	if cfg.ProcLimit <= 0 {
		cfg.ProcLimit = 50
	}

	return cfg
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	set := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		set[f.Name] = true
	})
	return set
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
		fmt.Fprintf(os.Stderr, "hideTop: ignoring malformed config at %s: %v\n", path, err)
		return nil
	}
	return &fc
}

// configPath returns the path to the user config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "hideTop", "config.json"), nil
}

// SaveInterval persists the refresh interval to the config file, preserving
// any other settings already present. Existing config is merged so unrelated
// fields are not lost.
func SaveInterval(d time.Duration) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	fc := fileConfig{}
	if existing := loadConfigFile(); existing != nil {
		fc = *existing
	}
	fc.Interval = d.String()

	data, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
