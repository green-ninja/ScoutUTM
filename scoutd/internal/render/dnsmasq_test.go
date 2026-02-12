package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func TestRenderDnsmasq_Default(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderDnsmasq(cfg); err != nil {
		t.Fatalf("RenderDnsmasq: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(e.outputDir, "dnsmasq.conf"))
	if err != nil {
		t.Fatalf("reading dnsmasq.conf: %v", err)
	}
	content := string(data)

	// DNS disabled.
	if !strings.Contains(content, "port=0") {
		t.Error("expected port=0 (DNS disabled)")
	}

	// Bind interfaces.
	if !strings.Contains(content, "bind-interfaces") {
		t.Error("expected bind-interfaces")
	}

	// DHCP authoritative.
	if !strings.Contains(content, "dhcp-authoritative") {
		t.Error("expected dhcp-authoritative")
	}

	// Lease file.
	if !strings.Contains(content, "/var/lib/scout/dnsmasq.leases") {
		t.Error("expected lease file path")
	}

	// LAN interface binding.
	if !strings.Contains(content, "interface=eth1") {
		t.Error("expected interface=eth1")
	}

	// DHCP range for LAN.
	if !strings.Contains(content, "dhcp-range=eth1,192.168.1.100,192.168.1.253,255.255.255.0,12h") {
		t.Error("expected DHCP range for eth1")
	}

	// Gateway option (option 3 = gateway).
	if !strings.Contains(content, "dhcp-option=eth1,3,192.168.1.1") {
		t.Error("expected gateway DHCP option for eth1")
	}

	// DNS option (option 6 = DNS, points to interface IP → Unbound).
	if !strings.Contains(content, "dhcp-option=eth1,6,192.168.1.1") {
		t.Error("expected DNS DHCP option for eth1")
	}
}

func TestRenderDnsmasq_NoWAN(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderDnsmasq(cfg); err != nil {
		t.Fatalf("RenderDnsmasq: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "dnsmasq.conf"))
	content := string(data)

	// WAN interface (eth0) should NOT appear.
	if strings.Contains(content, "interface=eth0") {
		t.Error("WAN interface eth0 should not have DHCP")
	}
}

func TestRenderDnsmasq_VLAN(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.Network.VLANs = append(cfg.Network.VLANs, config.VLANConfig{
		ID:     100,
		Parent: "eth1",
		Zone:   "guest",
		Subnet: "10.0.100.1/24",
	})

	if err := e.RenderDnsmasq(cfg); err != nil {
		t.Fatalf("RenderDnsmasq: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "dnsmasq.conf"))
	content := string(data)

	// VLAN sub-interface.
	if !strings.Contains(content, "interface=eth1.100") {
		t.Error("expected VLAN interface eth1.100")
	}

	// DHCP range for VLAN.
	if !strings.Contains(content, "dhcp-range=eth1.100,10.0.100.100,10.0.100.253,255.255.255.0,12h") {
		t.Error("expected DHCP range for VLAN")
	}
}
