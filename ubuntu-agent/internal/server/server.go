// Package server exposes the agent's HTTP API: pairing, auth, monitoring.
package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/auth"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/desktop"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/monitor"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/pairing"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/version"
)

type Server struct {
	collector *monitor.Collector
	pairing   *pairing.Pairing
	auth      *auth.Auth
	deviceID  string
	mux       *http.ServeMux
	streamer  *desktop.Streamer
}

func New(collector *monitor.Collector, p *pairing.Pairing, a *auth.Auth, deviceID string) *Server {
	s := &Server{
		collector: collector,
		pairing:   p,
		auth:      a,
		deviceID:  deviceID,
		mux:       http.NewServeMux(),
		// Phase 4: test pattern source. PortalSource replaces this
		// once validated in a graphical session.
		streamer: desktop.NewStreamer(desktop.NewTestPatternSource(960, 540), 5),
	}
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /api/v1/info", s.handleInfo)
	s.mux.HandleFunc("POST /api/v1/pair", s.handlePair)
	s.mux.HandleFunc("GET /api/v1/monitor", s.requireAuth(s.handleMonitor))
	s.mux.HandleFunc("POST /api/v1/unpair", s.requireAuth(s.handleUnpair))
	s.mux.HandleFunc("GET /api/v1/peers", s.requireAuth(s.handlePeers))
	s.mux.HandleFunc("GET /api/v1/desktop/stream", s.requireAuth(s.streamer.ServeHTTP))
	// Public read-only dashboard (no auth): static page + metrics JSON.
	s.mux.HandleFunc("GET /dashboard", s.handleDashboard)
	s.mux.HandleFunc("GET /api/v1/dashboard/metrics", s.handleDashboardMetrics)
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error_code": code, "error": msg})
}

// handleHealthz is public: load balancers and discovery use it.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version.Version})
}

// handleInfo is public but minimal: enough for discovery, nothing sensitive.
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"device_id":     s.deviceID,
		"agent_version": version.Version,
		"paired_count":  len(s.auth.List()),
	})
}

type pairRequest struct {
	Code       string `json:"code"`
	ClientID   string `json:"client_id"`
	ClientName string `json:"client_name"`
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	var req pairRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if err := s.pairing.Verify(req.Code); err != nil {
		switch {
		case errors.Is(err, pairing.ErrLocked):
			writeErr(w, http.StatusTooManyRequests, "locked", "too many attempts, try later")
		case errors.Is(err, pairing.ErrExpired):
			writeErr(w, http.StatusGone, "expired", "pairing code expired")
		default:
			writeErr(w, http.StatusUnauthorized, "invalid_code", "invalid pairing code")
		}
		slog.Warn("pairing failed", "client_id", req.ClientID)
		return
	}
	token, err := s.auth.IssueToken(req.ClientID, req.ClientName)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	slog.Info("device paired", "client_id", req.ClientID, "client_name", req.ClientName)
	writeJSON(w, http.StatusOK, map[string]string{"token": token, "device_id": s.deviceID})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dev, ok := s.auth.Validate(bearerToken(r))
		if !ok {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "valid Bearer token required")
			return
		}
		r.Header.Set("X-Client-ID", dev.ClientID)
		next(w, r)
	}
}

func (s *Server) handleMonitor(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.collector.Collect())
}

// handleUnpair revokes the caller's own token.
func (s *Server) handleUnpair(w http.ResponseWriter, r *http.Request) {
	s.auth.Revoke(bearerToken(r))
	writeJSON(w, http.StatusOK, map[string]string{"status": "unpaired"})
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"peers": s.auth.List()})
}

// Serve starts listening; blocks until the server stops.
func (s *Server) Serve(addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	slog.Info("listening", "addr", addr)
	return srv.ListenAndServe()
}
