package blocklist

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// StevenBlackURL is the unified hosts file with malware + phishing domains.
	StevenBlackURL = "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"

	// maxResponseSize limits the download to 50 MB to prevent resource exhaustion.
	maxResponseSize = 50 * 1024 * 1024
)

// FetchThreatsBlocklist downloads the Steven Black hosts file, converts it to
// RPZ zone format, and writes it to the blocklist directory. Returns the number
// of domains written.
func FetchThreatsBlocklist(dir string) (int, error) {
	if dir == "" {
		dir = blocklistDir
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("creating blocklist dir: %w", err)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	resp, err := client.Get(StevenBlackURL)
	if err != nil {
		return 0, fmt.Errorf("fetching blocklist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("blocklist returned HTTP %d", resp.StatusCode)
	}

	reader := io.LimitReader(resp.Body, maxResponseSize)
	domains := ParseHostsFile(reader)

	path := filepath.Join(dir, "rpz-threats.zone")
	if err := writeRPZZone(path, domains); err != nil {
		return 0, fmt.Errorf("writing RPZ zone: %w", err)
	}

	return len(domains), nil
}

// ParseHostsFile reads a hosts-format file (0.0.0.0 domain) and returns
// the list of unique domains, skipping comments and localhost entries.
func ParseHostsFile(r io.Reader) []string {
	seen := make(map[string]bool)
	var domains []string

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Steven Black format: "0.0.0.0 domain.com"
		ip := fields[0]
		if ip != "0.0.0.0" && ip != "127.0.0.1" {
			continue
		}

		domain := strings.ToLower(fields[1])

		// Skip localhost entries.
		if domain == "localhost" || domain == "localhost.localdomain" ||
			domain == "local" || domain == "0.0.0.0" {
			continue
		}

		if !seen[domain] {
			seen[domain] = true
			domains = append(domains, domain)
		}
	}

	return domains
}

// writeRPZZone writes an RPZ zone file with the given domains.
func writeRPZZone(path string, domains []string) error {
	var b strings.Builder
	b.WriteString(rpzHeader)
	for _, d := range domains {
		fmt.Fprintf(&b, "%s CNAME .\n", d)
		fmt.Fprintf(&b, "*.%s CNAME .\n", d)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}
