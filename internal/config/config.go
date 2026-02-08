package config

import (
	"flag"
	"time"
)

type Config struct {
	RefreshInterval time.Duration
}

func Parse() Config {
	interval := flag.Duration("interval", 1*time.Second,
		"metrics refresh interval (e.g. 500ms, 1s, 2s)")
	flag.Parse()

	cfg := Config{
		RefreshInterval: *interval,
	}

	if cfg.RefreshInterval < 100*time.Millisecond {
		cfg.RefreshInterval = 100 * time.Millisecond
	}

	return cfg
}
