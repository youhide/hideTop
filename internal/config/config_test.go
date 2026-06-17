package config

import (
	"flag"
	"io"
	"testing"
	"time"
)

func testFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func TestParseExplicitDefaultIntervalOverridesConfig(t *testing.T) {
	cfg := parse(testFlagSet(), []string{"--interval", "1s"}, func() *fileConfig {
		return &fileConfig{Interval: "5s"}
	})

	if cfg.RefreshInterval != time.Second {
		t.Fatalf("interval = %s, want 1s", cfg.RefreshInterval)
	}
}

func TestParseExplicitFalseBoolOverridesConfig(t *testing.T) {
	cfg := parse(testFlagSet(), []string{"--no-gpu=false"}, func() *fileConfig {
		return &fileConfig{NoGPU: true}
	})

	if cfg.NoGPU {
		t.Fatalf("NoGPU = true, want false from explicit CLI flag")
	}
}

func TestParseExplicitProcLimitZeroUsesDefault(t *testing.T) {
	cfg := parse(testFlagSet(), []string{"--proc-limit", "0"}, func() *fileConfig {
		return &fileConfig{ProcLimit: 10}
	})

	if cfg.ProcLimit != 50 {
		t.Fatalf("ProcLimit = %d, want default 50", cfg.ProcLimit)
	}
}
