package netmon

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type arpEntry struct {
	IP        string
	MAC       string
	Interface string
}

// readARPTable parses /proc/net/arp for IP/MAC/interface entries.
// Skips incomplete entries (HW address 00:00:00:00:00:00, flags 0x0).
func readARPTable() ([]arpEntry, error) {
	return parseARPFile("/proc/net/arp")
}

func parseARPFile(path string) ([]arpEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening ARP table: %w", err)
	}
	defer f.Close()

	return parseARP(bufio.NewScanner(f))
}

// parseARP reads ARP entries from a scanner.
// Format: IP address, HW type, Flags, HW address, Mask, Device
func parseARP(scanner *bufio.Scanner) ([]arpEntry, error) {
	var entries []arpEntry

	// Skip header line.
	if !scanner.Scan() {
		return entries, nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		ip := fields[0]
		flags := fields[2]
		mac := fields[3]
		iface := fields[5]

		// Skip incomplete entries (flags 0x0) and zero MACs.
		if flags == "0x0" || mac == "00:00:00:00:00:00" {
			continue
		}

		entries = append(entries, arpEntry{
			IP:        ip,
			MAC:       strings.ToLower(mac),
			Interface: iface,
		})
	}

	return entries, scanner.Err()
}
