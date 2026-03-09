package service

import (
	"fmt"
	"log"
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
	stubAll      bool   // Stub all reloads (unit tests).
	stubNftables bool   // Stub only nftables (Vagrant VM — SSH safety).
}

// NewManager creates a service manager. When stubAll is true, all daemon
// reloads are no-ops (for unit tests without real daemons).
func NewManager(stubAll bool) *Manager {
	return &Manager{stubAll: stubAll}
}

// NewVMManager creates a service manager for the Vagrant VM. Only nftables
// reloads are stubbed (to protect SSH access). All other daemons reload
// normally so the VM behaves like production.
func NewVMManager() *Manager {
	return &Manager{stubNftables: true}
}

// ReloadDaemon reloads a single daemon.
func (m *Manager) ReloadDaemon(name string) error {
	if m.stubAll {
		log.Printf("[dev-mode] would reload daemon: %s", name)
		return nil
	}
	if m.stubNftables && name == Nftables {
		log.Printf("[dev-mode] skipping nftables reload (SSH safety)")
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

// SuricataUpdate runs suricata-update to fetch new IDS rule signatures.
//
// The --etopen flag explicitly selects the Emerging Threats Open ruleset
// over HTTPS. suricata-update verifies checksums of downloaded archives.
// The --disable-ssl-verify flag is intentionally never used.
func (m *Manager) SuricataUpdate() error {
	if m.stubAll {
		log.Printf("[dev-mode] would run suricata-update")
		return nil
	}
	return runCmd("suricata-update", "--etopen")
}

// HealthCheck verifies that reloaded daemons are running correctly.
func (m *Manager) HealthCheck(daemons []string) error {
	if m.stubAll {
		return nil
	}
	for _, name := range daemons {
		if m.stubNftables && name == Nftables {
			continue
		}
		if err := checkDaemon(name); err != nil {
			return fmt.Errorf("health check failed for %s: %w", name, err)
		}
	}
	return nil
}
