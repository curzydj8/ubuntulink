package monitor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type cpuTimes struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

func (c cpuTimes) total() uint64 {
	return c.user + c.nice + c.system + c.idle + c.iowait + c.irq + c.softirq + c.steal
}

func (c cpuTimes) idleAll() uint64 { return c.idle + c.iowait }

// parseCPUStat parses the aggregate "cpu " line of /proc/stat.
func parseCPUStat(data []byte) (cpuTimes, error) {
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 9 {
			return cpuTimes{}, fmt.Errorf("short cpu line: %q", line)
		}
		vals := make([]uint64, 8)
		for i := 0; i < 8; i++ {
			v, err := strconv.ParseUint(f[i+1], 10, 64)
			if err != nil {
				return cpuTimes{}, fmt.Errorf("bad cpu field %d: %w", i, err)
			}
			vals[i] = v
		}
		return cpuTimes{
			user: vals[0], nice: vals[1], system: vals[2], idle: vals[3],
			iowait: vals[4], irq: vals[5], softirq: vals[6], steal: vals[7],
		}, nil
	}
	return cpuTimes{}, fmt.Errorf("no aggregate cpu line found")
}

// cpuUsage computes usage % between two samples.
func cpuUsage(prev, cur cpuTimes) float64 {
	totalDelta := cur.total() - prev.total()
	idleDelta := cur.idleAll() - prev.idleAll()
	if totalDelta == 0 {
		return 0
	}
	return 100 * float64(totalDelta-idleDelta) / float64(totalDelta)
}

func countCores() (int, error) {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0, err
	}
	n := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "processor") {
			n++
		}
	}
	if n == 0 {
		return 0, fmt.Errorf("no processor entries in /proc/cpuinfo")
	}
	return n, nil
}

// cpuFreqMHz returns the first "cpu MHz" found; not all platforms expose it.
func cpuFreqMHz() (float64, bool) {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "cpu MHz") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			v, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err == nil && v > 0 {
				return v, true
			}
		}
	}
	return 0, false
}
