package monitor

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func osRelease() (name, version string) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "unknown", "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "NAME=") {
			name = strings.Trim(strings.TrimPrefix(line, "NAME="), `"`)
		} else if strings.HasPrefix(line, "VERSION=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION="), `"`)
		}
	}
	if name == "" {
		name = "unknown"
	}
	if version == "" {
		version = "unknown"
	}
	return name, version
}

func kernelRelease() string {
	var u syscall.Utsname
	if err := syscall.Uname(&u); err != nil {
		return "unknown"
	}
	b := make([]byte, 0, 65)
	for _, v := range u.Release {
		if v == 0 {
			break
		}
		b = append(b, byte(v))
	}
	return string(b)
}

func uptimeSeconds() uint64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	f := strings.Fields(string(data))
	if len(f) == 0 {
		return 0
	}
	v, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return 0
	}
	return uint64(v)
}

func readSystem() SystemInfo {
	name, version := osRelease()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return SystemInfo{
		MetricMeta:    okMeta(""),
		OSName:        name,
		OSVersion:     version,
		Kernel:        kernelRelease(),
		Arch:          runtime.GOARCH,
		Hostname:      hostname,
		UptimeSeconds: uptimeSeconds(),
	}
}
