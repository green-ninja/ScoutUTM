package stats

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// cpuTimes holds raw CPU jiffies from /proc/stat.
type cpuTimes struct {
	idle  uint64
	total uint64
}

// readCPUTimes reads aggregate CPU times from /proc/stat.
func readCPUTimes() (cpuTimes, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuTimes{}, fmt.Errorf("reading /proc/stat: %w", err)
	}

	// First line: "cpu  user nice system idle iowait irq softirq steal ..."
	line := strings.SplitN(string(data), "\n", 2)[0]
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTimes{}, fmt.Errorf("unexpected /proc/stat format")
	}

	var total, idle uint64
	for i, f := range fields[1:] {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return cpuTimes{}, fmt.Errorf("parsing /proc/stat field %d: %w", i, err)
		}
		total += v
		if i == 3 { // idle is the 4th value (index 3)
			idle = v
		}
	}
	return cpuTimes{idle: idle, total: total}, nil
}

// memInfo holds memory stats from /proc/meminfo.
type memInfo struct {
	totalMB     uint64
	availableMB uint64
}

// readMemInfo reads memory info from /proc/meminfo.
func readMemInfo() (memInfo, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return memInfo{}, fmt.Errorf("reading /proc/meminfo: %w", err)
	}

	var total, available uint64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = val / 1024 // kB → MB
		case "MemAvailable:":
			available = val / 1024
		}
	}
	return memInfo{totalMB: total, availableMB: available}, nil
}

// readUptime reads system uptime in seconds from /proc/uptime.
func readUptime() (uint64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, fmt.Errorf("reading /proc/uptime: %w", err)
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected /proc/uptime format")
	}

	// Uptime is a float like "12345.67"
	f, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("parsing uptime: %w", err)
	}
	return uint64(f), nil
}
