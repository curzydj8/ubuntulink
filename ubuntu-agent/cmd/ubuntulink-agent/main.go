// Command ubuntulink-agent: UbuntuLink device agent.
//
// Phase 2: HTTP API with pairing-code authorization and Bearer tokens.
// Read-only: monitoring snapshots and (later) desktop viewing; no remote operation.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/auth"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/config"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/identity"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/monitor"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/pairing"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/server"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/version"
)

func stateDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".config", "ubuntulink")
	}
	return filepath.Join(".", ".ubuntulink")
}

func main() {
	var (
		cfgPath = flag.String("config", "", "path to JSON config file (optional)")
		port    = flag.Int("port", 0, "listen port (overrides config)")
		once    = flag.Bool("once", false, "print one monitor snapshot and exit (debug)")
		showVer = flag.Bool("version", false, "print version and exit")
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
	if *port > 0 {
		cfg.Port = *port
	}

	dir := stateDir()

	devID, err := identity.LoadOrCreate(dir)
	if err != nil {
		slog.Error("failed to load device identity", "err", err)
		os.Exit(1)
	}

	p := pairing.New(
		time.Duration(cfg.PairCodeTTLSeconds)*time.Second,
		cfg.PairMaxAttempts,
		time.Duration(cfg.PairLockSeconds)*time.Second,
	)
	authz, err := auth.New(dir)
	if err != nil {
		slog.Error("failed to load paired devices", "err", err)
		os.Exit(1)
	}

	collector := monitor.NewCollector()

	if *once {
		collector.Collect()
		time.Sleep(1100 * time.Millisecond)
		snap := collector.Collect()
		out, _ := json.Marshal(snap)
		fmt.Println(string(out))
		return
	}

	srv := server.New(collector, p, authz, devID.DeviceID)

	slog.Info("starting",
		"app", version.AppName,
		"version", version.Version,
		"device_id", devID.DeviceID,
		"port", cfg.Port,
		"paired_devices", len(authz.List()),
	)
	// The pairing code is the user-visible secret for authorizing a new phone.
	// It is short-lived and single-use.
	slog.Info("pairing code (valid 5m, single-use)", "code", p.Code())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutting down")
		os.Exit(0)
	}()

	if err := srv.Serve(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
