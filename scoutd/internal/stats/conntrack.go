package stats

import (
	"os"
	"os/exec"
	"sort"
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

// Connection represents a single conntrack entry.
type Connection struct {
	Proto   string `json:"proto"`
	State   string `json:"state"`
	SrcIP   string `json:"src_ip"`
	SrcPort uint16 `json:"src_port,omitempty"`
	DstIP   string `json:"dst_ip"`
	DstPort uint16 `json:"dst_port,omitempty"`
	Bytes   uint64 `json:"bytes"`
}

// conntrackCmd is the command used to list connections; overridden in tests.
var conntrackCmd = "conntrack"

// ReadConnections lists active conntrack entries and returns the top
// connections sorted by byte count (descending), capped at limit.
// Uses `conntrack -L` (netlink-based, works on all modern kernels)
// with fallback to /proc/net/nf_conntrack.
func ReadConnections(limit int) []Connection {
	var lines []string

	// Primary: conntrack -L (outputs to stdout, count to stderr).
	out, err := exec.Command(conntrackCmd, "-L", "-o", "extended").Output()
	if err == nil && len(out) > 0 {
		lines = strings.Split(strings.TrimSpace(string(out)), "\n")
	} else {
		// Fallback: proc file (requires CONFIG_NF_CONNTRACK_PROCFS).
		data, err := os.ReadFile("/proc/net/nf_conntrack")
		if err != nil {
			return nil
		}
		lines = strings.Split(strings.TrimSpace(string(data)), "\n")
	}

	conns := make([]Connection, 0, len(lines))
	for _, line := range lines {
		if c, ok := parseConntrackLine(line); ok {
			conns = append(conns, c)
		}
	}
	sort.Slice(conns, func(i, j int) bool {
		return conns[i].Bytes > conns[j].Bytes
	})
	if limit > 0 && len(conns) > limit {
		conns = conns[:limit]
	}
	return conns
}

// parseConntrackLine parses a single line from conntrack -L or /proc/net/nf_conntrack.
//
// conntrack -L format:
//
//	tcp  6 300 ESTABLISHED src=... dst=... sport=... dport=... bytes=... src=... ...
//
// /proc/net/nf_conntrack format:
//
//	ipv4  2 tcp  6 300 ESTABLISHED src=... dst=... sport=... dport=... bytes=... src=... ...
func parseConntrackLine(line string) (Connection, bool) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return Connection{}, false
	}

	// Find protocol name. In proc format it's field[2] ("tcp"), in conntrack -L
	// it's field[0] ("tcp"). Detect by checking if field[0] is "ipv4"/"ipv6".
	protoIdx := 0
	if fields[0] == "ipv4" || fields[0] == "ipv6" {
		protoIdx = 2 // proc format
	}
	if protoIdx+3 >= len(fields) {
		return Connection{}, false
	}

	c := Connection{
		Proto: fields[protoIdx],
	}

	// After proto name and proto number, there's a TTL, then optionally a state.
	// TCP has states (ESTABLISHED, TIME_WAIT, etc.); UDP/ICMP don't.
	stateIdx := protoIdx + 3
	if stateIdx < len(fields) && !strings.Contains(fields[stateIdx], "=") {
		c.State = fields[stateIdx]
	}

	// Parse key=value pairs from the first (original direction) tuple.
	// Stop at the second src= which begins the reply tuple.
	srcSeen := 0
	for _, f := range fields[protoIdx+2:] {
		k, v, ok := strings.Cut(f, "=")
		if !ok {
			continue
		}
		if k == "src" {
			srcSeen++
			if srcSeen > 1 {
				break // reply direction — stop
			}
			c.SrcIP = v
		} else if k == "dst" && srcSeen == 1 {
			c.DstIP = v
		} else if k == "sport" && srcSeen == 1 {
			if p, err := strconv.ParseUint(v, 10, 16); err == nil {
				c.SrcPort = uint16(p)
			}
		} else if k == "dport" && srcSeen == 1 {
			if p, err := strconv.ParseUint(v, 10, 16); err == nil {
				c.DstPort = uint16(p)
			}
		} else if k == "bytes" && srcSeen == 1 {
			if b, err := strconv.ParseUint(v, 10, 64); err == nil {
				c.Bytes = b
			}
		}
	}

	if c.SrcIP == "" || c.DstIP == "" {
		return Connection{}, false
	}
	return c, true
}
