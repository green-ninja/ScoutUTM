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

func TestValidate_ParentalDuplicateProfileID(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Parental.Profiles = []ParentalProfile{
		{ID: "p1", Name: "Kids", BlockCategories: []string{}},
		{ID: "p1", Name: "Teens", BlockCategories: []string{}},
	}
	errs := Validate(cfg)
	if !hasField(errs, "parental.profiles[1].id") {
		t.Error("expected error for duplicate profile ID")
	}
}

func TestValidate_ParentalDuplicateProfileName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Parental.Profiles = []ParentalProfile{
		{ID: "p1", Name: "Kids", BlockCategories: []string{}},
		{ID: "p2", Name: "kids", BlockCategories: []string{}},
	}
	errs := Validate(cfg)
	if !hasField(errs, "parental.profiles[1].name") {
		t.Error("expected error for duplicate profile name (case-insensitive)")
	}
}

func TestValidate_ParentalInvalidCategory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Parental.Profiles = []ParentalProfile{
		{ID: "p1", Name: "Kids", BlockCategories: []string{"adult", "nonsense"}},
	}
	errs := Validate(cfg)
	if !hasField(errs, "parental.profiles[0].block_categories[1]") {
		t.Error("expected error for invalid category")
	}
}

func TestValidate_ParentalDeviceBadProfileRef(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Parental.Devices = []ParentalDevice{
		{MAC: "aa:bb:cc:dd:ee:01", ProfileID: "nonexistent"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "parental.devices[0].profile_id") {
		t.Error("expected error for device referencing nonexistent profile")
	}
}

func TestValidate_ParentalDeviceInvalidIP(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Parental.Profiles = []ParentalProfile{
		{ID: "p1", Name: "Kids", BlockCategories: []string{}},
	}
	cfg.Parental.Devices = []ParentalDevice{
		{MAC: "aa:bb:cc:dd:ee:01", ProfileID: "p1", IP: "not-an-ip"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "parental.devices[0].ip") {
		t.Error("expected error for invalid device IP")
	}
}

func TestValidate_ParentalTagCollision(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Parental.Profiles = []ParentalProfile{
		{ID: "p1", Name: "Kids-Only", BlockCategories: []string{}},
		{ID: "p2", Name: "KidsOnly", BlockCategories: []string{}},
	}
	errs := Validate(cfg)
	// Both names sanitize to "kidsonly", producing a tag collision.
	found := false
	for _, e := range errs {
		if e.Field == "parental.profiles[1].name" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error for profile names that produce duplicate DNS tags")
	}
}

// --- Input sanitization tests (config injection prevention) ---

func TestValidate_ZoneNameFormat(t *testing.T) {
	tests := []struct {
		name    string
		zone    string
		wantErr bool
	}{
		{"valid", "iot", false},
		{"valid_underscore", "iot_devices", false},
		{"valid_uppercase", "DMZ", false},
		{"starts_with_digit", "1zone", true},
		{"contains_space", "my zone", true},
		{"contains_semicolon", "lan;drop", true},
		{"contains_newline", "lan\nchain", true},
		{"contains_hyphen", "my-zone", true}, // hyphens not allowed in nftables identifiers
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Network.Zones = append(cfg.Network.Zones, ZoneConfig{Name: tt.zone, InputPolicy: "drop", ForwardPolicy: "drop", OutputPolicy: "accept"})
			errs := Validate(cfg)
			hasErr := hasField(errs, "network.zones[2].name")
			if tt.wantErr && !hasErr {
				t.Errorf("expected error for zone name %q", tt.zone)
			}
			if !tt.wantErr && hasErr {
				t.Errorf("unexpected error for zone name %q", tt.zone)
			}
		})
	}
}

func TestValidate_InterfaceNameFormat(t *testing.T) {
	tests := []struct {
		name    string
		iface   string
		wantErr bool
	}{
		{"valid_simple", "eth0", false},
		{"valid_vlan", "eth1.20", false},
		{"valid_bridge", "br0", false},
		{"valid_wg", "wg0", false},
		{"contains_space", "eth 0", true},
		{"contains_newline", "eth0\n", true},
		{"contains_semicolon", "eth0;rm", true},
		{"too_long", "abcdefghijklmnop", true}, // 16 chars, max 15
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Network.Interfaces = append(cfg.Network.Interfaces, InterfaceConfig{Name: tt.iface, Zone: "lan"})
			errs := Validate(cfg)
			hasErr := hasField(errs, "network.interfaces[2].name")
			if tt.wantErr && !hasErr {
				t.Errorf("expected error for interface name %q", tt.iface)
			}
			if !tt.wantErr && hasErr {
				t.Errorf("unexpected error for interface name %q", tt.iface)
			}
		})
	}
}

