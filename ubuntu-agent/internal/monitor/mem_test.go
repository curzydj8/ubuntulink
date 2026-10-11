package monitor

import "testing"

const sampleMeminfo = `MemTotal:        3920808 kB
MemFree:           512000 kB
MemAvailable:    1234567 kB
Buffers:           100000 kB
`

func TestParseMemInfo(t *testing.T) {
	total, avail, err := parseMemInfo([]byte(sampleMeminfo))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if total != 3920808 {
		t.Fatalf("total = %d, want 3920808", total)
	}
	if avail != 1234567 {
		t.Fatalf("avail = %d, want 1234567", avail)
	}
}

func TestParseMemInfoMissing(t *testing.T) {
	if _, _, err := parseMemInfo([]byte("MemFree: 1 kB\n")); err == nil {
		t.Fatal("expected error for missing MemTotal")
	}
}
