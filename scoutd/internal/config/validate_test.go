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

func TestValidate_ForwardingZoneNoInterfaces(t *testing.T) {
	cfg := DefaultConfig()
	// Add a zone with no interfaces assigned.
	cfg.Network.Zones = append(cfg.Network.Zones, ZoneConfig{Name: "orphan", InputPolicy: "drop", ForwardPolicy: "drop", OutputPolicy: "accept"})
	cfg.Firewall.Forwardings = append(cfg.Firewall.Forwardings, ForwardingRule{Src: "orphan", Dest: "wan"})

	errs := Validate(cfg)
	if !hasField(errs, "firewall.forwardings[1].src") {
		t.Error("expected error for forwarding with zone that has no interfaces")
	}
}

func TestValidate_HostnameTrailingHyphen(t *testing.T) {
	cfg := DefaultConfig()
	cfg.System.Hostname = "scout-"
	errs := Validate(cfg)
	if !hasField(errs, "system.hostname") {
		t.Error("expected error for hostname with trailing hyphen")
	}
}

func TestValidate_HostnameSingleChar(t *testing.T) {
	cfg := DefaultConfig()
	cfg.System.Hostname = "s"
	errs := Validate(cfg)
	if hasField(errs, "system.hostname") {
		t.Error("single-letter hostname should be valid")
	}
}

func TestValidate_InvalidZonePolicy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Network.Zones[0].InputPolicy = "invalid"
	errs := Validate(cfg)
	if !hasField(errs, "network.zones[0].input_policy") {
		t.Error("expected error for invalid input policy")
	}
}

func TestValidate_InvalidIDSHomeNet(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IDS.HomeNet = []string{"not-a-cidr"}
	errs := Validate(cfg)
	if !hasField(errs, "ids.home_net[0]") {
		t.Error("expected error for invalid IDS home_net CIDR")
	}
}

func TestValidate_InvalidWireGuardAllowedIPs(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VPN.WireGuard.Peers = []WireGuardPeer{
		{Name: "test", PublicKey: "key", AllowedIPs: []string{"bad-cidr"}, TunnelIP: "10.0.0.2"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "vpn.wireguard.peers[0].allowed_ips[0]") {
		t.Error("expected error for invalid WireGuard AllowedIPs CIDR")
	}
}

func TestValidate_ValidCIDRValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IDS.HomeNet = []string{"192.168.1.0/24", "10.0.0.0/8"}
	cfg.VPN.WireGuard.Peers = []WireGuardPeer{
		{Name: "test", PublicKey: "key", AllowedIPs: []string{"0.0.0.0/0"}, TunnelIP: "10.0.0.2"},
	}
	errs := Validate(cfg)
	if hasField(errs, "ids.home_net[0]") || hasField(errs, "vpn.wireguard.peers[0].allowed_ips[0]") {
		t.Error("valid CIDRs should not produce errors")
	}
}

func TestValidate_VLANIDRange(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Network.VLANs = []VLANConfig{
		{ID: 0, Parent: "eth1", Zone: "lan", Subnet: "10.0.100.1/24"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "network.vlans[0].id") {
		t.Error("expected error for VLAN ID 0")
	}

	cfg.Network.VLANs = []VLANConfig{
		{ID: 5000, Parent: "eth1", Zone: "lan", Subnet: "10.0.100.1/24"},
	}
	errs = Validate(cfg)
	if !hasField(errs, "network.vlans[0].id") {
		t.Error("expected error for VLAN ID 5000")
	}
}

func TestValidate_VLANDuplicateID(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Network.VLANs = []VLANConfig{
		{ID: 100, Parent: "eth1", Zone: "lan", Subnet: "10.0.100.1/24"},
		{ID: 100, Parent: "eth1", Zone: "lan", Subnet: "10.0.101.1/24"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "network.vlans[1].id") {
		t.Error("expected error for duplicate VLAN ID")
	}
}

func TestValidate_VLANInvalidParent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Network.VLANs = []VLANConfig{
		{ID: 100, Parent: "nonexistent", Zone: "lan", Subnet: "10.0.100.1/24"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "network.vlans[0].parent") {
		t.Error("expected error for invalid VLAN parent interface")
	}
}

func TestValidate_VLANInvalidZone(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Network.VLANs = []VLANConfig{
		{ID: 100, Parent: "eth1", Zone: "nonexistent", Subnet: "10.0.100.1/24"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "network.vlans[0].zone") {
		t.Error("expected error for invalid VLAN zone reference")
	}
}

func TestValidate_VLANInvalidSubnet(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Network.VLANs = []VLANConfig{
		{ID: 100, Parent: "eth1", Zone: "lan", Subnet: "not-a-cidr"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "network.vlans[0].subnet") {
		t.Error("expected error for invalid VLAN subnet")
	}
}

func TestValidate_WireGuardListenPortRange(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VPN.WireGuard.Enabled = true
	cfg.VPN.WireGuard.ListenPort = 0
	errs := Validate(cfg)
	if !hasField(errs, "vpn.wireguard.listen_port") {
		t.Error("expected error for listen_port 0")
	}

	cfg.VPN.WireGuard.ListenPort = 70000
	errs = Validate(cfg)
	if !hasField(errs, "vpn.wireguard.listen_port") {
		t.Error("expected error for listen_port 70000")
	}
}

func TestValidate_WireGuardInvalidAddress(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VPN.WireGuard.Enabled = true
	cfg.VPN.WireGuard.Address = "not-a-cidr"
	errs := Validate(cfg)
	if !hasField(errs, "vpn.wireguard.address") {
		t.Error("expected error for invalid WireGuard address")
	}
}

func TestValidate_WireGuardDuplicatePeerKeys(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VPN.WireGuard.Peers = []WireGuardPeer{
		{Name: "a", PublicKey: "key1", AllowedIPs: []string{"10.0.0.2/32"}, TunnelIP: "10.0.0.2"},
		{Name: "b", PublicKey: "key1", AllowedIPs: []string{"10.0.0.3/32"}, TunnelIP: "10.0.0.3"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "vpn.wireguard.peers[1].public_key") {
		t.Error("expected error for duplicate peer public key")
	}
}

func TestValidate_WireGuardDuplicateTunnelIPs(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VPN.WireGuard.Peers = []WireGuardPeer{
		{Name: "a", PublicKey: "key1", AllowedIPs: []string{"10.0.0.2/32"}, TunnelIP: "10.0.0.2"},
		{Name: "b", PublicKey: "key2", AllowedIPs: []string{"10.0.0.2/32"}, TunnelIP: "10.0.0.2"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "vpn.wireguard.peers[1].tunnel_ip") {
		t.Error("expected error for duplicate peer tunnel IP")
	}
}

func TestValidate_ValidVLAN(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Network.VLANs = []VLANConfig{
		{ID: 100, Parent: "eth1", Zone: "lan", Subnet: "10.0.100.1/24"},
	}
	errs := Validate(cfg)
	if hasField(errs, "network.vlans[0].id") || hasField(errs, "network.vlans[0].parent") || hasField(errs, "network.vlans[0].zone") {
		t.Error("valid VLAN should not produce errors")
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
