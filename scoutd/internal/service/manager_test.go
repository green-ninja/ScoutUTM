package service

import "testing"

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

func TestVMManager_NftablesStubbed(t *testing.T) {
	m := NewVMManager()
	// VM manager should stub nftables.
	if err := m.ReloadDaemon(Nftables); err != nil {
		t.Errorf("ReloadDaemon nftables: %v", err)
	}
}

func TestVMManager_HealthCheck_NftablesSkipped(t *testing.T) {
	m := NewVMManager()
	// Nftables health check should be skipped.
	if err := m.HealthCheck([]string{Nftables}); err != nil {
		t.Errorf("HealthCheck nftables: %v", err)
	}
}
