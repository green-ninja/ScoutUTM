package netmon

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type dhcpLease struct {
	Expiry   int64
	MAC      string
	IP       string
	Hostname string
}

// readDHCPLeases parses a dnsmasq lease file.
// Format: <expiry_epoch> <mac> <ip> <hostname> <client-id>
func readDHCPLeases(path string) ([]dhcpLease, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No lease file yet.
		}
		return nil, fmt.Errorf("opening lease file: %w", err)
	}
	defer f.Close()

	return parseLeases(bufio.NewScanner(f))
}

func parseLeases(scanner *bufio.Scanner) ([]dhcpLease, error) {
	var leases []dhcpLease

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		expiry, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}

		hostname := fields[3]
		if hostname == "*" {
			hostname = ""
		}

		leases = append(leases, dhcpLease{
			Expiry:   expiry,
			MAC:      strings.ToLower(fields[1]),
			IP:       fields[2],
			Hostname: hostname,
		})
	}

	return leases, scanner.Err()
}
