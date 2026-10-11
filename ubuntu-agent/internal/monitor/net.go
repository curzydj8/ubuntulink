package monitor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type netCounters struct {
	rxBytes, txBytes uint64
}

// parseNetDev parses /proc/net/dev into per-iface counters.
func parseNetDev(data []byte) (map[string]netCounters, error) {
	out := map[string]netCounters{}
	lines := strings.Split(string(data), "\n")
	if len(lines) < 3 {
		return nil, fmt.Errorf("/proc/net/dev too short")
	}
	for _, line := range lines[2:] { // skip 2 header lines
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		if iface == "" {
			continue
		}
		f := strings.Fields(parts[1])
		if len(f) < 16 {
			continue
		}
		rx, err1 := strconv.ParseUint(f[0], 10, 64)
		tx, err2 := strconv.ParseUint(f[8], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		out[iface] = netCounters{rxBytes: rx, txBytes: tx}
	}
	return out, nil
}

func operState(iface string) string {
	data, err := os.ReadFile("/sys/class/net/" + iface + "/operstate")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

// readNetwork computes per-second rates from the previous sample.
func readNetwork(prev map[string]netCounters, prevTime time.Time) []NetInfo {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return []NetInfo{{
			MetricMeta: failMeta("cannot read /proc/net/dev: " + err.Error()),
		}}
	}
	cur, err := parseNetDev(data)
	if err != nil {
		return []NetInfo{{MetricMeta: failMeta(err.Error())}}
	}
	dt := time.Since(prevTime).Seconds()
	var out []NetInfo
	for iface, c := range cur {
		if iface == "lo" {
			continue
		}
		info := NetInfo{
			MetricMeta: okMeta("bytes"),
			Iface:      iface,
			RxBytes:    c.rxBytes,
			TxBytes:    c.txBytes,
			OperState:  operState(iface),
		}
		if p, ok := prev[iface]; ok && dt > 0 {
			if c.rxBytes >= p.rxBytes {
				info.RxBytesPerSec = uint64(float64(c.rxBytes-p.rxBytes) / dt)
			}
			if c.txBytes >= p.txBytes {
				info.TxBytesPerSec = uint64(float64(c.txBytes-p.txBytes) / dt)
			}
		}
		out = append(out, info)
	}
	return out
}
