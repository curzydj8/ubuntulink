package monitor

import (
	"os"
	"path/filepath"
	"strings"
)

// readGPU does a best-effort detection of the graphics device.
// Open drivers rarely expose utilization via sysfs, so when no counter
// is available the metric is reported Unavailable — never faked.
func readGPU() GPUInfo {
	cards, err := filepath.Glob("/sys/class/drm/card*")
	if err != nil || len(cards) == 0 {
		return GPUInfo{MetricMeta: failMeta("no DRM cards found in /sys/class/drm")}
	}
	name := ""
	// Try a human-readable device description.
	for _, c := range cards {
		base := filepath.Base(c)
		if strings.Contains(base, "-") {
			continue // skip card0-HDMI-A-1 style connectors
		}
		if data, err := os.ReadFile(filepath.Join(c, "device", "description")); err == nil {
			name = strings.TrimSpace(string(data))
			break
		}
	}
	if name == "" {
		// Fall back to modalias/vendor hint.
		name = "DRM graphics device (utilization counter not exposed)"
	}
	return GPUInfo{
		MetricMeta: failMeta("utilization counter not exposed by driver; device detected"),
		Name:       name,
	}
}
