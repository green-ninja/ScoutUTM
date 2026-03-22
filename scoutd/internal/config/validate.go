package config

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// ValidationError describes a single validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

var hostnameRe = regexp.MustCompile(`^[a-zA-Z]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)
var domainRe = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
var tagSanitizeRe = regexp.MustCompile(`[^a-z0-9_]`)
var zoneNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,31}$`)
var ifaceNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,14}$`)
var safeNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_ -]{0,63}$`)
var macRe = regexp.MustCompile(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)

// containsTemplateDirective checks for Go template injection characters.
func containsTemplateDirective(s string) bool {
	return strings.Contains(s, "{{") || strings.Contains(s, "}}")
}
var timezoneRe = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_+-]+(/[A-Za-z0-9_+-]+)*|UTC|GMT)$`)
var languageRe = regexp.MustCompile(`^[a-z]{2}(-[A-Z]{2})?$`)
var rulesetNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
var validRulesets = map[string]bool{"et-open": true, "et-pro": true, "abuse-ch": true}
var wgPubKeyRe = regexp.MustCompile(`^[A-Za-z0-9+/]{42}[AEIMQUYcgkosw048]=?$`)

var validProtos = map[string]bool{"tcp": true, "udp": true, "icmp": true, "": true}
var validActions = map[string]bool{"accept": true, "drop": true, "reject": true}
var validPortForwardProtos = map[string]bool{"tcp": true, "udp": true}

// Collection size limits to prevent resource exhaustion (CWE-770).
const (
	maxFirewallRules      = 500
	maxPortForwards       = 100
	maxZones              = 20
	maxInterfaces         = 20
	maxVLANs              = 50
	maxDNSCustomBlock     = 1000
	maxDNSCustomAllow     = 1000
	maxDNSUpstream        = 10
	maxNTPServers         = 10
	maxIDSHomeNet         = 20
	maxIDSSuppressed      = 500
	maxIDSRulesets        = 10
	maxParentalProfiles   = 20
	maxParentalDevices    = 200
	maxProfileNameLen     = 64
	maxEndpointOverrideLen = 253
)

// Validate checks a Config for semantic errors and returns all violations.
func Validate(cfg *Config) []ValidationError {
	var errs []ValidationError

	// Reject Go template directive injection in all user-facing string fields.
	// text/template does NOT auto-escape for config file contexts, so {{ or }}
	// in any value could alter rendered daemon configs.
	for _, check := range []struct {
		field, value string
	}{
		{"system.hostname", cfg.System.Hostname},
		{"vpn.wireguard.endpoint_override", cfg.VPN.WireGuard.EndpointOverride},
	} {
		if containsTemplateDirective(check.value) {
			errs = append(errs, ValidationError{check.field, "must not contain template directive characters ({{ or }})"})
		}
	}
	for i, z := range cfg.Network.Zones {
		if containsTemplateDirective(z.Name) {
			errs = append(errs, ValidationError{fmt.Sprintf("network.zones[%d].name", i), "must not contain template directive characters"})
		}
	}
	for i, r := range cfg.Firewall.Rules {
		if containsTemplateDirective(r.Name) {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.rules[%d].name", i), "must not contain template directive characters"})
		}
	}
	for i, pf := range cfg.Firewall.PortForwards {
		if containsTemplateDirective(pf.Name) {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.port_forwards[%d].name", i), "must not contain template directive characters"})
		}
	}
	for i, p := range cfg.VPN.WireGuard.Peers {
		if containsTemplateDirective(p.Name) || containsTemplateDirective(p.PublicKey) {
			errs = append(errs, ValidationError{fmt.Sprintf("vpn.wireguard.peers[%d]", i), "must not contain template directive characters"})
		}
	}
	for i, p := range cfg.Parental.Profiles {
		if containsTemplateDirective(p.Name) {
			errs = append(errs, ValidationError{fmt.Sprintf("parental.profiles[%d].name", i), "must not contain template directive characters"})
		}
	}

	// Version.
	if cfg.Version < 1 || cfg.Version > CurrentSchemaVersion {
		errs = append(errs, ValidationError{"version", fmt.Sprintf("must be between 1 and %d", CurrentSchemaVersion)})
	}

	// System.
	if cfg.System.Hostname == "" {
		errs = append(errs, ValidationError{"system.hostname", "must not be empty"})
	} else if !hostnameRe.MatchString(cfg.System.Hostname) {
		errs = append(errs, ValidationError{"system.hostname", "invalid hostname format"})
	}
	if cfg.System.Timezone == "" {
		errs = append(errs, ValidationError{"system.timezone", "must not be empty"})
	} else if !timezoneRe.MatchString(cfg.System.Timezone) {
		errs = append(errs, ValidationError{"system.timezone", "must be in Area/Location format (e.g. America/New_York) or UTC"})
	}
	if cfg.System.Language != "" && !languageRe.MatchString(cfg.System.Language) {
		errs = append(errs, ValidationError{"system.language", "must be a valid language code (e.g. en, de, fr-FR)"})
	}
	if len(cfg.System.NTPServers) > maxNTPServers {
		errs = append(errs, ValidationError{"system.ntp_servers", fmt.Sprintf("too many NTP servers (max %d)", maxNTPServers)})
	}
	for i, ntp := range cfg.System.NTPServers {
		if net.ParseIP(ntp) == nil && !domainRe.MatchString(ntp) {
			errs = append(errs, ValidationError{fmt.Sprintf("system.ntp_servers[%d]", i), fmt.Sprintf("must be a valid IP address or domain name, got %q", ntp)})
		}
	}

	// Network — unique zone names + valid policies.
	if len(cfg.Network.Zones) > maxZones {
		errs = append(errs, ValidationError{"network.zones", fmt.Sprintf("too many zones (max %d)", maxZones)})
	}
	if len(cfg.Network.Interfaces) > maxInterfaces {
		errs = append(errs, ValidationError{"network.interfaces", fmt.Sprintf("too many interfaces (max %d)", maxInterfaces)})
	}
	zoneNames := make(map[string]bool)
	validPolicies := map[string]bool{"accept": true, "drop": true, "reject": true}
	for i, z := range cfg.Network.Zones {
		if z.Name == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("network.zones[%d].name", i), "must not be empty"})
			continue
		}
		if !zoneNameRe.MatchString(z.Name) {
			errs = append(errs, ValidationError{fmt.Sprintf("network.zones[%d].name", i), "must start with a letter and contain only letters, digits, and underscores (max 32 chars)"})
		}
		if zoneNames[z.Name] {
			errs = append(errs, ValidationError{fmt.Sprintf("network.zones[%d].name", i), fmt.Sprintf("duplicate zone name %q", z.Name)})
		}
		zoneNames[z.Name] = true

		if !validPolicies[z.InputPolicy] {
			errs = append(errs, ValidationError{fmt.Sprintf("network.zones[%d].input_policy", i), "must be \"accept\", \"drop\", or \"reject\""})
		}
		if !validPolicies[z.ForwardPolicy] {
			errs = append(errs, ValidationError{fmt.Sprintf("network.zones[%d].forward_policy", i), "must be \"accept\", \"drop\", or \"reject\""})
		}
		if !validPolicies[z.OutputPolicy] {
			errs = append(errs, ValidationError{fmt.Sprintf("network.zones[%d].output_policy", i), "must be \"accept\", \"drop\", or \"reject\""})
		}
	}

	// Interfaces must reference valid zones and have valid names.
	for i, iface := range cfg.Network.Interfaces {
		if iface.Name == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("network.interfaces[%d].name", i), "must not be empty"})
		} else if !ifaceNameRe.MatchString(iface.Name) {
			errs = append(errs, ValidationError{fmt.Sprintf("network.interfaces[%d].name", i), "must contain only letters, digits, dots, hyphens, and underscores (max 15 chars)"})
		}
		if iface.Zone != "" && !zoneNames[iface.Zone] {
			errs = append(errs, ValidationError{fmt.Sprintf("network.interfaces[%d].zone", i), fmt.Sprintf("references unknown zone %q", iface.Zone)})
		}
		// Validate IPv4Mode whitelist.
		if iface.IPv4Mode != "" && iface.IPv4Mode != "static" && iface.IPv4Mode != "dhcp" {
			errs = append(errs, ValidationError{fmt.Sprintf("network.interfaces[%d].ipv4_mode", i), "must be \"static\", \"dhcp\", or empty"})
		}
		// Validate IPv4Address is valid CIDR when mode is static.
		if iface.IPv4Mode == "static" && iface.IPv4Address != "" {
			if _, _, err := net.ParseCIDR(iface.IPv4Address); err != nil {
				errs = append(errs, ValidationError{fmt.Sprintf("network.interfaces[%d].ipv4_address", i), fmt.Sprintf("must be a valid CIDR (e.g. 192.168.1.1/24), got %q", iface.IPv4Address)})
			}
		}
	}

	// Build zone→interface map for forwarding validation.
	zoneHasIface := make(map[string]bool)
	for _, iface := range cfg.Network.Interfaces {
		if iface.Zone != "" {
			zoneHasIface[iface.Zone] = true
		}
	}
	for _, vlan := range cfg.Network.VLANs {
		if vlan.Zone != "" {
			zoneHasIface[vlan.Zone] = true
		}
	}

	// Firewall forwardings must reference valid zones with at least one interface.
	for i, fwd := range cfg.Firewall.Forwardings {
		if !zoneNames[fwd.Src] {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.forwardings[%d].src", i), fmt.Sprintf("references unknown zone %q", fwd.Src)})
		} else if !zoneHasIface[fwd.Src] {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.forwardings[%d].src", i), fmt.Sprintf("zone %q has no interfaces assigned", fwd.Src)})
		}
		if !zoneNames[fwd.Dest] {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.forwardings[%d].dest", i), fmt.Sprintf("references unknown zone %q", fwd.Dest)})
		} else if !zoneHasIface[fwd.Dest] {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.forwardings[%d].dest", i), fmt.Sprintf("zone %q has no interfaces assigned", fwd.Dest)})
		}
	}

	// Firewall rules.
	if len(cfg.Firewall.Rules) > maxFirewallRules {
		errs = append(errs, ValidationError{"firewall.rules", fmt.Sprintf("too many rules (max %d)", maxFirewallRules)})
	}
	for i, rule := range cfg.Firewall.Rules {
		if rule.Name != "" && !safeNameRe.MatchString(rule.Name) {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.rules[%d].name", i), "must contain only letters, digits, spaces, hyphens, and underscores (max 64 chars)"})
		}
		if !validProtos[rule.Proto] {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.rules[%d].proto", i), "must be \"tcp\", \"udp\", \"icmp\", or empty"})
		}
		if !validActions[rule.Action] {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.rules[%d].action", i), "must be \"accept\", \"drop\", or \"reject\""})
		}
		if rule.Src != "" && !zoneNames[rule.Src] {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.rules[%d].src", i), fmt.Sprintf("references unknown zone %q", rule.Src)})
		}
		if rule.Dest != "" && !zoneNames[rule.Dest] {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.rules[%d].dest", i), fmt.Sprintf("references unknown zone %q", rule.Dest)})
		}
		if rule.Port != 0 && (rule.Port < 1 || rule.Port > 65535) {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.rules[%d].port", i), "must be between 1 and 65535"})
		}
		// CFG-4: Prevent admin lockout — reject rules that would block
		// management access (HTTPS) from the LAN zone to the appliance.
		// Rules with a Dest zone are forwarding rules (through the appliance),
		// not input rules (to the appliance). Only flag rules without a Dest
		// or with Dest matching the LAN zone itself (input traffic).
		if rule.Action != "accept" && rule.Src == "lan" && rule.Dest == "" &&
			(rule.Proto == "tcp" || rule.Proto == "") &&
			(rule.Port == 443 || rule.Port == 80 || rule.Port == 8443) {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.rules[%d]", i), "cannot block management ports (80, 443, 8443) from LAN zone — this would lock out admin access"})
		}
	}

	// Validate LAN zone has accept input policy (lockout protection).
	for i, z := range cfg.Network.Zones {
		if z.Name == "lan" && z.InputPolicy != "accept" {
			errs = append(errs, ValidationError{fmt.Sprintf("network.zones[%d].input_policy", i), "LAN zone input policy must be \"accept\" — changing it would lock out admin access"})
		}
	}

	// Port forwards.
	if len(cfg.Firewall.PortForwards) > maxPortForwards {
		errs = append(errs, ValidationError{"firewall.port_forwards", fmt.Sprintf("too many port forwards (max %d)", maxPortForwards)})
	}
	for i, pf := range cfg.Firewall.PortForwards {
		if pf.Name != "" && !safeNameRe.MatchString(pf.Name) {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.port_forwards[%d].name", i), "must contain only letters, digits, spaces, hyphens, and underscores (max 64 chars)"})
		}
		if !validPortForwardProtos[pf.Proto] {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.port_forwards[%d].proto", i), "must be \"tcp\" or \"udp\""})
		}
		if pf.SrcPort < 1 || pf.SrcPort > 65535 {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.port_forwards[%d].src_port", i), "must be between 1 and 65535"})
		}
		if pf.DestPort < 1 || pf.DestPort > 65535 {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.port_forwards[%d].dest_port", i), "must be between 1 and 65535"})
		}
		if pf.DestIP == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.port_forwards[%d].dest_ip", i), "must not be empty"})
		} else if net.ParseIP(pf.DestIP) == nil {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.port_forwards[%d].dest_ip", i), "invalid IP address"})
		}
	}

	// DNS upstreams.
	if len(cfg.DNS.Upstream) > maxDNSUpstream {
		errs = append(errs, ValidationError{"dns.upstream", fmt.Sprintf("too many upstream servers (max %d)", maxDNSUpstream)})
	}
	if len(cfg.DNS.Upstream) == 0 {
		errs = append(errs, ValidationError{"dns.upstream", "must have at least one upstream server"})
	}
	for i, u := range cfg.DNS.Upstream {
		if net.ParseIP(u) == nil {
			errs = append(errs, ValidationError{fmt.Sprintf("dns.upstream[%d]", i), fmt.Sprintf("invalid IP address %q", u)})
		}
	}

	// DNS: custom block domains.
	if len(cfg.DNS.CustomBlock) > maxDNSCustomBlock {
		errs = append(errs, ValidationError{"dns.custom_block", fmt.Sprintf("too many custom block entries (max %d)", maxDNSCustomBlock)})
	}
	for i, d := range cfg.DNS.CustomBlock {
		if !domainRe.MatchString(d) {
			errs = append(errs, ValidationError{fmt.Sprintf("dns.custom_block[%d]", i), fmt.Sprintf("invalid domain name %q", d)})
		}
	}

	// DNS: custom allow domains.
	if len(cfg.DNS.CustomAllow) > maxDNSCustomAllow {
		errs = append(errs, ValidationError{"dns.custom_allow", fmt.Sprintf("too many custom allow entries (max %d)", maxDNSCustomAllow)})
	}
	for i, d := range cfg.DNS.CustomAllow {
		if !domainRe.MatchString(d) {
			errs = append(errs, ValidationError{fmt.Sprintf("dns.custom_allow[%d]", i), fmt.Sprintf("invalid domain name %q", d)})
		}
	}

	// IDS.
	if cfg.IDS.Mode != "ids" && cfg.IDS.Mode != "ips" {
		errs = append(errs, ValidationError{"ids.mode", "must be \"ids\" or \"ips\""})
	}
	if len(cfg.IDS.HomeNet) > maxIDSHomeNet {
		errs = append(errs, ValidationError{"ids.home_net", fmt.Sprintf("too many home_net entries (max %d)", maxIDSHomeNet)})
	}
	for i, cidr := range cfg.IDS.HomeNet {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			errs = append(errs, ValidationError{fmt.Sprintf("ids.home_net[%d]", i), fmt.Sprintf("invalid CIDR %q", cidr)})
		}
	}
	if len(cfg.IDS.Suppressed) > maxIDSSuppressed {
		errs = append(errs, ValidationError{"ids.suppressed", fmt.Sprintf("too many suppressed SIDs (max %d)", maxIDSSuppressed)})
	}
	for i, sid := range cfg.IDS.Suppressed {
		if sid < 1 || sid > 99999999 {
			errs = append(errs, ValidationError{fmt.Sprintf("ids.suppressed[%d]", i), "must be a positive integer (1-99999999)"})
		}
	}
	if len(cfg.IDS.Rulesets) > maxIDSRulesets {
		errs = append(errs, ValidationError{"ids.rulesets", fmt.Sprintf("too many rulesets (max %d)", maxIDSRulesets)})
	}
	for i, rs := range cfg.IDS.Rulesets {
		if !validRulesets[rs] {
			errs = append(errs, ValidationError{fmt.Sprintf("ids.rulesets[%d]", i), fmt.Sprintf("unknown ruleset %q (supported: et-open, et-pro, abuse-ch)", rs)})
		}
	}

	// Network VLANs.
	if len(cfg.Network.VLANs) > maxVLANs {
		errs = append(errs, ValidationError{"network.vlans", fmt.Sprintf("too many VLANs (max %d)", maxVLANs)})
	}
	ifaceNames := make(map[string]bool)
	for _, iface := range cfg.Network.Interfaces {
		ifaceNames[iface.Name] = true
	}
	vlanIDs := make(map[int]bool)
	for i, vlan := range cfg.Network.VLANs {
		if vlan.ID < 1 || vlan.ID > 4094 {
			errs = append(errs, ValidationError{fmt.Sprintf("network.vlans[%d].id", i), "must be between 1 and 4094"})
		}
		if vlanIDs[vlan.ID] {
			errs = append(errs, ValidationError{fmt.Sprintf("network.vlans[%d].id", i), fmt.Sprintf("duplicate VLAN ID %d", vlan.ID)})
		}
		vlanIDs[vlan.ID] = true
		if vlan.Parent == "" || !ifaceNames[vlan.Parent] {
			errs = append(errs, ValidationError{fmt.Sprintf("network.vlans[%d].parent", i), fmt.Sprintf("references unknown interface %q", vlan.Parent)})
		}
		if vlan.Zone != "" && !zoneNames[vlan.Zone] {
			errs = append(errs, ValidationError{fmt.Sprintf("network.vlans[%d].zone", i), fmt.Sprintf("references unknown zone %q", vlan.Zone)})
		}
		if vlan.Subnet != "" {
			if _, _, err := net.ParseCIDR(vlan.Subnet); err != nil {
				errs = append(errs, ValidationError{fmt.Sprintf("network.vlans[%d].subnet", i), fmt.Sprintf("invalid CIDR %q", vlan.Subnet)})
			}
		}
		if vlan.DHCPRange.Start != "" && net.ParseIP(vlan.DHCPRange.Start) == nil {
			errs = append(errs, ValidationError{fmt.Sprintf("network.vlans[%d].dhcp_range.start", i), "invalid IP address"})
		}
		if vlan.DHCPRange.End != "" && net.ParseIP(vlan.DHCPRange.End) == nil {
			errs = append(errs, ValidationError{fmt.Sprintf("network.vlans[%d].dhcp_range.end", i), "invalid IP address"})
		}
	}

	// VPN: WireGuard settings (when enabled).
	if cfg.VPN.WireGuard.Enabled {
		if cfg.VPN.WireGuard.ListenPort < 1 || cfg.VPN.WireGuard.ListenPort > 65535 {
			errs = append(errs, ValidationError{"vpn.wireguard.listen_port", "must be between 1 and 65535"})
		}
		if cfg.VPN.WireGuard.Address != "" {
			if _, _, err := net.ParseCIDR(cfg.VPN.WireGuard.Address); err != nil {
				errs = append(errs, ValidationError{"vpn.wireguard.address", fmt.Sprintf("invalid CIDR %q", cfg.VPN.WireGuard.Address)})
			}
		}
		if cfg.VPN.WireGuard.TunnelSubnet != "" {
			if _, _, err := net.ParseCIDR(cfg.VPN.WireGuard.TunnelSubnet); err != nil {
				errs = append(errs, ValidationError{"vpn.wireguard.tunnel_subnet", fmt.Sprintf("invalid CIDR %q", cfg.VPN.WireGuard.TunnelSubnet)})
			}
		}
	}

	// VPN: EndpointOverride must not contain newlines and must be a valid host or host:port.
	if cfg.VPN.WireGuard.EndpointOverride != "" {
		eo := cfg.VPN.WireGuard.EndpointOverride
		if strings.ContainsAny(eo, "\n\r") {
			errs = append(errs, ValidationError{"vpn.wireguard.endpoint_override", "must not contain newlines"})
		}
		if len(eo) > maxEndpointOverrideLen {
			errs = append(errs, ValidationError{"vpn.wireguard.endpoint_override", fmt.Sprintf("too long (max %d chars)", maxEndpointOverrideLen)})
		}
	}

	// VPN: WireGuard peer validation.
	peerPubKeys := make(map[string]bool)
	peerTunnelIPs := make(map[string]bool)
	for i, peer := range cfg.VPN.WireGuard.Peers {
		if peer.Name != "" && !safeNameRe.MatchString(peer.Name) {
			errs = append(errs, ValidationError{fmt.Sprintf("vpn.wireguard.peers[%d].name", i), "must contain only letters, digits, spaces, hyphens, and underscores (max 64 chars)"})
		}
		// PublicKey: reject newlines (config injection) and validate base64 format.
		if peer.PublicKey != "" {
			if strings.ContainsAny(peer.PublicKey, "\n\r") {
				errs = append(errs, ValidationError{fmt.Sprintf("vpn.wireguard.peers[%d].public_key", i), "must not contain newlines"})
			} else if !wgPubKeyRe.MatchString(peer.PublicKey) {
				errs = append(errs, ValidationError{fmt.Sprintf("vpn.wireguard.peers[%d].public_key", i), "must be a valid base64-encoded 32-byte key"})
			}
		}
		if peerPubKeys[peer.PublicKey] {
			errs = append(errs, ValidationError{fmt.Sprintf("vpn.wireguard.peers[%d].public_key", i), "duplicate public key"})
		}
		peerPubKeys[peer.PublicKey] = true
		// TunnelIP: must be a valid IP address.
		if peer.TunnelIP != "" && net.ParseIP(peer.TunnelIP) == nil {
			errs = append(errs, ValidationError{fmt.Sprintf("vpn.wireguard.peers[%d].tunnel_ip", i), fmt.Sprintf("invalid IP address %q", peer.TunnelIP)})
		}
		if peerTunnelIPs[peer.TunnelIP] {
			errs = append(errs, ValidationError{fmt.Sprintf("vpn.wireguard.peers[%d].tunnel_ip", i), "duplicate tunnel IP"})
		}
		peerTunnelIPs[peer.TunnelIP] = true
		for j, cidr := range peer.AllowedIPs {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				errs = append(errs, ValidationError{
					fmt.Sprintf("vpn.wireguard.peers[%d].allowed_ips[%d]", i, j),
					fmt.Sprintf("invalid CIDR %q", cidr),
				})
			}
		}
	}

	// Parental: profiles.
	if len(cfg.Parental.Profiles) > maxParentalProfiles {
		errs = append(errs, ValidationError{"parental.profiles", fmt.Sprintf("too many profiles (max %d)", maxParentalProfiles)})
	}
	profileIDs := make(map[string]bool)
	profileNames := make(map[string]bool)
	profileTagNames := make(map[string]bool) // sanitized tag names for collision detection
	// Must match blocklist.Categories() — only categories with zone files.
	validParentalCategories := map[string]bool{
		"adult": true, "gambling": true, "gaming": true, "social_media": true, "streaming": true,
	}
	for i, p := range cfg.Parental.Profiles {
		if p.ID == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("parental.profiles[%d].id", i), "must not be empty"})
		} else if profileIDs[p.ID] {
			errs = append(errs, ValidationError{fmt.Sprintf("parental.profiles[%d].id", i), fmt.Sprintf("duplicate profile ID %q", p.ID)})
		}
		profileIDs[p.ID] = true

		lower := strings.ToLower(p.Name)
		if p.Name == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("parental.profiles[%d].name", i), "must not be empty"})
		} else if len(p.Name) > maxProfileNameLen {
			errs = append(errs, ValidationError{fmt.Sprintf("parental.profiles[%d].name", i), fmt.Sprintf("too long (max %d chars)", maxProfileNameLen)})
		} else if profileNames[lower] {
			errs = append(errs, ValidationError{fmt.Sprintf("parental.profiles[%d].name", i), fmt.Sprintf("duplicate profile name %q", p.Name)})
		}
		profileNames[lower] = true

		// Check for sanitized tag collision (e.g. "Kids-Only" and "KidsOnly" both become "kidsonly").
		if p.Name != "" {
			tag := tagSanitizeRe.ReplaceAllString(strings.ReplaceAll(lower, " ", "_"), "")
			if tag == "" {
				tag = "profile"
			}
			if profileTagNames[tag] {
				errs = append(errs, ValidationError{fmt.Sprintf("parental.profiles[%d].name", i), fmt.Sprintf("profile name %q produces duplicate DNS tag %q", p.Name, tag)})
			}
			profileTagNames[tag] = true
		}

		for j, cat := range p.BlockCategories {
			if !validParentalCategories[cat] {
				errs = append(errs, ValidationError{fmt.Sprintf("parental.profiles[%d].block_categories[%d]", i, j), fmt.Sprintf("invalid category %q", cat)})
			}
		}
	}

	// Parental: devices.
	if len(cfg.Parental.Devices) > maxParentalDevices {
		errs = append(errs, ValidationError{"parental.devices", fmt.Sprintf("too many devices (max %d)", maxParentalDevices)})
	}
	for i, d := range cfg.Parental.Devices {
		if d.MAC == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("parental.devices[%d].mac", i), "must not be empty"})
		} else if !macRe.MatchString(d.MAC) {
			errs = append(errs, ValidationError{fmt.Sprintf("parental.devices[%d].mac", i), fmt.Sprintf("invalid MAC address format %q (expected XX:XX:XX:XX:XX:XX)", d.MAC)})
		}
		if d.ProfileID != "" && !profileIDs[d.ProfileID] {
			errs = append(errs, ValidationError{fmt.Sprintf("parental.devices[%d].profile_id", i), fmt.Sprintf("references unknown profile %q", d.ProfileID)})
		}
		if d.IP != "" && net.ParseIP(d.IP) == nil {
			errs = append(errs, ValidationError{fmt.Sprintf("parental.devices[%d].ip", i), fmt.Sprintf("invalid IP address %q", d.IP)})
		}
	}

	// Update.
	switch cfg.Update.Channel {
	case "stable", "beta":
		// ok
	default:
		errs = append(errs, ValidationError{"update.channel", "must be \"stable\" or \"beta\""})
	}
	if cfg.Update.ServerURL != "" {
		if !strings.HasPrefix(cfg.Update.ServerURL, "https://") {
			errs = append(errs, ValidationError{"update.server_url", "must use HTTPS"})
		} else if !isAllowedUpdateHost(cfg.Update.ServerURL) {
			errs = append(errs, ValidationError{"update.server_url", "must use an authorized update server"})
		}
	}

	return errs
}

// allowedUpdateHosts is the hardcoded allowlist of domains permitted for OTA updates.
var allowedUpdateHosts = []string{
	"update.scoututm.com",
	"update.zenti.sh",
}

// isAllowedUpdateHost checks if a URL's host is on the OTA update allowlist.
func isAllowedUpdateHost(rawURL string) bool {
	host := rawURL
	if idx := strings.Index(host, "://"); idx >= 0 {
		host = host[idx+3:]
	}
	if idx := strings.IndexByte(host, '/'); idx >= 0 {
		host = host[:idx]
	}
	if idx := strings.LastIndexByte(host, ':'); idx >= 0 {
		host = host[:idx]
	}
	for _, allowed := range allowedUpdateHosts {
		if strings.EqualFold(host, allowed) {
			return true
		}
	}
	return false
}

// isPrivateURL checks if a URL's host resolves to a private/internal IP range.
// Prevents SSRF attacks where the update server URL is set to an internal host.
func isPrivateURL(rawURL string) bool {
	// Extract host from URL.
	host := rawURL
	if idx := strings.Index(host, "://"); idx >= 0 {
		host = host[idx+3:]
	}
	// Remove path.
	if idx := strings.IndexByte(host, '/'); idx >= 0 {
		host = host[:idx]
	}
	// Remove port.
	if idx := strings.LastIndexByte(host, ':'); idx >= 0 {
		host = host[:idx]
	}
	if host == "" {
		return true
	}

	// Check if host is a literal IP.
	if ip := net.ParseIP(host); ip != nil {
		return isPrivateIP(ip)
	}

	// Resolve hostname and check all IPs.
	ips, err := net.LookupIP(host)
	if err != nil {
		return false // Can't resolve — let the HTTP client fail later.
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return true
		}
	}
	return false
}

// isPrivateIP checks if an IP is in a private/internal range.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range privateRanges {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}
