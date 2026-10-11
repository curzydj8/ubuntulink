package config

import (
	"encoding/json"
	"os"
)

// Config holds agent settings. JSON keeps the agent dependency-free.
type Config struct {
	IntervalSeconds    int    `json:"interval_seconds"`
	LogLevel           string `json:"log_level"`
	Port               int    `json:"port"`
	PairCodeTTLSeconds int    `json:"pair_code_ttl_seconds"`
	PairMaxAttempts    int    `json:"pair_max_attempts"`
	PairLockSeconds    int    `json:"pair_lock_seconds"`
}

func Default() Config {
	return Config{
		IntervalSeconds:    5,
		LogLevel:           "info",
		Port:               38080,
		PairCodeTTLSeconds: 300, // 5 minutes
		PairMaxAttempts:    5,
		PairLockSeconds:    300, // 5 minutes lockout
	}
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
	def := Default()
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = def.IntervalSeconds
	}
	if cfg.Port <= 0 {
		cfg.Port = def.Port
	}
	if cfg.PairCodeTTLSeconds <= 0 {
		cfg.PairCodeTTLSeconds = def.PairCodeTTLSeconds
	}
	if cfg.PairMaxAttempts <= 0 {
		cfg.PairMaxAttempts = def.PairMaxAttempts
	}
	if cfg.PairLockSeconds <= 0 {
		cfg.PairLockSeconds = def.PairLockSeconds
	}
	return cfg, nil
}
