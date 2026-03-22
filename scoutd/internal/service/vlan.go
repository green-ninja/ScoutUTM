package service

import (
	"fmt"
	"log"
	"net"

	"github.com/scoututm/scoutd/internal/config"
)

// vlanIfName returns the Linux sub-interface name for a VLAN (e.g. "eth1.20").
func vlanIfName(v config.VLANConfig) string {
	return fmt.Sprintf("%s.%d", v.Parent, v.ID)
}

// gatewayAddr computes the gateway IP from a VLAN subnet CIDR.
// For "192.168.20.0/24" it returns "192.168.20.1/24" (first usable host).
func gatewayAddr(subnet string) (string, error) {
	_, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", fmt.Errorf("parsing subnet %q: %w", subnet, err)
	}
	netIP := ipnet.IP.To4()
	if netIP == nil {
		return "", fmt.Errorf("subnet %q is not IPv4", subnet)
	}
	// First usable host = network address + 1 (works for all prefix lengths).
	gw := make(net.IP, 4)
	copy(gw, netIP)
	// Increment as a 32-bit big-endian integer.
	for i := 3; i >= 0; i-- {
		gw[i]++
		if gw[i] != 0 {
			break
		}
	}
	ones, _ := ipnet.Mask.Size()
	return fmt.Sprintf("%s/%d", gw.String(), ones), nil
}

// SyncVLANs diffs old and new VLAN configs and creates/deletes Linux
// VLAN sub-interfaces accordingly. Must be called before daemon reload
// so that interfaces exist when nftables/dnsmasq reference them.
func (m *Manager) SyncVLANs(old, new []config.VLANConfig) error {
	if m.stubAll {
		for _, v := range diffAdded(old, new) {
			log.Printf("[dev-mode] would create VLAN interface %s (id %d, subnet %s)", vlanIfName(v), v.ID, v.Subnet)
		}
		for _, v := range diffRemoved(old, new) {
			log.Printf("[dev-mode] would delete VLAN interface %s", vlanIfName(v))
		}
		return nil
	}

	// Delete removed VLANs first.
	for _, v := range diffRemoved(old, new) {
		ifname := vlanIfName(v)
		log.Printf("deleting VLAN interface %s", ifname)
		if err := runCmd("ip", "link", "del", ifname); err != nil {
			log.Printf("warning: deleting VLAN %s: %v", ifname, err)
			// Non-fatal: interface may not exist yet.
		}
	}

	// Create added VLANs.
	for _, v := range diffAdded(old, new) {
		ifname := vlanIfName(v)
		log.Printf("creating VLAN interface %s (id %d)", ifname, v.ID)

		if err := runCmd("ip", "link", "add", "link", v.Parent,
			"name", ifname, "type", "vlan", "id", fmt.Sprint(v.ID)); err != nil {
			return fmt.Errorf("creating VLAN %s: %w", ifname, err)
		}

		gw, err := gatewayAddr(v.Subnet)
		if err != nil {
			return fmt.Errorf("computing gateway for VLAN %s: %w", ifname, err)
		}
		if err := runCmd("ip", "addr", "add", gw, "dev", ifname); err != nil {
			return fmt.Errorf("assigning address to VLAN %s: %w", ifname, err)
		}

		if err := runCmd("ip", "link", "set", ifname, "up"); err != nil {
			return fmt.Errorf("bringing up VLAN %s: %w", ifname, err)
		}
	}

	return nil
}

// diffAdded returns VLANs in new that are not in old (by ID).
func diffAdded(old, new []config.VLANConfig) []config.VLANConfig {
	oldIDs := make(map[int]bool, len(old))
	for _, v := range old {
		oldIDs[v.ID] = true
	}
	var added []config.VLANConfig
	for _, v := range new {
		if !oldIDs[v.ID] {
			added = append(added, v)
		}
	}
	return added
}

// diffRemoved returns VLANs in old that are not in new (by ID).
func diffRemoved(old, new []config.VLANConfig) []config.VLANConfig {
	newIDs := make(map[int]bool, len(new))
	for _, v := range new {
		newIDs[v.ID] = true
	}
	var removed []config.VLANConfig
	for _, v := range old {
		if !newIDs[v.ID] {
			removed = append(removed, v)
		}
	}
	return removed
}
