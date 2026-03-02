package stats

import (
	"os"
	"strconv"
	"strings"
)

// readConntrackCount reads the current connection count from
// /proc/sys/net/netfilter/nf_conntrack_count. Returns 0 if unavailable.
func readConntrackCount() uint64 {
	data, err := os.ReadFile("/proc/sys/net/netfilter/nf_conntrack_count")
	if err != nil {
		return 0
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
