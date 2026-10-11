package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/auth"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/monitor"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/pairing"
)

func newTestServer(t *testing.T) (*Server, *pairing.Pairing) {
	t.Helper()
	p := pairing.New(5*time.Minute, 5, 5*time.Minute)
	a, err := auth.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := New(monitor.NewCollector(), p, a, "test-device-id")
	return s, p
}

func TestHealthzAndInfoPublic(t *testing.T) {
	s, _ := newTestServer(t)
	for _, path := range []string{"/healthz", "/api/v1/info"} {
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", path, rec.Code)
		}
	}
}

func TestMonitorRequiresAuth(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/monitor", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestPairFlow(t *testing.T) {
	s, p := newTestServer(t)

	// Wrong code -> 401.
	body := `{"code":"000000","client_id":"c1","client_name":"Phone"}`
	req := httptest.NewRequest("POST", "/api/v1/pair", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong code: expected 401, got %d", rec.Code)
	}

	// Right code -> 200 + token.
	body = `{"code":"` + p.Code() + `","client_id":"c1","client_name":"Phone"}`
	req = httptest.NewRequest("POST", "/api/v1/pair", strings.NewReader(body))
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("right code: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || resp.Token == "" {
		t.Fatalf("no token in response: %v", err)
	}

	// Monitor with token -> 200.
	req = httptest.NewRequest("GET", "/api/v1/monitor", nil)
	req.Header.Set("Authorization", "Bearer "+resp.Token)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("monitor with token: expected 200, got %d", rec.Code)
	}

	// Unpair -> token dead.
	req = httptest.NewRequest("POST", "/api/v1/unpair", nil)
	req.Header.Set("Authorization", "Bearer "+resp.Token)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unpair: expected 200, got %d", rec.Code)
	}
	req = httptest.NewRequest("GET", "/api/v1/monitor", nil)
	req.Header.Set("Authorization", "Bearer "+resp.Token)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("after unpair: expected 401, got %d", rec.Code)
	}
}

func TestPairLockout(t *testing.T) {
	p := pairing.New(5*time.Minute, 2, 5*time.Minute)
	a, _ := auth.New(t.TempDir())
	s := New(monitor.NewCollector(), p, a, "test-device-id")
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/v1/pair", strings.NewReader(`{"code":"000000","client_id":"c"}`))
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
	}
	req := httptest.NewRequest("POST", "/api/v1/pair", strings.NewReader(`{"code":"000000","client_id":"c"}`))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after lockout, got %d", rec.Code)
	}
}
