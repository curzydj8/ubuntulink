package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/monitor"
)

func TestDashboardPagePublic(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequest("GET", "/dashboard", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") || !strings.Contains(body, "UbuntuLink") {
		t.Fatalf("dashboard page does not look like the embedded HTML")
	}
}

func TestDashboardMetricsPublicNoAuth(t *testing.T) {
	s, _ := newTestServer(t)
	// No Authorization header: must still succeed (public read-only).
	req := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 without auth, got %d", rec.Code)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, key := range []string{"timestamp", "cpu", "memory", "disk", "network", "temperature"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("missing expected key %q in metrics response", key)
		}
	}
}

func TestDashboardMetricsNoSensitiveData(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	body := strings.ToLower(rec.Body.String())
	for _, leak := range []string{"device_id", "hostname", "token", "paired", "client_id", "\"system\"", "\"gpu\""} {
		if strings.Contains(body, leak) {
			t.Fatalf("metrics response leaks sensitive field %q", leak)
		}
	}
}

func TestToDashboardMetricsProjection(t *testing.T) {
	snap := monitor.Snapshot{
		Timestamp: time.Now(),
		CPU: monitor.CPUInfo{
			MetricMeta:   monitor.MetricMeta{Available: monitor.Available},
			UsagePercent: 12.5,
			Cores:        4,
			FreqMHz:      2100,
			FreqStatus:   monitor.Available,
		},
		Memory: monitor.MemInfo{
			MetricMeta:     monitor.MetricMeta{Available: monitor.Available},
			TotalBytes:     4 * 1073741824,
			AvailableBytes: 2 * 1073741824,
			UsagePercent:   50,
		},
		Disks: []monitor.DiskInfo{
			{
				MetricMeta:   monitor.MetricMeta{Available: monitor.Available},
				Mountpoint:   "/",
				TotalBytes:   100 * 1073741824,
				UsedBytes:    36 * 1073741824,
				UsagePercent: 36,
			},
		},
		Network: []monitor.NetInfo{
			{
				MetricMeta:    monitor.MetricMeta{Available: monitor.Available},
				Iface:         "eth0",
				RxBytesPerSec: 2048,
				TxBytesPerSec: 1024,
			},
		},
		Temps: []monitor.TempInfo{
			{
				MetricMeta: monitor.MetricMeta{Available: monitor.Available},
				Label:      "Package",
				Celsius:    45.5,
			},
		},
	}
	m := toDashboardMetrics(snap)
	if m.CPU.UsagePercent == nil || *m.CPU.UsagePercent != 12.5 {
		t.Fatalf("cpu usage not projected: %+v", m.CPU)
	}
	if m.CPU.Cores == nil || *m.CPU.Cores != 4 {
		t.Fatalf("cpu cores not projected")
	}
	if m.Memory.TotalMB == nil || *m.Memory.TotalMB != 4096 {
		t.Fatalf("memory total_mb not projected: %+v", m.Memory)
	}
	if m.Disk.Path != "/" || m.Disk.TotalGB == nil || *m.Disk.TotalGB != 100 {
		t.Fatalf("disk not projected: %+v", m.Disk)
	}
	if len(m.Network) != 1 || m.Network[0].RxKBps != 2 {
		t.Fatalf("network not projected: %+v", m.Network)
	}
	if len(m.Temperature) != 1 || m.Temperature[0].Celsius != 45.5 {
		t.Fatalf("temperature not projected: %+v", m.Temperature)
	}

	// Unavailable metrics must serialize as null, never as faked 0.
	empty := toDashboardMetrics(monitor.Snapshot{})
	out, _ := json.Marshal(empty)
	if strings.Contains(string(out), `"usage_percent":0`) {
		t.Fatalf("unavailable metric serialized as 0: %s", out)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	var cpu map[string]json.RawMessage
	if err := json.Unmarshal(decoded["cpu"], &cpu); err != nil {
		t.Fatalf("invalid cpu JSON: %v", err)
	}
	if string(cpu["usage_percent"]) != "null" {
		t.Fatalf("expected null usage_percent when unavailable, got %s", cpu["usage_percent"])
	}
}
