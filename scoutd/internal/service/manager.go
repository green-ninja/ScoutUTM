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
	devMode bool
}

// NewManager creates a service manager.
func NewManager(devMode bool) *Manager {
	return &Manager{devMode: devMode}
}

// ReloadDaemon reloads a single daemon. In dev-mode, it only logs.
func (m *Manager) ReloadDaemon(name string) error {
	if m.devMode {
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

// HealthCheck verifies that reloaded daemons are running correctly.
// In dev-mode, always passes. In production, checks each daemon's status.
func (m *Manager) HealthCheck(daemons []string) error {
	if m.devMode {
		log.Printf("[dev-mode] would health-check daemons: %v", daemons)
		return nil
	}
	for _, name := range daemons {
		if err := checkDaemon(name); err != nil {
			return fmt.Errorf("health check failed for %s: %w", name, err)
		}
	}
	return nil
}
