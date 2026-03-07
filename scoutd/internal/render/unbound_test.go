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

	// Reply logging on by default.
	if !strings.Contains(content, "log-replies: yes") {
		t.Error("expected log-replies: yes")
	}

	// No TLS by default.
	if strings.Contains(content, "forward-tls-upstream") {
		t.Error("unexpected forward-tls-upstream in default config")
	}

	// Default config has malware+phishing RPZ, so respip module is needed.
	if !strings.Contains(content, `module-config: "respip validator iterator"`) {
		t.Error("expected module-config with respip for RPZ")
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

func parentalTestConfig() *config.Config {
	cfg := testConfig()
	cfg.Parental.Profiles = []config.ParentalProfile{
		{
			ID:              "prof1",
			Name:            "Kids",
			BlockCategories: []string{"adult", "gambling"},
			SafeSearch:      true,
			YoutubeRestricted: true,
		},
		{
			ID:              "prof2",
			Name:            "Teens",
			BlockCategories: []string{"adult"},
			SafeSearch:      true,
			YoutubeRestricted: false,
		},
	}
	cfg.Parental.Devices = []config.ParentalDevice{
		{MAC: "aa:bb:cc:dd:ee:01", ProfileID: "prof1", IP: "192.168.1.101"},
		{MAC: "aa:bb:cc:dd:ee:02", ProfileID: "prof2", IP: "192.168.1.102"},
	}
	return cfg
}

func TestRenderUnbound_ParentalTags(t *testing.T) {
	e := testEngine(t)
	cfg := parentalTestConfig()

	if err := e.RenderUnbound(cfg); err != nil {
		t.Fatalf("RenderUnbound: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	content := string(data)

	// Tag definition should include both profile tags.
	if !strings.Contains(content, `define-tag: "`) {
		t.Error("expected define-tag directive")
	}
	if !strings.Contains(content, "kids") || !strings.Contains(content, "teens") {
		t.Error("expected both profile tags in define-tag")
	}

	// Device IP→tag mappings.
	if !strings.Contains(content, `access-control-tag: 192.168.1.101/32 "kids"`) {
		t.Error("expected access-control-tag for kids device")
	}
	if !strings.Contains(content, `access-control-tag: 192.168.1.102/32 "teens"`) {
		t.Error("expected access-control-tag for teens device")
	}
}

func TestRenderUnbound_ParentalRPZ(t *testing.T) {
	e := testEngine(t)
	cfg := parentalTestConfig()

	if err := e.RenderUnbound(cfg); err != nil {
		t.Fatalf("RenderUnbound: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	content := string(data)

	// Adult RPZ should have both tags.
	if !strings.Contains(content, "rpz-parental-adult") {
		t.Error("expected rpz-parental-adult zone")
	}
	// Gambling RPZ should only have kids tag.
	if !strings.Contains(content, "rpz-parental-gambling") {
		t.Error("expected rpz-parental-gambling zone")
	}
	// Parental RPZ zones should have tags: directive.
	if !strings.Contains(content, `tags: "`) {
		t.Error("expected tags directive on parental RPZ zones")
	}
	// RPZ requires respip module.
	if !strings.Contains(content, `module-config: "respip validator iterator"`) {
		t.Error("expected module-config with respip for RPZ")
	}
}

func TestRenderUnbound_SafeSearch(t *testing.T) {
	e := testEngine(t)
	cfg := parentalTestConfig()

	if err := e.RenderUnbound(cfg); err != nil {
		t.Fatalf("RenderUnbound: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	content := string(data)

	// Safe search CNAMEs.
	if !strings.Contains(content, "forcesafesearch.google.com") {
		t.Error("expected Google safe search CNAME")
	}
	if !strings.Contains(content, "strict.bing.com") {
		t.Error("expected Bing safe search CNAME")
	}
	if !strings.Contains(content, "safe.duckduckgo.com") {
		t.Error("expected DuckDuckGo safe search CNAME")
	}
	// YouTube restricted mode.
	if !strings.Contains(content, "restrict.youtube.com") {
		t.Error("expected YouTube restricted mode CNAME")
	}
	// local-zone-tag should exist for safe search.
	if !strings.Contains(content, `local-zone-tag: "www.google.com"`) {
		t.Error("expected local-zone-tag for Google safe search")
	}
}

func TestRenderUnbound_NoParentalWithoutDevices(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	// No parental devices or profiles.

	if err := e.RenderUnbound(cfg); err != nil {
		t.Fatalf("RenderUnbound: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	content := string(data)

	if strings.Contains(content, "define-tag") {
		t.Error("should not have define-tag without parental devices")
	}
	if strings.Contains(content, "access-control-tag") {
		t.Error("should not have access-control-tag without parental devices")
	}
	if strings.Contains(content, "rpz-parental") {
		t.Error("should not have parental RPZ zones without parental devices")
	}
}

func TestRenderUnbound_ScheduleBlocked(t *testing.T) {
	e := testEngine(t)
	cfg := parentalTestConfig()

	blockedIPs := []string{"192.168.1.101"}
	if err := e.RenderUnboundWithSchedule(cfg, blockedIPs); err != nil {
		t.Fatalf("RenderUnboundWithSchedule: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "unbound", "unbound.conf"))
	content := string(data)

	if !strings.Contains(content, "access-control: 192.168.1.101/32 deny") {
		t.Error("expected access-control deny for schedule-blocked device")
	}
}

func TestSanitizeTag(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Kids", "kids"},
		{"Young Teens", "young_teens"},
		{"My-Profile!", "myprofile"},
		{"  Spaced  ", "spaced"},
		{"", "profile"},
		{"ABC 123", "abc_123"},
	}

	for _, tt := range tests {
		got := sanitizeTag(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeTag(%q) = %q, want %q", tt.input, got, tt.want)
		}
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
	if strings.Contains(content, "module-config") {
		t.Error("unexpected module-config without RPZ zones")
	}
}
