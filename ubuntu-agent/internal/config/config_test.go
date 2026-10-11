package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsOnMissing(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if cfg.IntervalSeconds != 5 {
		t.Fatalf("interval = %d, want 5", cfg.IntervalSeconds)
	}
}

func TestLoadEmptyPath(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("empty path should not error: %v", err)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("log level = %q, want info", cfg.LogLevel)
	}
}

func TestLoadFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "agent.json")
	if err := os.WriteFile(p, []byte(`{"interval_seconds": 2, "log_level": "debug"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if cfg.IntervalSeconds != 2 || cfg.LogLevel != "debug" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadBadJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(p, []byte(`{oops`), 0600)
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestDefaultsPhase2(t *testing.T) {
	cfg, _ := Load("")
	if cfg.Port != 38080 {
		t.Fatalf("port = %d, want 38080", cfg.Port)
	}
	if cfg.PairCodeTTLSeconds != 300 || cfg.PairMaxAttempts != 5 || cfg.PairLockSeconds != 300 {
		t.Fatalf("pair defaults wrong: %+v", cfg)
	}
}
