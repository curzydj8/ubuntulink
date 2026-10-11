package monitor

import "time"

// Availability tells whether a metric was actually read from the system.
// Unsupported sensors must report Unavailable — never fake a 0.
type Availability string

const (
	Available   Availability = "available"
	Unavailable Availability = "unavailable"
)

// MetricMeta is embedded in every metric.
type MetricMeta struct {
	Available Availability `json:"available"`
	Reason    string       `json:"reason,omitempty"`
	Unit      string       `json:"unit,omitempty"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func okMeta(unit string) MetricMeta {
	return MetricMeta{Available: Available, Unit: unit, UpdatedAt: time.Now()}
}

func failMeta(reason string) MetricMeta {
	return MetricMeta{Available: Unavailable, Reason: reason, UpdatedAt: time.Now()}
}

type CPUInfo struct {
	MetricMeta
	UsagePercent float64      `json:"usage_percent"`
	Cores        int          `json:"cores"`
	FreqMHz      float64      `json:"freq_mhz,omitempty"`
	FreqStatus   Availability `json:"freq_status"`
}

type MemInfo struct {
	MetricMeta
	TotalBytes     uint64  `json:"total_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
}

type DiskInfo struct {
	MetricMeta
	Mountpoint   string  `json:"mountpoint"`
	TotalBytes   uint64  `json:"total_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type NetInfo struct {
	MetricMeta
	Iface         string `json:"iface"`
	RxBytes       uint64 `json:"rx_bytes"`
	TxBytes       uint64 `json:"tx_bytes"`
	RxBytesPerSec uint64 `json:"rx_bytes_per_sec"`
	TxBytesPerSec uint64 `json:"tx_bytes_per_sec"`
	OperState     string `json:"operstate"`
}

type GPUInfo struct {
	MetricMeta
	Name string `json:"name,omitempty"`
	// Utilization is not exposed by most open drivers via sysfs;
	// when unsupported the whole metric stays Unavailable.
	UsagePercent float64 `json:"usage_percent,omitempty"`
}

type TempInfo struct {
	MetricMeta
	Label   string  `json:"label"`
	Celsius float64 `json:"celsius"`
	Source  string  `json:"source"`
}

type SystemInfo struct {
	MetricMeta
	OSName        string `json:"os_name"`
	OSVersion     string `json:"os_version"`
	Kernel        string `json:"kernel"`
	Arch          string `json:"arch"`
	Hostname      string `json:"hostname"`
	UptimeSeconds uint64 `json:"uptime_seconds"`
}

type Snapshot struct {
	Timestamp time.Time  `json:"timestamp"`
	System    SystemInfo `json:"system"`
	CPU       CPUInfo    `json:"cpu"`
	Memory    MemInfo    `json:"memory"`
	Disks     []DiskInfo `json:"disks"`
	Network   []NetInfo  `json:"network"`
	GPU       GPUInfo    `json:"gpu"`
	Temps     []TempInfo `json:"temperatures"`
}
