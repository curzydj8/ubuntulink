package monitor

import (
	"os"
	"time"
)

// Collector holds previous samples needed for rate/usage computation.
type Collector struct {
	prevCPU  cpuTimes
	prevNet  map[string]netCounters
	prevTime time.Time
	hasPrev  bool
}

func NewCollector() *Collector { return &Collector{} }

// Collect gathers a full snapshot. The first call primes the delta
// accumulators, so CPU usage and network rates are 0 on that pass.
func (c *Collector) Collect() Snapshot {
	now := time.Now()

	cpuT, cpuErr := func() (cpuTimes, error) {
		data, err := os.ReadFile("/proc/stat")
		if err != nil {
			return cpuTimes{}, err
		}
		return parseCPUStat(data)
	}()

	cpuInfo := CPUInfo{MetricMeta: failMeta("unavailable")}
	if cpuErr == nil {
		cores, coreErr := countCores()
		usage := 0.0
		if c.hasPrev {
			usage = cpuUsage(c.prevCPU, cpuT)
		}
		cpuInfo = CPUInfo{
			MetricMeta:   okMeta("percent"),
			UsagePercent: usage,
			Cores:        cores,
		}
		if coreErr != nil {
			cpuInfo.MetricMeta = failMeta("core count failed: " + coreErr.Error())
		} else if freq, ok := cpuFreqMHz(); ok {
			cpuInfo.FreqMHz = freq
			cpuInfo.FreqStatus = Available
		} else {
			cpuInfo.FreqStatus = Unavailable
		}
		c.prevCPU = cpuT
	} else {
		cpuInfo.MetricMeta = failMeta("cannot read /proc/stat: " + cpuErr.Error())
	}

	netInfo := readNetwork(c.prevNet, c.prevTime)
	// Refresh prevNet for the next pass.
	if data, err := os.ReadFile("/proc/net/dev"); err == nil {
		if m, perr := parseNetDev(data); perr == nil {
			c.prevNet = m
		}
	}
	c.prevTime = now
	c.hasPrev = true

	return Snapshot{
		Timestamp: now,
		System:    readSystem(),
		CPU:       cpuInfo,
		Memory:    readMemory(),
		Disks:     readDisks(),
		Network:   netInfo,
		GPU:       readGPU(),
		Temps:     readTemps(),
	}
}
