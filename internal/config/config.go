package config

import (
	"flag"
	"time"
)

type Config struct {
	RefreshInterval time.Duration
	ShowVersion     bool
}

func Parse() Config {
	interval := flag.Duration("interval", 1*time.Second,
		"metrics refresh interval (e.g. 500ms, 1s, 2s)")
	showVersion := flag.Bool("version", false, "print version and exit")
	showVersionShort := flag.Bool("v", false, "print version and exit")
	flag.Parse()

	cfg := Config{
		RefreshInterval: *interval,
		ShowVersion:     *showVersion || *showVersionShort,
	}

	if cfg.RefreshInterval < 100*time.Millisecond {
		cfg.RefreshInterval = 100 * time.Millisecond
	}

	return cfg
}
