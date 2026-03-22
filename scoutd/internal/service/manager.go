package service

import (
	"fmt"
	"log"

	"github.com/scoututm/scoutd/internal/config"
)

// Daemon name constants.
const (
	Nftables  = "nftables"
	Dnsmasq   = "dnsmasq"
	Unbound   = "unbound"
	Suricata  = "suricata"
	WireGuard = "wireguard"
	Chrony    = "chrony"
)

// ReloadOrder defines the order in which daemons must be reloaded.
var ReloadOrder = []string{Nftables, Dnsmasq, Unbound, Suricata, WireGuard, Chrony}

// Manager controls system daemons via OpenRC.
type Manager struct {
	stubAll bool // Stub all reloads (unit tests).
}

// NewManager creates a service manager. When stubAll is true, all daemon
// reloads are no-ops (for unit tests without real daemons).
func NewManager(stubAll bool) *Manager {
	return &Manager{stubAll: stubAll}
}

// ReloadDaemon reloads a single daemon.
func (m *Manager) ReloadDaemon(name string) error {
	if m.stubAll {
		log.Printf("[dev-mode] would reload daemon: %s", name)
		return nil
	}
	return reloadDaemon(name)
}

// ReloadDaemons reloads daemons in the correct dependency order.
// Only daemons in the provided list are reloaded, but ordering is preserved.
func (m *Manager) ReloadDaemons(daemons []string) error {
	wanted := make(map[string]bool, len(daemons))
	for _, d := range daemons {
		wanted[d] = true
	}

	for _, name := range ReloadOrder {
		if !wanted[name] {
			continue
		}
		if err := m.ReloadDaemon(name); err != nil {
			return fmt.Errorf("reloading %s: %w", name, err)
		}
	}
	return nil
}

// abuseCHSources are the suricata-update source names for abuse.ch feeds.
// These are managed via enable-source/disable-source and merged into
// suricata.rules alongside ET Open/Pro rules.
var abuseCHSources = []string{
	"sslbl/ssl-fp-blacklist",
	"sslbl/ja3-fingerprints",
}

// SuricataUpdate runs suricata-update to fetch new IDS rule signatures.
//
// The --etopen flag selects the Emerging Threats Open ruleset over HTTPS.
// abuse.ch sources (SSLBL, JA3) are managed via suricata-update's
// enable-source/disable-source mechanism.
// suricata-update verifies checksums of downloaded archives.
// The --disable-ssl-verify flag is intentionally never used.
func (m *Manager) SuricataUpdate(rulesets []string) error {
	if m.stubAll {
		log.Printf("[dev-mode] would run suricata-update (rulesets: %v)", rulesets)
		return nil
	}

	// Sync abuse.ch sources based on config.
	wantAbuseCH := false
	for _, rs := range rulesets {
		if rs == "abuse-ch" {
			wantAbuseCH = true
			break
		}
	}
	for _, src := range abuseCHSources {
		if wantAbuseCH {
			_ = runCmd("suricata-update", "enable-source", src)
		} else {
			_ = runCmd("suricata-update", "disable-source", src)
		}
	}

	return runCmd("suricata-update", "--etopen")
}

// ApplyInterfaceAddresses configures IPv4 addresses on static (non-WAN)
// interfaces whose address changed between old and new config.
func (m *Manager) ApplyInterfaceAddresses(oldIfaces, newIfaces []config.InterfaceConfig) error {
	oldAddrs := make(map[string]string, len(oldIfaces))
	for _, iface := range oldIfaces {
		if iface.IPv4Mode == "static" && iface.IPv4Address != "" {
			oldAddrs[iface.Name] = iface.IPv4Address
		}
	}

	for _, iface := range newIfaces {
		if iface.Zone == "wan" || iface.IPv4Mode != "static" || iface.IPv4Address == "" {
			continue
		}
		if oldAddrs[iface.Name] == iface.IPv4Address {
			continue
		}
		if m.stubAll {
			log.Printf("[stub] would apply address %s to %s", iface.IPv4Address, iface.Name)
			continue
		}
		// Flush old IPv4 addresses, then assign the new one.
		_ = runCmd("ip", "-4", "addr", "flush", "dev", iface.Name)
		if err := runCmd("ip", "addr", "add", iface.IPv4Address, "dev", iface.Name); err != nil {
			return fmt.Errorf("applying address %s to %s: %w", iface.IPv4Address, iface.Name, err)
		}
		_ = runCmd("ip", "link", "set", iface.Name, "up")
		log.Printf("applied address %s to %s", iface.IPv4Address, iface.Name)
	}
	return nil
}

// HealthCheck verifies that reloaded daemons are running correctly.
func (m *Manager) HealthCheck(daemons []string) error {
	if m.stubAll {
		return nil
	}
	for _, name := range daemons {
		if err := checkDaemon(name); err != nil {
			return fmt.Errorf("health check failed for %s: %w", name, err)
		}
	}
	return nil
}
