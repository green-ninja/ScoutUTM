package config

import (
	"testing"
)

func TestValidate_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	errs := Validate(cfg)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("unexpected error: %s", e)
		}
	}
}

func TestValidate_EmptyHostname(t *testing.T) {
	cfg := DefaultConfig()
	cfg.System.Hostname = ""
	errs := Validate(cfg)
	if !hasField(errs, "system.hostname") {
		t.Error("expected error for empty hostname")
	}
}

func TestValidate_InvalidHostname(t *testing.T) {
	cfg := DefaultConfig()
	cfg.System.Hostname = "123-invalid"
	errs := Validate(cfg)
	if !hasField(errs, "system.hostname") {
		t.Error("expected error for invalid hostname")
	}
}

func TestValidate_DuplicateZones(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Network.Zones = append(cfg.Network.Zones, ZoneConfig{Name: "wan"})
	errs := Validate(cfg)
	if !hasField(errs, "network.zones[2].name") {
		t.Error("expected error for duplicate zone name")
	}
}

func TestValidate_BadForwardingRef(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Firewall.Forwardings = append(cfg.Firewall.Forwardings, ForwardingRule{Src: "lan", Dest: "nonexistent"})
	errs := Validate(cfg)
	if !hasField(errs, "firewall.forwardings[1].dest") {
		t.Error("expected error for bad forwarding destination")
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Firewall.Rules = []FirewallRule{
		{Name: "test", Src: "lan", Dest: "wan", Proto: "tcp", Port: 99999, Action: "accept"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "firewall.rules[0].port") {
		t.Error("expected error for invalid port")
	}
}

func TestValidate_InvalidDNSUpstream(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DNS.Upstream = []string{"not-an-ip"}
	errs := Validate(cfg)
	if !hasField(errs, "dns.upstream[0]") {
		t.Error("expected error for invalid DNS upstream")
	}
}

func TestValidate_EmptyDNSUpstream(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DNS.Upstream = []string{}
	errs := Validate(cfg)
	if !hasField(errs, "dns.upstream") {
		t.Error("expected error for empty DNS upstream")
	}
}

func TestValidate_InvalidIDSMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IDS.Mode = "invalid"
	errs := Validate(cfg)
	if !hasField(errs, "ids.mode") {
		t.Error("expected error for invalid IDS mode")
	}
}

func TestValidate_InvalidUpdateChannel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Update.Channel = "nightly"
	errs := Validate(cfg)
	if !hasField(errs, "update.channel") {
		t.Error("expected error for invalid update channel")
	}
}

func TestValidate_InvalidPortForward(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Firewall.PortForwards = []PortForward{
		{Name: "test", Proto: "tcp", SrcPort: 0, DestIP: "192.168.1.100", DestPort: 80},
	}
	errs := Validate(cfg)
	if !hasField(errs, "firewall.port_forwards[0].src_port") {
		t.Error("expected error for invalid src_port")
	}
}

func hasField(errs []ValidationError, field string) bool {
	for _, e := range errs {
		if e.Field == field {
			return true
		}
	}
	return false
}
