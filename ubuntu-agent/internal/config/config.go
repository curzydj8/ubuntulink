package config

import (
	"encoding/json"
	"os"
)

// Config holds agent settings. JSON keeps the Phase 1 agent dependency-free.
type Config struct {
	IntervalSeconds int    `json:"interval_seconds"`
	LogLevel        string `json:"log_level"`
}

func Default() Config {
	return Config{IntervalSeconds: 5, LogLevel: "info"}
}

// Load reads path; empty path or missing file yields defaults (not an error).
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = Default().IntervalSeconds
	}
	return cfg, nil
}
