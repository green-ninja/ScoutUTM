package render

import (
	"fmt"

	"github.com/scoututm/scoutd/internal/config"
)

// NftablesData holds template data for nftables.conf.
type NftablesData struct {
	Zones          []config.ZoneConfig
	Interfaces     []config.InterfaceConfig
	VLANs          []config.VLANConfig
	Forwardings    []config.ForwardingRule
	Rules          []config.FirewallRule
	PortForwards   []config.PortForward
	IDS            config.IDSConfig
	VPN            config.WireGuardConfig
	ZoneInterfaces map[string][]string
}

// RenderNftables renders nftables.conf from the config.
func (e *Engine) RenderNftables(cfg *config.Config) error {
	data := NftablesData{
		Zones:          cfg.Network.Zones,
		Interfaces:     cfg.Network.Interfaces,
		VLANs:          cfg.Network.VLANs,
		Forwardings:    cfg.Firewall.Forwardings,
		Rules:          cfg.Firewall.Rules,
		PortForwards:   cfg.Firewall.PortForwards,
		IDS:            cfg.IDS,
		VPN:            cfg.VPN.WireGuard,
		ZoneInterfaces: buildZoneInterfaceMap(cfg),
	}
	return e.renderTemplate("nftables.conf.tmpl", e.outputPath("nftables.conf"), data)
}

// buildZoneInterfaceMap maps zone name → list of interface names (includes VLAN sub-interfaces).
func buildZoneInterfaceMap(cfg *config.Config) map[string][]string {
	m := make(map[string][]string)
	for _, iface := range cfg.Network.Interfaces {
		if iface.Zone != "" {
			m[iface.Zone] = append(m[iface.Zone], iface.Name)
		}
	}
	for _, vlan := range cfg.Network.VLANs {
		if vlan.Zone != "" {
			ifName := fmt.Sprintf("%s.%d", vlan.Parent, vlan.ID)
			m[vlan.Zone] = append(m[vlan.Zone], ifName)
		}
	}
	return m
}
