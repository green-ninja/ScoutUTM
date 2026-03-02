package stats

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// netCounters holds byte counters for a network interface.
type netCounters struct {
	rxBytes uint64
	txBytes uint64
}

// readNetCounters reads byte counters for iface from /proc/net/dev.
func readNetCounters(iface string) (netCounters, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return netCounters{}, fmt.Errorf("reading /proc/net/dev: %w", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, iface+":") {
			continue
		}
		// Format: "iface: rxbytes rxpackets ... txbytes txpackets ..."
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 10 {
			return netCounters{}, fmt.Errorf("unexpected /proc/net/dev format for %s", iface)
		}
		rx, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return netCounters{}, fmt.Errorf("parsing rx bytes: %w", err)
		}
		tx, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			return netCounters{}, fmt.Errorf("parsing tx bytes: %w", err)
		}
		return netCounters{rxBytes: rx, txBytes: tx}, nil
	}
	return netCounters{}, fmt.Errorf("interface %s not found in /proc/net/dev", iface)
}
