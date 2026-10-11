package monitor

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// readTemps reads /sys/class/thermal/thermal_zone*/temp (millidegree Celsius).
func readTemps() []TempInfo {
	zones, err := filepath.Glob("/sys/class/thermal/thermal_zone*")
	if err != nil || len(zones) == 0 {
		return []TempInfo{{
			MetricMeta: failMeta("no thermal zones in /sys/class/thermal"),
		}}
	}
	var out []TempInfo
	for _, z := range zones {
		rawTemp, err := os.ReadFile(filepath.Join(z, "temp"))
		if err != nil {
			continue
		}
		milli, err := strconv.ParseFloat(strings.TrimSpace(string(rawTemp)), 64)
		if err != nil {
			continue
		}
		label := filepath.Base(z)
		if t, err := os.ReadFile(filepath.Join(z, "type")); err == nil {
			label = strings.TrimSpace(string(t))
		}
		out = append(out, TempInfo{
			MetricMeta: okMeta("celsius"),
			Label:      label,
			Celsius:    milli / 1000,
			Source:     z + "/temp",
		})
	}
	if len(out) == 0 {
		return []TempInfo{{
			MetricMeta: failMeta("thermal zones exist but no readable temp"),
		}}
	}
	return out
}
