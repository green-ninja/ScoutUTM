package stats

import (
	"os/exec"
	"strconv"
	"strings"
)

// dnsStats holds counters from unbound-control stats_noreset.
type dnsStats struct {
	totalQueries uint64
	blockedRPZ   uint64
}

// readUnboundStats calls unbound-control stats_noreset and parses the output.
// Returns zero values if unbound-control is unavailable or fails.
func readUnboundStats() dnsStats {
	cmd := exec.Command("unbound-control", "stats_noreset")
	out, err := cmd.Output()
	if err != nil {
		return dnsStats{}
	}
	return parseUnboundStats(string(out))
}

// parseUnboundStats extracts query and block counts from unbound-control output.
// Format is "key=value" lines, e.g.:
//
//	total.num.queries=12345
//	num.answer.rcode.NXDOMAIN=42
func parseUnboundStats(output string) dnsStats {
	var s dnsStats
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "total.num.queries":
			s.totalQueries = val
		case "num.answer.rcode.NXDOMAIN":
			s.blockedRPZ = val
		}
	}
	return s
}
