package monitor

import (
	"math"
	"testing"
)

const sampleStat = `cpu  100 0 50 800 10 0 5 0 0 0
cpu0 60 0 30 400 5 0 2 0 0 0
cpu1 40 0 20 400 5 0 3 0 0 0
intr 1234
`

func TestParseCPUStat(t *testing.T) {
	ct, err := parseCPUStat([]byte(sampleStat))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if ct.user != 100 || ct.system != 50 || ct.idle != 800 || ct.iowait != 10 {
		t.Fatalf("unexpected parse result: %+v", ct)
	}
}

func TestParseCPUStatMissing(t *testing.T) {
	if _, err := parseCPUStat([]byte("intr 1\n")); err == nil {
		t.Fatal("expected error for missing cpu line")
	}
}

func TestCPUUsage(t *testing.T) {
	prev := cpuTimes{user: 100, nice: 0, system: 50, idle: 800, iowait: 10}
	cur := cpuTimes{user: 150, nice: 0, system: 75, idle: 900, iowait: 10}
	// total delta = 175, idle delta = 100 -> usage = 75/175
	got := cpuUsage(prev, cur)
	want := 100 * 75.0 / 175.0
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestCPUUsageZeroDelta(t *testing.T) {
	c := cpuTimes{user: 1, idle: 2}
	if got := cpuUsage(c, c); got != 0 {
		t.Fatalf("expected 0 for zero delta, got %v", got)
	}
}
