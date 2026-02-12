package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func TestRenderNftables_Default(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	if err != nil {
		t.Fatalf("reading nftables.conf: %v", err)
	}
	content := string(data)

	// Single table.
	if !strings.Contains(content, "table inet scout") {
		t.Error("expected 'table inet scout'")
	}

	// Zone device sets.
	if !strings.Contains(content, "set wan_devices") {
		t.Error("expected wan_devices set")
	}
	if !strings.Contains(content, "set lan_devices") {
		t.Error("expected lan_devices set")
	}

	// WAN interface in set.
	if !strings.Contains(content, "elements = { eth0 }") {
		t.Error("expected eth0 in wan_devices")
	}

	// LAN interface in set.
	if !strings.Contains(content, "elements = { eth1 }") {
		t.Error("expected eth1 in lan_devices")
	}
}

func TestRenderNftables_IPv6Drop(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	if !strings.Contains(content, "meta nfproto ipv6 drop") {
		t.Error("expected IPv6 drop rule")
	}
}

func TestRenderNftables_DNSRedirect(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	// DNS redirect for LAN (non-WAN) zones.
	if !strings.Contains(content, "iifname @lan_devices udp dport 53 redirect") {
		t.Error("expected DNS redirect for LAN (UDP)")
	}
	if !strings.Contains(content, "iifname @lan_devices tcp dport 53 redirect") {
		t.Error("expected DNS redirect for LAN (TCP)")
	}

	// WAN should NOT have DNS redirect.
	if strings.Contains(content, "iifname @wan_devices udp dport 53 redirect") {
		t.Error("WAN should not have DNS redirect")
	}
}

func TestRenderNftables_Masquerade(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	// WAN zone has masquerade=true.
	if !strings.Contains(content, "oifname @wan_devices masquerade") {
		t.Error("expected masquerade on WAN")
	}

	// LAN zone has masquerade=false — should not masquerade.
	if strings.Contains(content, "oifname @lan_devices masquerade") {
		t.Error("LAN should not have masquerade")
	}
}

func TestRenderNftables_IPSMode(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.IDS.Mode = "ips"

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	if !strings.Contains(content, "chain ips") {
		t.Error("expected IPS chain when mode=ips")
	}
	if !strings.Contains(content, "queue num 0-3 fanout,bypass") {
		t.Error("expected NFQUEUE rule in IPS chain")
	}
	if !strings.Contains(content, "jump ips") {
		t.Error("expected jump to IPS chain in forward")
	}
}

func TestRenderNftables_IDSMode_NoIPSChain(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.IDS.Mode = "ids"

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	if strings.Contains(content, "chain ips") {
		t.Error("IDS mode should not have IPS chain")
	}
}

func TestRenderNftables_PortForwards(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.Firewall.PortForwards = []config.PortForward{
		{Name: "HTTP Server", Proto: "tcp", SrcPort: 80, DestIP: "192.168.1.100", DestPort: 8080},
	}

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	if !strings.Contains(content, "dnat to 192.168.1.100:8080") {
		t.Error("expected DNAT rule for port forward")
	}
	if !strings.Contains(content, "tcp dport 80") {
		t.Error("expected source port 80 in DNAT rule")
	}
}

func TestRenderNftables_PortForwards_SingleDNATAccept(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.Firewall.PortForwards = []config.PortForward{
		{Name: "HTTP", Proto: "tcp", SrcPort: 80, DestIP: "192.168.1.100", DestPort: 8080},
		{Name: "HTTPS", Proto: "tcp", SrcPort: 443, DestIP: "192.168.1.100", DestPort: 8443},
		{Name: "SSH", Proto: "tcp", SrcPort: 2222, DestIP: "192.168.1.200", DestPort: 22},
	}

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	// Should have exactly one ct status dnat rule regardless of port forward count.
	count := strings.Count(content, "ct status dnat accept")
	if count != 1 {
		t.Errorf("expected 1 'ct status dnat accept' rule, got %d", count)
	}

	// All three DNAT rules should be present.
	if !strings.Contains(content, "dnat to 192.168.1.100:8080") {
		t.Error("missing HTTP DNAT rule")
	}
	if !strings.Contains(content, "dnat to 192.168.1.100:8443") {
		t.Error("missing HTTPS DNAT rule")
	}
	if !strings.Contains(content, "dnat to 192.168.1.200:22") {
		t.Error("missing SSH DNAT rule")
	}
}

func TestRenderNftables_CustomRules(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.Firewall.Rules = []config.FirewallRule{
		{Name: "Block SSH from LAN to WAN", Src: "lan", Dest: "wan", Proto: "tcp", Port: 22, Action: "drop"},
	}

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	if !strings.Contains(content, "tcp dport 22 drop") {
		t.Error("expected custom rule: tcp dport 22 drop")
	}
}

func TestRenderNftables_Forwarding(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	// Default config has lan→wan forwarding.
	if !strings.Contains(content, "iifname @lan_devices oifname @wan_devices accept") {
		t.Error("expected lan→wan forwarding rule")
	}
}

func TestBuildZoneInterfaceMap(t *testing.T) {
	cfg := testConfig()
	m := buildZoneInterfaceMap(cfg)

	if ifaces, ok := m["wan"]; !ok || len(ifaces) != 1 || ifaces[0] != "eth0" {
		t.Errorf("wan interfaces = %v, want [eth0]", m["wan"])
	}
	if ifaces, ok := m["lan"]; !ok || len(ifaces) != 1 || ifaces[0] != "eth1" {
		t.Errorf("lan interfaces = %v, want [eth1]", m["lan"])
	}
}

func TestBuildZoneInterfaceMap_WithVLAN(t *testing.T) {
	cfg := testConfig()
	cfg.Network.VLANs = []config.VLANConfig{
		{ID: 100, Parent: "eth1", Zone: "guest", Subnet: "10.0.100.1/24"},
	}

	m := buildZoneInterfaceMap(cfg)

	if ifaces, ok := m["guest"]; !ok || len(ifaces) != 1 || ifaces[0] != "eth1.100" {
		t.Errorf("guest interfaces = %v, want [eth1.100]", m["guest"])
	}
}
