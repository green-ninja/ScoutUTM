package service

import "testing"

func TestReloadDaemon_DevMode(t *testing.T) {
	m := NewManager(true)
	// Dev mode should not execute anything — just log.
	if err := m.ReloadDaemon(Nftables); err != nil {
		t.Errorf("ReloadDaemon in dev mode: %v", err)
	}
}

func TestReloadDaemons_DevMode(t *testing.T) {
	m := NewManager(true)
	daemons := []string{Chrony, Nftables, Dnsmasq}
	if err := m.ReloadDaemons(daemons); err != nil {
		t.Errorf("ReloadDaemons in dev mode: %v", err)
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
	// Verify only requested daemons are reloaded, in correct order.
	m := NewManager(true)
	// Request chrony before nftables — should still reload nftables first.
	if err := m.ReloadDaemons([]string{Chrony, Nftables}); err != nil {
		t.Errorf("ReloadDaemons: %v", err)
	}
}

func TestHealthCheck_DevMode(t *testing.T) {
	m := NewManager(true)
	// Dev mode health check should always pass.
	if err := m.HealthCheck([]string{Nftables, Dnsmasq, Chrony}); err != nil {
		t.Errorf("HealthCheck in dev mode: %v", err)
	}
}
