package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func TestRenderUnbound_Default(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderUnbound(cfg); err != nil {
		t.Fatalf("RenderUnbound: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	if err != nil {
		t.Fatalf("reading unbound.conf: %v", err)
	}
	content := string(data)

	// DNSSEC auto-trust-anchor.
	if !strings.Contains(content, "auto-trust-anchor-file") {
		t.Error("expected DNSSEC auto-trust-anchor-file")
	}

	// Access control for LAN.
	if !strings.Contains(content, "access-control: 192.168.1.0/24 allow") {
		t.Error("expected access-control for LAN subnet")
	}

	// Localhost allowed.
	if !strings.Contains(content, "access-control: 127.0.0.0/8 allow") {
		t.Error("expected access-control for localhost")
	}

	// Default upstreams.
	if !strings.Contains(content, "forward-addr: 1.1.1.1@53") {
		t.Error("expected forward-addr for 1.1.1.1")
	}
	if !strings.Contains(content, "forward-addr: 9.9.9.9@53") {
		t.Error("expected forward-addr for 9.9.9.9")
	}

	// Query logging on by default.
	if !strings.Contains(content, "log-queries: yes") {
		t.Error("expected log-queries: yes")
	}

	// No TLS by default.
	if strings.Contains(content, "forward-tls-upstream") {
		t.Error("unexpected forward-tls-upstream in default config")
	}
}

func TestRenderUnbound_DOT(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.DNS.UseDOT = true

	if err := e.RenderUnbound(cfg); err != nil {
		t.Fatalf("RenderUnbound: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	content := string(data)

	if !strings.Contains(content, "forward-tls-upstream: yes") {
		t.Error("expected forward-tls-upstream: yes with DoT enabled")
	}

	// Upstream port should be 853.
	if !strings.Contains(content, "@853") {
		t.Error("expected upstream port 853 with DoT enabled")
	}
}

func TestRenderUnbound_Blocklists(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.DNS.BlockMalware = true
	cfg.DNS.BlockPhishing = true
	cfg.DNS.BlockAds = true

	if err := e.RenderUnbound(cfg); err != nil {
		t.Fatalf("RenderUnbound: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	content := string(data)

	if !strings.Contains(content, "rpz-malware") {
		t.Error("expected rpz-malware zone")
	}
	if !strings.Contains(content, "rpz-phishing") {
		t.Error("expected rpz-phishing zone")
	}
	if !strings.Contains(content, "rpz-ads") {
		t.Error("expected rpz-ads zone")
	}

	// RPZ zones must be top-level clauses, not inside server: block.
	if strings.Contains(content, "auth-zone") {
		t.Error("auth-zone should not appear — RPZ uses rpz: clause only")
	}
	// Verify rpz: clause has correct structure.
	if !strings.Contains(content, "\nrpz:") {
		t.Error("rpz: should be a top-level clause (no leading spaces)")
	}
}

func TestRenderUnbound_CustomBlock(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.DNS.CustomBlock = []string{"badsite.com", "malware.org"}

	if err := e.RenderUnbound(cfg); err != nil {
		t.Fatalf("RenderUnbound: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	content := string(data)

	if !strings.Contains(content, `local-zone: "badsite.com" always_nxdomain`) {
		t.Error("expected local-zone for badsite.com")
	}
	if !strings.Contains(content, `local-zone: "malware.org" always_nxdomain`) {
		t.Error("expected local-zone for malware.org")
	}
}

func TestRenderUnbound_CustomAllow(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.DNS.CustomAllow = []string{"allowed.com", "safe.org"}

	if err := e.RenderUnbound(cfg); err != nil {
		t.Fatalf("RenderUnbound: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	content := string(data)

	if !strings.Contains(content, `local-zone: "allowed.com" transparent`) {
		t.Error("expected local-zone transparent for allowed.com")
	}
	if !strings.Contains(content, `local-zone: "safe.org" transparent`) {
		t.Error("expected local-zone transparent for safe.org")
	}
}

func TestRenderUnbound_VLAN(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.Network.VLANs = []config.VLANConfig{
		{
			ID:     100,
			Parent: "eth1",
			Zone:   "guest",
			Subnet: "10.0.100.1/24",
			DHCPRange: config.DHCPRange{
				Start: "10.0.100.100",
				End:   "10.0.100.200",
			},
		},
	}

	if err := e.RenderUnbound(cfg); err != nil {
		t.Fatalf("RenderUnbound: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	content := string(data)

	// LAN subnet.
	if !strings.Contains(content, "access-control: 192.168.1.0/24 allow") {
		t.Error("expected access-control for LAN subnet")
	}

	// VLAN subnet.
	if !strings.Contains(content, "access-control: 10.0.100.0/24 allow") {
		t.Error("expected access-control for VLAN subnet")
	}
}

func TestRenderUnbound_NoBlocklists(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.DNS.BlockMalware = false
	cfg.DNS.BlockPhishing = false
	cfg.DNS.BlockAds = false

	if err := e.RenderUnbound(cfg); err != nil {
		t.Fatalf("RenderUnbound: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	content := string(data)

	if strings.Contains(content, "rpz-") {
		t.Error("unexpected RPZ zone when all blocklists disabled")
	}
}
