package monitor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// parseMemInfo extracts MemTotal and MemAvailable (kB) from /proc/meminfo.
func parseMemInfo(data []byte) (totalKB, availKB uint64, err error) {
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		switch f[0] {
		case "MemTotal:":
			totalKB, err = strconv.ParseUint(f[1], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("bad MemTotal: %w", err)
			}
		case "MemAvailable:":
			availKB, err = strconv.ParseUint(f[1], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("bad MemAvailable: %w", err)
			}
		}
	}
	if totalKB == 0 {
		return 0, 0, fmt.Errorf("MemTotal not found")
	}
	return totalKB, availKB, nil
}

func readMemory() MemInfo {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return MemInfo{MetricMeta: failMeta("cannot read /proc/meminfo: " + err.Error())}
	}
	totalKB, availKB, err := parseMemInfo(data)
	if err != nil {
		return MemInfo{MetricMeta: failMeta(err.Error())}
	}
	total := totalKB * 1024
	avail := availKB * 1024
	used := total - avail
	pct := 0.0
	if total > 0 {
		pct = 100 * float64(used) / float64(total)
	}
	return MemInfo{
		MetricMeta:     okMeta("bytes"),
		TotalBytes:     total,
		AvailableBytes: avail,
		UsedBytes:      used,
		UsagePercent:   pct,
	}
}
