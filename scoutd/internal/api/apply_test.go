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
	if len(names) != 3 {
		t.Errorf("expected 3 daemon names from engine, got %d: %v", len(names), names)
	}

	// Verify the expected daemons are present (order may vary).
	expected := map[string]bool{"nftables": true, "dnsmasq": true, "chrony": true}
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

func TestAffectedDaemons_NoChange(t *testing.T) {
	old := config.DefaultConfig()
	new_, _ := old.DeepCopy()

	daemons := affectedDaemons(old, new_)
	if len(daemons) != 0 {
		t.Errorf("expected 0 affected daemons for identical config, got %d: %v", len(daemons), daemons)
	}
}
