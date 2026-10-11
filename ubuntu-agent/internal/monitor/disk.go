package monitor

import (
	"os"
	"syscall"
)

var diskMounts = []string{"/", "/home"}

// readDisks reports usage for each mountpoint via statfs.
func readDisks() []DiskInfo {
	seen := map[string]bool{}
	var out []DiskInfo
	for _, mp := range diskMounts {
		if seen[mp] {
			continue
		}
		seen[mp] = true
		var st syscall.Statfs_t
		if err := syscall.Statfs(mp, &st); err != nil {
			if _, serr := os.Stat(mp); serr != nil {
				continue // mountpoint does not exist; skip quietly
			}
			out = append(out, DiskInfo{
				MetricMeta: failMeta("statfs failed: " + err.Error()),
				Mountpoint: mp,
			})
			continue
		}
		total := st.Blocks * uint64(st.Bsize)
		free := st.Bavail * uint64(st.Bsize) // df's "Avail"
		// df's "Used" = (Blocks - Bfree); reserved blocks are not "used".
		used := (st.Blocks - st.Bfree) * uint64(st.Bsize)
		pct := 0.0
		// Match df(1): percentage over usable space (used+avail).
		if used+free > 0 {
			pct = 100 * float64(used) / float64(used+free)
		}
		out = append(out, DiskInfo{
			MetricMeta:   okMeta("bytes"),
			Mountpoint:   mp,
			TotalBytes:   total,
			FreeBytes:    free,
			UsedBytes:    used,
			UsagePercent: pct,
		})
	}
	return out
}
