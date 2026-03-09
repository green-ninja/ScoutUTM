package update

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// UpdateInfo describes an available firmware update from the server.
type UpdateInfo struct {
	Version     string `json:"version"`
	Channel     string `json:"channel"`
	Description string `json:"description"`
	BundleURL   string `json:"bundle_url"`
	BundleSHA   string `json:"bundle_sha256"`
	Size        int64  `json:"size"`
}

// Checker polls the update server for new firmware versions.
//
// All communication uses HTTPS with TLS 1.2+ and certificate
// verification enabled. HTTP URLs are rejected to prevent
// downgrade attacks.
type Checker struct {
	serverURL string
	channel   string
	devMode   bool
	client    *http.Client
}

// NewChecker creates an update checker with a hardened HTTP client.
func NewChecker(serverURL, channel string, devMode bool) *Checker {
	return &Checker{
		serverURL: serverURL,
		channel:   channel,
		devMode:   devMode,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
					// InsecureSkipVerify is intentionally NOT set.
					// Certificate verification must remain enabled to
					// prevent MITM attacks on the update channel.
				},
				ForceAttemptHTTP2: true,
			},
		},
	}
}

// Check contacts the update server to see if a newer version is available.
// Returns nil if no update is available or if running in dev mode.
func (c *Checker) Check(currentVersion string) (*UpdateInfo, error) {
	if c.devMode {
		log.Println("[dev-mode] would check for updates")
		return nil, nil
	}

	// Enforce HTTPS — reject plaintext HTTP to prevent MITM.
	if !strings.HasPrefix(c.serverURL, "https://") {
		return nil, fmt.Errorf("update server URL must use HTTPS: %s", c.serverURL)
	}

	url := fmt.Sprintf("%s/api/v1/check?version=%s&channel=%s",
		strings.TrimRight(c.serverURL, "/"),
		currentVersion,
		c.channel,
	)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("contacting update server: %w", err)
	}
	defer resp.Body.Close()

	// Limit response body to 1MB to prevent resource exhaustion.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading update response: %w", err)
	}

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil, nil // No update available.
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update server returned %d: %s", resp.StatusCode, truncateOutput(body))
	}

	var info UpdateInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parsing update info: %w", err)
	}

	// Validate the bundle URL is also HTTPS.
	if info.BundleURL != "" && !strings.HasPrefix(info.BundleURL, "https://") {
		return nil, fmt.Errorf("bundle URL must use HTTPS: %s", info.BundleURL)
	}

	return &info, nil
}