func TestValidate_FirewallRuleProto(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Firewall.Rules = []FirewallRule{
		{Name: "test", Src: "lan", Dest: "wan", Proto: "invalid", Port: 80, Action: "accept"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "firewall.rules[0].proto") {
		t.Error("expected error for invalid proto")
	}
}

func TestValidate_FirewallRuleProtoValid(t *testing.T) {
	for _, proto := range []string{"tcp", "udp", "icmp", ""} {
		cfg := DefaultConfig()
		cfg.Firewall.Rules = []FirewallRule{
			{Name: "test", Src: "lan", Dest: "wan", Proto: proto, Port: 80, Action: "accept"},
		}
		errs := Validate(cfg)
		if hasField(errs, "firewall.rules[0].proto") {
			t.Errorf("proto %q should be valid", proto)
		}
	}
}

func TestValidate_FirewallRuleAction(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Firewall.Rules = []FirewallRule{
		{Name: "test", Src: "lan", Dest: "wan", Proto: "tcp", Port: 80, Action: "invalid"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "firewall.rules[0].action") {
		t.Error("expected error for invalid action")
	}
}

func TestValidate_FirewallRuleActionInjection(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Firewall.Rules = []FirewallRule{
		{Name: "test", Src: "lan", Dest: "wan", Proto: "tcp", Port: 80, Action: "accept; drop"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "firewall.rules[0].action") {
		t.Error("expected error for action with injection attempt")
	}
}

func TestValidate_FirewallRuleSrcDest(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Firewall.Rules = []FirewallRule{
		{Name: "test", Src: "nonexistent", Dest: "wan", Proto: "tcp", Port: 80, Action: "accept"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "firewall.rules[0].src") {
		t.Error("expected error for rule referencing unknown src zone")
	}

	cfg2 := DefaultConfig()
	cfg2.Firewall.Rules = []FirewallRule{
		{Name: "test", Src: "lan", Dest: "fakezone", Proto: "tcp", Port: 80, Action: "drop"},
	}
	errs2 := Validate(cfg2)
	if !hasField(errs2, "firewall.rules[0].dest") {
		t.Error("expected error for rule referencing unknown dest zone")
	}
}

func TestValidate_FirewallRuleName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Firewall.Rules = []FirewallRule{
		{Name: "test\nchain evil {}", Src: "lan", Dest: "wan", Proto: "tcp", Port: 80, Action: "accept"},
	}
	errs := Validate(cfg)
	if !hasField(errs, "firewall.rules[0].name") {
		t.Error("expected error for rule name with newline injection")
	}
}

func TestValidate_PortForwardProto(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Firewall.PortForwards = []PortForward{
		{Name: "test", Proto: "icmp", SrcPort: 80, DestIP: "192.168.1.100", DestPort: 80},
	}
	errs := Validate(cfg)
	if !hasField(errs, "firewall.port_forwards[0].proto") {
		t.Error("expected error for port forward with icmp proto")
	}
}

func TestValidate_PortForwardName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Firewall.PortForwards = []PortForward{
		{Name: "test\niifname eth0 accept", Proto: "tcp", SrcPort: 80, DestIP: "192.168.1.100", DestPort: 80},
	}
	errs := Validate(cfg)
	if !hasField(errs, "firewall.port_forwards[0].name") {
		t.Error("expected error for port forward name with newline injection")
	}
}

func TestValidate_NTPServerFormat(t *testing.T) {
	tests := []struct {
		name    string
		ntp     string
		wantErr bool
	}{
		{"valid_domain", "pool.ntp.org", false},
		{"valid_ip", "1.2.3.4", false},
		{"valid_subdomain", "time.google.com", false},
		{"injection_newline", "evil.com\nallow 0.0.0.0/0", true},
		{"injection_space", "evil.com iburst\nallow", true},
		{"empty", "", true},
		{"bare_word", "localhost", true}, // not a valid domain (no TLD dot)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.System.NTPServers = []string{tt.ntp}
			errs := Validate(cfg)
			hasErr := hasField(errs, "system.ntp_servers[0]")
			if tt.wantErr && !hasErr {
				t.Errorf("expected error for NTP server %q", tt.ntp)
			}
			if !tt.wantErr && hasErr {
				t.Errorf("unexpected error for NTP server %q", tt.ntp)
			}
		})
	}
}

func TestValidate_MACAddressFormat(t *testing.T) {
	tests := []struct {
		name    string
		mac     string
		wantErr bool
	}{
		{"valid_lower", "aa:bb:cc:dd:ee:ff", false},
		{"valid_upper", "AA:BB:CC:DD:EE:FF", false},
		{"valid_mixed", "aA:bB:01:23:45:67", false},
		{"injection_comma", "aa:bb:cc:dd:ee:ff,set:evil", true},
		{"injection_newline", "aa:bb:cc:dd:ee:ff\ndhcp-host=ff:ff:ff:ff:ff:ff", true},
		{"too_short", "aa:bb:cc", true},
		{"dash_separated", "aa-bb-cc-dd-ee-ff", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Parental.Profiles = []ParentalProfile{
				{ID: "p1", Name: "Kids", BlockCategories: []string{}},
			}
			cfg.Parental.Devices = []ParentalDevice{
				{MAC: tt.mac, ProfileID: "p1", IP: "192.168.1.100"},
			}
			errs := Validate(cfg)
			hasErr := hasField(errs, "parental.devices[0].mac")
			if tt.wantErr && !hasErr {
				t.Errorf("expected error for MAC %q", tt.mac)
			}
			if !tt.wantErr && hasErr {
				t.Errorf("unexpected error for MAC %q", tt.mac)
			}
		})
	}
}

func TestValidate_IDSRulesetName(t *testing.T) {
	tests := []struct {
		name    string
		ruleset string
		wantErr bool
	}{
		{"valid_et_open", "et-open", false},
		{"valid_et_pro", "et-pro", false},
		{"valid_abuse_ch", "abuse-ch", false},
		{"unknown_custom", "custom_rules", true},
		{"path_traversal", "../../etc/passwd", true},
		{"yaml_injection", "rules\n  - evil", true},
		{"space", "et open", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.IDS.Rulesets = []string{tt.ruleset}
			errs := Validate(cfg)
			hasErr := hasField(errs, "ids.rulesets[0]")
			if tt.wantErr && !hasErr {
				t.Errorf("expected error for ruleset %q", tt.ruleset)
			}
			if !tt.wantErr && hasErr {
				t.Errorf("unexpected error for ruleset %q", tt.ruleset)
			}
		})
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
