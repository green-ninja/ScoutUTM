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

var hostnameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-]{0,62}$`)

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

	// Network — unique zone names.
	zoneNames := make(map[string]bool)
	for i, z := range cfg.Network.Zones {
		if z.Name == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("network.zones[%d].name", i), "must not be empty"})
			continue
		}
		if zoneNames[z.Name] {
			errs = append(errs, ValidationError{fmt.Sprintf("network.zones[%d].name", i), fmt.Sprintf("duplicate zone name %q", z.Name)})
		}
		zoneNames[z.Name] = true
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

	// IDS.
	if cfg.IDS.Mode != "ids" && cfg.IDS.Mode != "ips" {
		errs = append(errs, ValidationError{"ids.mode", "must be \"ids\" or \"ips\""})
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
