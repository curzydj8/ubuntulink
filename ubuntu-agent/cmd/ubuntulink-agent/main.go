// Command ubuntulink-agent collects real system metrics and prints them
// as JSON snapshots. Phase 1: local collection only, no network server.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/config"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/monitor"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/version"
)

func main() {
	var (
		cfgPath  = flag.String("config", "", "path to JSON config file (optional)")
		interval = flag.Int("interval", 0, "collection interval seconds (overrides config)")
		once     = flag.Bool("once", false, "collect a single snapshot and exit")
		showVer  = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("%s %s\n", version.AppName, version.Version)
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}
	if *interval > 0 {
		cfg.IntervalSeconds = *interval
	}
	slog.Info("starting", "app", version.AppName, "version", version.Version,
		"interval_s", cfg.IntervalSeconds)

	collector := monitor.NewCollector()
	emit := func() {
		snap := collector.Collect()
		out, err := json.Marshal(snap)
		if err != nil {
			slog.Error("failed to marshal snapshot", "err", err)
			return
		}
		fmt.Println(string(out))
	}

	if *once {
		// Prime deltas, then take the real sample.
		collector.Collect()
		time.Sleep(1100 * time.Millisecond)
		emit()
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()
	emit()
	for {
		select {
		case <-ticker.C:
			emit()
		case sig := <-sigCh:
			slog.Info("shutting down", "signal", sig.String())
			return
		}
	}
}
