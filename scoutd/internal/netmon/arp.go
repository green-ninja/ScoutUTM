package netmon

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type neighEntry struct {
	IP    string
	MAC   string
	Iface string
	State string // REACHABLE, STALE, DELAY, PROBE, FAILED, INCOMPLETE
}

// readNeighTable runs `ip -4 neigh show` and parses the output.
// Unlike /proc/net/arp, this includes NUD state so we can distinguish
// REACHABLE devices from STALE entries that may have gone offline.
func readNeighTable() ([]neighEntry, error) {
	cmd := exec.Command("ip", "-4", "neigh", "show")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ip neigh: %w", err)
	}
	return parseNeigh(bufio.NewScanner(strings.NewReader(string(out))))
}

// parseNeighFile reads ip neigh format from a file (for testing).
func parseNeighFile(path string) ([]neighEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening neigh file: %w", err)
	}
	defer f.Close()
	return parseNeigh(bufio.NewScanner(f))
}

// parseNeigh parses `ip -4 neigh show` output.
// Format: <ip> dev <iface> lladdr <mac> <STATE>
// Or:     <ip> dev <iface> <STATE> (no lladdr for FAILED/INCOMPLETE)
func parseNeigh(scanner *bufio.Scanner) ([]neighEntry, error) {
	var entries []neighEntry

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		ip := fields[0]

		// Find "dev" keyword.
		var iface string
		for i, f := range fields {
			if f == "dev" && i+1 < len(fields) {
				iface = fields[i+1]
				break
			}
		}

		// Find "lladdr" keyword.
		var mac string
		for i, f := range fields {
			if f == "lladdr" && i+1 < len(fields) {
				mac = strings.ToLower(fields[i+1])
				break
			}
		}

		// State is the last field.
		state := fields[len(fields)-1]

		// Skip entries without a resolved MAC (FAILED/INCOMPLETE).
		if mac == "" {
			continue
		}

		entries = append(entries, neighEntry{
			IP:    ip,
			MAC:   mac,
			Iface: iface,
			State: state,
		})
	}

	return entries, scanner.Err()
}
