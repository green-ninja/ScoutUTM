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

func TestRenderNftables_AntiSpoofing_Production(t *testing.T) {
	tmplDir := filepath.Join("..", "..", "templates")
	outputDir := t.TempDir()
	e := NewEngine(tmplDir, outputDir, false) // production mode
	cfg := testConfig()

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(outputDir, "nftables.conf"))
	content := string(data)

	if !strings.Contains(content, "set bogons") {
		t.Error("expected bogons set in production mode")
	}
	if !strings.Contains(content, "ip saddr @bogons drop") {
		t.Error("expected anti-spoofing rule in production mode")
	}
}

func TestRenderNftables_AntiSpoofing_DevMode(t *testing.T) {
	e := testEngine(t) // dev mode
	cfg := testConfig()

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	// Anti-spoofing should NOT be present in dev mode (breaks VirtualBox NAT).
	if strings.Contains(content, "set bogons") {
		t.Error("bogons set should not exist in dev mode")
	}
	if strings.Contains(content, "ip saddr @bogons") {
		t.Error("anti-spoofing rule should not exist in dev mode")
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
	if !strings.Contains(content, "queue num 0-3 fanout") {
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

func TestRenderNftables_CustomRules_BeforeForwardings(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.Firewall.Rules = []config.FirewallRule{
		{Name: "Block HTTP", Src: "lan", Dest: "wan", Proto: "tcp", Port: 80, Action: "drop"},
	}

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	// Custom rules must appear before zone forwardings so deny rules take effect.
	rulePos := strings.Index(content, "tcp dport 80 drop")
	fwdPos := strings.Index(content, "iifname @lan_devices oifname @wan_devices accept")
	if rulePos < 0 {
		t.Fatal("custom rule not found in output")
	}
	if fwdPos < 0 {
		t.Fatal("forwarding rule not found in output")
	}
	if rulePos > fwdPos {
		t.Error("custom rules must render before zone forwardings")
	}
}

func TestRenderNftables_CustomRules_AnyProtoWithPort(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.Firewall.Rules = []config.FirewallRule{
		{Name: "Block HTTP any proto", Src: "lan", Dest: "wan", Proto: "", Port: 80, Action: "drop"},
	}

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	// When proto is empty but port is set, should match both tcp and udp.
	if !strings.Contains(content, "meta l4proto { tcp, udp } th dport 80 drop") {
		t.Errorf("expected 'meta l4proto { tcp, udp } th dport 80 drop', got:\n%s", content)
	}
}

func TestRenderNftables_CustomRules_ProtoOnlyNoPort(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.Firewall.Rules = []config.FirewallRule{
		{Name: "Block ICMP", Src: "lan", Dest: "wan", Proto: "icmp", Port: 0, Action: "drop"},
	}

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	if !strings.Contains(content, "meta l4proto icmp drop") {
		t.Errorf("expected 'meta l4proto icmp drop', got:\n%s", content)
	}
}

func TestRenderNftables_DefaultBlockRules(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig() // DefaultConfig() now has default block rules

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	// Default config should block Telnet, SMB, NetBIOS, RPC.
	if !strings.Contains(content, "tcp dport 23 drop") {
		t.Error("expected default Telnet block rule")
	}
	if !strings.Contains(content, "tcp dport 445 drop") {
		t.Error("expected default SMB block rule")
	}
	if !strings.Contains(content, "tcp dport 139 drop") {
		t.Error("expected default NetBIOS block rule")
	}
	if !strings.Contains(content, "tcp dport 135 drop") {
		t.Error("expected default RPC block rule")
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

func TestRenderNftables_WireGuardEnabled(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.VPN.WireGuard.Enabled = true
	cfg.VPN.WireGuard.ListenPort = 51820

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	if !strings.Contains(content, "udp dport 51820 accept") {
		t.Error("expected WireGuard UDP port accept rule in WAN input")
	}
}

func TestRenderNftables_WireGuardDisabled(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.VPN.WireGuard.Enabled = false

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	if strings.Contains(content, "udp dport 51820 accept") {
		t.Error("WireGuard disabled should not have UDP port accept rule")
	}
}

func TestRenderNftables_WireGuardVPNInputChain(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.VPN.WireGuard.Enabled = true
	cfg.VPN.WireGuard.ListenPort = 51820

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	// VPN input chain should exist.
	if !strings.Contains(content, "chain input_vpn") {
		t.Error("expected input_vpn chain when WireGuard enabled")
	}

	// Jump to input_vpn from main input chain.
	if !strings.Contains(content, "iifname wg0 jump input_vpn") {
		t.Error("expected jump to input_vpn for wg0")
	}

	// DNS redirect for VPN clients.
	if !strings.Contains(content, "iifname wg0 udp dport 53 redirect") {
		t.Error("expected DNS redirect for VPN clients (UDP)")
	}
	if !strings.Contains(content, "iifname wg0 tcp dport 53 redirect") {
		t.Error("expected DNS redirect for VPN clients (TCP)")
	}
}

func TestRenderNftables_WireGuardVPNForwarding(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.VPN.WireGuard.Enabled = true
	cfg.VPN.WireGuard.ListenPort = 51820

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	// VPN clients should be able to forward to WAN.
	if !strings.Contains(content, "iifname wg0 oifname @wan_devices accept") {
		t.Error("expected VPN→WAN forward rule")
	}

	// VPN clients should be able to forward to LAN.
	if !strings.Contains(content, "iifname wg0 oifname @lan_devices accept") {
		t.Error("expected VPN→LAN forward rule")
	}
}

func TestRenderNftables_WireGuardDisabled_NoVPNRules(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.VPN.WireGuard.Enabled = false

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	if strings.Contains(content, "chain input_vpn") {
		t.Error("VPN disabled should not have input_vpn chain")
	}
	if strings.Contains(content, "iifname wg0") {
		t.Error("VPN disabled should not have any wg0 rules")
	}
}

func TestRenderNftables_WireGuardCustomPort(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.VPN.WireGuard.Enabled = true
	cfg.VPN.WireGuard.ListenPort = 12345

	if err := e.RenderNftables(cfg); err != nil {
		t.Fatalf("RenderNftables: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "nftables.conf"))
	content := string(data)

	if !strings.Contains(content, "udp dport 12345 accept") {
		t.Error("expected WireGuard custom port 12345 in WAN input")
	}
}
