package api

import (
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func TestApplyConfig_Success(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	cfg, err := srv.Config.Current().DeepCopy()
	if err != nil {
		t.Fatalf("DeepCopy: %v", err)
	}
	cfg.System.Hostname = "newhost"

	if err := srv.applyConfig(cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	// Config should be committed.
	current := srv.Config.Current()
	if current.System.Hostname != "newhost" {
		t.Errorf("hostname = %q, want newhost", current.System.Hostname)
	}
}

func TestApplyConfig_ValidationFailure(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	cfg, _ := srv.Config.Current().DeepCopy()
	cfg.System.Hostname = "" // Invalid — hostname cannot be empty.

	err := srv.applyConfig(cfg)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	ve, ok := err.(*ValidationErrors)
	if !ok {
		t.Fatalf("expected *ValidationErrors, got %T", err)
	}
	if len(ve.Errors) == 0 {
		t.Error("expected at least one validation error")
	}
}

func TestApplyConfig_RollbackOnValidation(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	original := srv.Config.Current()

	cfg, _ := srv.Config.Current().DeepCopy()
	cfg.System.Hostname = "" // Will fail validation.

	_ = srv.applyConfig(cfg)

	// Config should be unchanged.
	current := srv.Config.Current()
	if current.System.Hostname != original.System.Hostname {
		t.Errorf("hostname changed after failed apply: %q", current.System.Hostname)
	}
}

func TestAffectedDaemons_NilOld_UsesDaemonNames(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Engine.DaemonNames returns all registered renderers.
	names := srv.Render.DaemonNames()
	if len(names) != 5 {
		t.Errorf("expected 5 daemon names from engine, got %d: %v", len(names), names)
	}

	// Verify the expected daemons are present (order may vary).
	expected := map[string]bool{"nftables": true, "dnsmasq": true, "chrony": true, "unbound": true, "wireguard": true}
	for _, name := range names {
		if !expected[name] {
			t.Errorf("unexpected daemon name: %s", name)
		}
	}
}

func TestAffectedDaemons_FirewallChange(t *testing.T) {
	old := config.DefaultConfig()
	new_, _ := old.DeepCopy()
	new_.Firewall.Rules = append(new_.Firewall.Rules, config.FirewallRule{
		Name: "test", Src: "lan", Dest: "wan", Proto: "tcp", Port: 80, Action: "drop",
	})

	daemons := affectedDaemons(old, new_)
	found := false
	for _, d := range daemons {
		if d == "nftables" {
			found = true
		}
	}
	if !found {
		t.Error("expected nftables in affected daemons for firewall change")
	}
}

func TestAffectedDaemons_NTPChange(t *testing.T) {
	old := config.DefaultConfig()
	new_, _ := old.DeepCopy()
	new_.System.NTPServers = []string{"time.google.com"}

	daemons := affectedDaemons(old, new_)
	found := false
	for _, d := range daemons {
		if d == "chrony" {
			found = true
		}
	}
	if !found {
		t.Error("expected chrony in affected daemons for NTP change")
	}
}

func TestAffectedDaemons_DNSChange(t *testing.T) {
	old := config.DefaultConfig()
	new_, _ := old.DeepCopy()
	new_.DNS.Upstream = []string{"8.8.8.8"}

	daemons := affectedDaemons(old, new_)
	found := false
	for _, d := range daemons {
		if d == "unbound" {
			found = true
		}
	}
	if !found {
		t.Error("expected unbound in affected daemons for DNS change")
	}
}

func TestAffectedDaemons_NetworkChange_TriggersUnbound(t *testing.T) {
	old := config.DefaultConfig()
	new_, _ := old.DeepCopy()
	new_.Network.VLANs = append(new_.Network.VLANs, config.VLANConfig{
		ID: 100, Parent: "eth1", Zone: "guest", Subnet: "10.0.100.1/24",
	})

	daemons := affectedDaemons(old, new_)
	found := false
	for _, d := range daemons {
		if d == "unbound" {
			found = true
		}
	}
	if !found {
		t.Error("expected unbound in affected daemons for network change (access-control subnets)")
	}
}

func TestAffectedDaemons_IDSChange(t *testing.T) {
	old := config.DefaultConfig()
	new_, _ := old.DeepCopy()
	new_.IDS.Mode = "ips"

	daemons := affectedDaemons(old, new_)
	foundNftables := false
	foundSuricata := false
	for _, d := range daemons {
		if d == "nftables" {
			foundNftables = true
		}
		if d == "suricata" {
			foundSuricata = true
		}
	}
	if !foundNftables {
		t.Error("expected nftables in affected daemons for IDS change")
	}
	if !foundSuricata {
		t.Error("expected suricata in affected daemons for IDS change")
	}
}

func TestAffectedDaemons_VPNChange(t *testing.T) {
	old := config.DefaultConfig()
	new_, _ := old.DeepCopy()
	new_.VPN.WireGuard.Enabled = true

	daemons := affectedDaemons(old, new_)
	found := false
	for _, d := range daemons {
		if d == "wireguard" {
			found = true
		}
	}
	if !found {
		t.Error("expected wireguard in affected daemons for VPN change")
	}
}

func TestAffectedDaemons_NetworkChange_TriggersChrony(t *testing.T) {
	old := config.DefaultConfig()
	new_, _ := old.DeepCopy()
	new_.Network.VLANs = append(new_.Network.VLANs, config.VLANConfig{
		ID: 20, Parent: "eth1", Zone: "iot", Subnet: "192.168.20.1/24",
	})

	daemons := affectedDaemons(old, new_)
	found := false
	for _, d := range daemons {
		if d == "chrony" {
			found = true
		}
	}
	if !found {
		t.Error("expected chrony in affected daemons for network change (allow list includes VLAN subnets)")
	}
}

func TestAffectedDaemons_NoChange(t *testing.T) {
	old := config.DefaultConfig()
	new_, _ := old.DeepCopy()

	daemons := affectedDaemons(old, new_)
	if len(daemons) != 0 {
		t.Errorf("expected 0 affected daemons for identical config, got %d: %v", len(daemons), daemons)
	}
}
