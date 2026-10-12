package server

import (
	"net/http"
	"time"

	"github.com/curzydj8/ubuntulink/ubuntu-agent/internal/monitor"
	"github.com/curzydj8/ubuntulink/ubuntu-agent/web"
)

// Dashboard endpoints are public and read-only: they expose only
// non-sensitive system metrics (CPU, memory, disk, network, temperature).
// No device identity, hostname, tokens, or pairing material is included.

// dashboardCPU mirrors the fields consumed by web/dashboard.html.
// Pointer fields serialize as null when the metric is unavailable,
// which the page renders as "不可用" (never a faked 0).
type dashboardCPU struct {
	UsagePercent *float64 `json:"usage_percent"`
	Cores        *int     `json:"cores"`
	FreqMHz      *float64 `json:"freq_mhz,omitempty"`
}

type dashboardMemory struct {
	UsedPercent *float64 `json:"used_percent"`
	TotalMB     *float64 `json:"total_mb"`
	AvailableMB *float64 `json:"available_mb"`
}

type dashboardDisk struct {
	UsedPercent *float64 `json:"used_percent"`
	UsedGB      *float64 `json:"used_gb"`
	TotalGB     *float64 `json:"total_gb"`
	Path        string   `json:"path,omitempty"`
}

type dashboardNet struct {
	Iface  string  `json:"iface"`
	RxKBps float64 `json:"rx_kbps"`
	TxKBps float64 `json:"tx_kbps"`
}

type dashboardTemp struct {
	Label   string  `json:"label"`
	Celsius float64 `json:"celsius"`
}

type dashboardMetrics struct {
	Timestamp   time.Time       `json:"timestamp"`
	CPU         dashboardCPU    `json:"cpu"`
	Memory      dashboardMemory `json:"memory"`
	Disk        dashboardDisk   `json:"disk"`
	Network     []dashboardNet  `json:"network"`
	Temperature []dashboardTemp `json:"temperature"`
}

func f64ptr(v float64) *float64 { return &v }
func intptr(v int) *int         { return &v }

// toDashboardMetrics projects a monitor snapshot onto the public dashboard
// shape. Only CPU, memory, disk, network and temperature are included;
// system identity (hostname, OS details) and GPU are deliberately excluded.
func toDashboardMetrics(snap monitor.Snapshot) dashboardMetrics {
	m := dashboardMetrics{Timestamp: snap.Timestamp}

	if snap.CPU.Available == monitor.Available {
		m.CPU.UsagePercent = f64ptr(snap.CPU.UsagePercent)
		m.CPU.Cores = intptr(snap.CPU.Cores)
		if snap.CPU.FreqStatus == monitor.Available {
			m.CPU.FreqMHz = f64ptr(snap.CPU.FreqMHz)
		}
	}

	if snap.Memory.Available == monitor.Available {
		m.Memory.UsedPercent = f64ptr(snap.Memory.UsagePercent)
		m.Memory.TotalMB = f64ptr(float64(snap.Memory.TotalBytes) / 1048576)
		m.Memory.AvailableMB = f64ptr(float64(snap.Memory.AvailableBytes) / 1048576)
	}

	// The dashboard shows a single disk: prefer the root mountpoint.
	var root *monitor.DiskInfo
	for i := range snap.Disks {
		d := &snap.Disks[i]
		if d.Available != monitor.Available {
			continue
		}
		if root == nil {
			root = d
		}
		if d.Mountpoint == "/" {
			root = d
			break
		}
	}
	if root != nil {
		m.Disk.UsedPercent = f64ptr(root.UsagePercent)
		m.Disk.UsedGB = f64ptr(float64(root.UsedBytes) / 1073741824)
		m.Disk.TotalGB = f64ptr(float64(root.TotalBytes) / 1073741824)
		m.Disk.Path = root.Mountpoint
	}

	for _, n := range snap.Network {
		if n.Available != monitor.Available {
			continue
		}
		m.Network = append(m.Network, dashboardNet{
			Iface:  n.Iface,
			RxKBps: float64(n.RxBytesPerSec) / 1024,
			TxKBps: float64(n.TxBytesPerSec) / 1024,
		})
	}

	for _, t := range snap.Temps {
		if t.Available != monitor.Available {
			continue
		}
		m.Temperature = append(m.Temperature, dashboardTemp{
			Label:   t.Label,
			Celsius: t.Celsius,
		})
	}

	return m
}

// handleDashboard serves the embedded dashboard HTML page.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(web.DashboardHTML)
}

// handleDashboardMetrics serves the public read-only metrics JSON.
// No authentication: the payload contains only non-sensitive system metrics.
func (s *Server) handleDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, toDashboardMetrics(s.collector.Collect()))
}
