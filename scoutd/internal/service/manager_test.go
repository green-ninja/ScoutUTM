package service

import (
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func TestReloadDaemon_StubAll(t *testing.T) {
	m := NewManager(true)
	// StubAll mode should not execute anything — just log.
	if err := m.ReloadDaemon(Nftables); err != nil {
		t.Errorf("ReloadDaemon: %v", err)
	}
	if err := m.ReloadDaemon(Unbound); err != nil {
		t.Errorf("ReloadDaemon: %v", err)
	}
}

func TestReloadDaemons_StubAll(t *testing.T) {
	m := NewManager(true)
	daemons := []string{Chrony, Nftables, Dnsmasq}
	if err := m.ReloadDaemons(daemons); err != nil {
		t.Errorf("ReloadDaemons: %v", err)
	}
}

func TestReloadDaemons_Ordering(t *testing.T) {
	// Verify that ReloadOrder is correct per CLAUDE.md spec.
	expected := []string{Nftables, Dnsmasq, Unbound, Suricata, WireGuard, Chrony}
	if len(ReloadOrder) != len(expected) {
		t.Fatalf("ReloadOrder length = %d, want %d", len(ReloadOrder), len(expected))
	}
	for i, name := range expected {
		if ReloadOrder[i] != name {
			t.Errorf("ReloadOrder[%d] = %q, want %q", i, ReloadOrder[i], name)
		}
	}
}

func TestReloadDaemons_SubsetOrdering(t *testing.T) {
	m := NewManager(true)
	// Request chrony before nftables — should still reload nftables first.
	if err := m.ReloadDaemons([]string{Chrony, Nftables}); err != nil {
		t.Errorf("ReloadDaemons: %v", err)
	}
}

func TestHealthCheck_StubAll(t *testing.T) {
	m := NewManager(true)
	// StubAll health check should always pass.
	if err := m.HealthCheck([]string{Nftables, Dnsmasq, Chrony}); err != nil {
		t.Errorf("HealthCheck: %v", err)
	}
}


func TestSuricataUpdate_StubAll(t *testing.T) {
	m := NewManager(true)
	// StubAll mode should not execute anything — just log.
	if err := m.SuricataUpdate([]string{"et-open"}); err != nil {
		t.Errorf("SuricataUpdate: %v", err)
	}
}

func TestSuricataUpdate_StubAll_WithAbuseCH(t *testing.T) {
	m := NewManager(true)
	if err := m.SuricataUpdate([]string{"et-open", "abuse-ch"}); err != nil {
		t.Errorf("SuricataUpdate with abuse-ch: %v", err)
	}
}

func TestSyncVLANs_StubAll(t *testing.T) {
	m := NewManager(true)
	old := []config.VLANConfig{}
	new := []config.VLANConfig{
		{ID: 20, Parent: "eth1", Zone: "iot", Subnet: "192.168.20.0/24"},
	}
	if err := m.SyncVLANs(old, new); err != nil {
		t.Errorf("SyncVLANs create: %v", err)
	}
	// Reverse: delete
	if err := m.SyncVLANs(new, old); err != nil {
		t.Errorf("SyncVLANs delete: %v", err)
	}
}

func TestSyncVLANs_StubAll_NoChange(t *testing.T) {
	m := NewManager(true)
	vlans := []config.VLANConfig{
		{ID: 20, Parent: "eth1", Zone: "iot", Subnet: "192.168.20.0/24"},
	}
	if err := m.SyncVLANs(vlans, vlans); err != nil {
		t.Errorf("SyncVLANs no-change: %v", err)
	}
}

func TestGatewayAddr(t *testing.T) {
	tests := []struct {
		subnet string
		want   string
	}{
		{"192.168.20.0/24", "192.168.20.1/24"},
		{"10.0.0.0/8", "10.0.0.1/8"},
		{"172.16.30.0/24", "172.16.30.1/24"},
	}
	for _, tc := range tests {
		got, err := gatewayAddr(tc.subnet)
		if err != nil {
			t.Errorf("gatewayAddr(%q): %v", tc.subnet, err)
			continue
		}
		if got != tc.want {
			t.Errorf("gatewayAddr(%q) = %q, want %q", tc.subnet, got, tc.want)
		}
	}
}

func TestGatewayAddr_Invalid(t *testing.T) {
	_, err := gatewayAddr("not-a-cidr")
	if err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

func TestDiffAddedRemoved(t *testing.T) {
	old := []config.VLANConfig{
		{ID: 10, Parent: "eth1"},
		{ID: 20, Parent: "eth1"},
	}
	new := []config.VLANConfig{
		{ID: 20, Parent: "eth1"},
		{ID: 30, Parent: "eth1"},
	}

	added := diffAdded(old, new)
	if len(added) != 1 || added[0].ID != 30 {
		t.Errorf("diffAdded: got %v, want [{ID:30}]", added)
	}

	removed := diffRemoved(old, new)
	if len(removed) != 1 || removed[0].ID != 10 {
		t.Errorf("diffRemoved: got %v, want [{ID:10}]", removed)
	}
}

func TestVlanIfName(t *testing.T) {
	v := config.VLANConfig{ID: 20, Parent: "eth1"}
	if got := vlanIfName(v); got != "eth1.20" {
		t.Errorf("vlanIfName = %q, want %q", got, "eth1.20")
	}
}
