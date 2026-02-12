package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func TestRenderWireGuard_Disabled(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.VPN.WireGuard.Enabled = false

	if err := e.RenderWireGuard(cfg); err != nil {
		t.Fatalf("RenderWireGuard: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(e.outputDir, "wireguard.conf"))
	if err != nil {
		t.Fatalf("reading wireguard.conf: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "[Interface]") {
		t.Error("disabled WireGuard should not have [Interface] section")
	}
}

func TestRenderWireGuard_Enabled(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.VPN.WireGuard.Enabled = true
	cfg.VPN.WireGuard.PrivateKey = "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY="
	cfg.VPN.WireGuard.ListenPort = 51820
	cfg.VPN.WireGuard.Address = "10.0.0.1/24"

	if err := e.RenderWireGuard(cfg); err != nil {
		t.Fatalf("RenderWireGuard: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(e.outputDir, "wireguard.conf"))
	if err != nil {
		t.Fatalf("reading wireguard.conf: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "[Interface]") {
		t.Error("expected [Interface] section")
	}
	if !strings.Contains(content, "ListenPort = 51820") {
		t.Error("expected ListenPort = 51820")
	}
	if !strings.Contains(content, "Address = 10.0.0.1/24") {
		t.Error("expected Address = 10.0.0.1/24")
	}
	if !strings.Contains(content, "PrivateKey = YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=") {
		t.Error("expected PrivateKey in output")
	}
}

func TestRenderWireGuard_WithPeers(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.VPN.WireGuard.Enabled = true
	cfg.VPN.WireGuard.PrivateKey = "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY="
	cfg.VPN.WireGuard.ListenPort = 51820
	cfg.VPN.WireGuard.Address = "10.0.0.1/24"
	cfg.VPN.WireGuard.Peers = []config.WireGuardPeer{
		{Name: "phone", PublicKey: "cGhvbmVwdWJrZXkxMjM0NTY3ODkwMTIzNDU2Nzg5MDEy", AllowedIPs: []string{"10.0.0.2/32"}, TunnelIP: "10.0.0.2"},
		{Name: "laptop", PublicKey: "bGFwdG9wcHVia2V5MTIzNDU2Nzg5MDEyMzQ1Njc4OTAx", AllowedIPs: []string{"10.0.0.3/32"}, TunnelIP: "10.0.0.3"},
	}

	if err := e.RenderWireGuard(cfg); err != nil {
		t.Fatalf("RenderWireGuard: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(e.outputDir, "wireguard.conf"))
	if err != nil {
		t.Fatalf("reading wireguard.conf: %v", err)
	}
	content := string(data)

	peerCount := strings.Count(content, "[Peer]")
	if peerCount != 2 {
		t.Errorf("expected 2 [Peer] blocks, got %d", peerCount)
	}
	if !strings.Contains(content, "PublicKey = cGhvbmVwdWJrZXkxMjM0NTY3ODkwMTIzNDU2Nzg5MDEy") {
		t.Error("expected phone peer public key")
	}
	if !strings.Contains(content, "PublicKey = bGFwdG9wcHVia2V5MTIzNDU2Nzg5MDEyMzQ1Njc4OTAx") {
		t.Error("expected laptop peer public key")
	}
	if !strings.Contains(content, "AllowedIPs = 10.0.0.2/32") {
		t.Error("expected phone AllowedIPs")
	}
}
