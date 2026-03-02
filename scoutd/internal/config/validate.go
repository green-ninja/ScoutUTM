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

// Validate checks a Config for semantic errors and returns all violations.
func Validate(cfg *Config) []ValidationError {
	var errs []ValidationError

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
	}

	// Network — unique zone names + valid policies.
	zoneNames := make(map[string]bool)
	validPolicies := map[string]bool{"accept": true, "drop": true, "reject": true}
	for i, z := range cfg.Network.Zones {
		if z.Name == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("network.zones[%d].name", i), "must not be empty"})
			continue
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

	// Interfaces must reference valid zones.
	for i, iface := range cfg.Network.Interfaces {
		if iface.Name == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("network.interfaces[%d].name", i), "must not be empty"})
		}
		if iface.Zone != "" && !zoneNames[iface.Zone] {
			errs = append(errs, ValidationError{fmt.Sprintf("network.interfaces[%d].zone", i), fmt.Sprintf("references unknown zone %q", iface.Zone)})
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
	for i, rule := range cfg.Firewall.Rules {
		if rule.Port != 0 && (rule.Port < 1 || rule.Port > 65535) {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.rules[%d].port", i), "must be between 1 and 65535"})
		}
	}

	// Port forwards.
	for i, pf := range cfg.Firewall.PortForwards {
		if pf.SrcPort < 1 || pf.SrcPort > 65535 {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.port_forwards[%d].src_port", i), "must be between 1 and 65535"})
		}
		if pf.DestPort < 1 || pf.DestPort > 65535 {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.port_forwards[%d].dest_port", i), "must be between 1 and 65535"})
		}
		if pf.DestIP != "" && net.ParseIP(pf.DestIP) == nil {
			errs = append(errs, ValidationError{fmt.Sprintf("firewall.port_forwards[%d].dest_ip", i), "invalid IP address"})
		}
	}

	// DNS upstreams.
	if len(cfg.DNS.Upstream) == 0 {
		errs = append(errs, ValidationError{"dns.upstream", "must have at least one upstream server"})
	}
	for i, u := range cfg.DNS.Upstream {
		if net.ParseIP(u) == nil {
			errs = append(errs, ValidationError{fmt.Sprintf("dns.upstream[%d]", i), fmt.Sprintf("invalid IP address %q", u)})
		}
	}

	// DNS: DoT and DoH are mutually exclusive.
	if cfg.DNS.UseDOT && cfg.DNS.UseDOH {
		errs = append(errs, ValidationError{"dns.use_dot", "use_dot and use_doh cannot both be enabled"})
	}

	// DNS: custom block domains.
	for i, d := range cfg.DNS.CustomBlock {
		if !domainRe.MatchString(d) {
			errs = append(errs, ValidationError{fmt.Sprintf("dns.custom_block[%d]", i), fmt.Sprintf("invalid domain name %q", d)})
		}
	}

	// DNS: custom allow domains.
	for i, d := range cfg.DNS.CustomAllow {
		if !domainRe.MatchString(d) {
			errs = append(errs, ValidationError{fmt.Sprintf("dns.custom_allow[%d]", i), fmt.Sprintf("invalid domain name %q", d)})
		}
	}

	// IDS.
	if cfg.IDS.Mode != "ids" && cfg.IDS.Mode != "ips" {
		errs = append(errs, ValidationError{"ids.mode", "must be \"ids\" or \"ips\""})
	}
	for i, cidr := range cfg.IDS.HomeNet {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			errs = append(errs, ValidationError{fmt.Sprintf("ids.home_net[%d]", i), fmt.Sprintf("invalid CIDR %q", cidr)})
		}
	}

	// Network VLANs.
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

	// VPN: WireGuard peer validation.
	peerPubKeys := make(map[string]bool)
	peerTunnelIPs := make(map[string]bool)
	for i, peer := range cfg.VPN.WireGuard.Peers {
		if peerPubKeys[peer.PublicKey] {
			errs = append(errs, ValidationError{fmt.Sprintf("vpn.wireguard.peers[%d].public_key", i), "duplicate public key"})
		}
		peerPubKeys[peer.PublicKey] = true
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
	profileIDs := make(map[string]bool)
	profileNames := make(map[string]bool)
	profileTagNames := make(map[string]bool) // sanitized tag names for collision detection
	validParentalCategories := map[string]bool{
		"social_media": true, "gaming": true, "adult": true, "streaming": true,
		"gambling": true, "drugs": true, "weapons": true, "malware": true,
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
	for i, d := range cfg.Parental.Devices {
		if d.MAC == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("parental.devices[%d].mac", i), "must not be empty"})
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
		if !strings.HasPrefix(cfg.Update.ServerURL, "https://") && !strings.HasPrefix(cfg.Update.ServerURL, "http://") {
			errs = append(errs, ValidationError{"update.server_url", "must be a valid URL"})
		}
	}

	return errs
}
