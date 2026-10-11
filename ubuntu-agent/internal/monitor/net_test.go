package monitor

import "testing"

const sampleNetDev = `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1000       10    0    0    0     0          0         0     2000      20    0    0    0     0       0          0
  eth0: 5000       50    0    0    0     0          0         0     6000      60    0    0    0     0       0          0
`

func TestParseNetDev(t *testing.T) {
	m, err := parseNetDev([]byte(sampleNetDev))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	eth0, ok := m["eth0"]
	if !ok {
		t.Fatal("eth0 missing")
	}
	if eth0.rxBytes != 5000 || eth0.txBytes != 6000 {
		t.Fatalf("unexpected counters: %+v", eth0)
	}
	if _, ok := m["lo"]; !ok {
		t.Fatal("lo missing (parser keeps it; caller filters)")
	}
}

func TestParseNetDevShort(t *testing.T) {
	if _, err := parseNetDev([]byte("only-one-line")); err == nil {
		t.Fatal("expected error for short input")
	}
}
