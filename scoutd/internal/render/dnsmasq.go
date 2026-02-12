package render

import (
	"fmt"
	"net"

	"github.com/scoututm/scoutd/internal/config"
)

// DnsmasqData holds template data for dnsmasq.conf.
type DnsmasqData struct {
	Interfaces []DnsmasqInterface
	LeaseFile  string
}

// DnsmasqInterface represents DHCP config for one interface.
type DnsmasqInterface struct {
	Name      string
	IP        string
	DHCPStart string
	DHCPEnd   string
	Netmask   string
	LeaseTime string
}

// RenderDnsmasq renders dnsmasq.conf from the config.
func (e *Engine) RenderDnsmasq(cfg *config.Config) error {
	data := DnsmasqData{
		Interfaces: buildDnsmasqInterfaces(cfg),
		LeaseFile:  "/var/lib/scout/dnsmasq.leases",
	}
	return e.renderTemplate("dnsmasq.conf.tmpl", e.outputPath("dnsmasq.conf"), data)
}

// buildDnsmasqInterfaces builds the DHCP interface list from static interfaces and VLANs.
func buildDnsmasqInterfaces(cfg *config.Config) []DnsmasqInterface {
	var ifaces []DnsmasqInterface

	for _, iface := range cfg.Network.Interfaces {
		if iface.Zone == "wan" {
			continue
		}
		if iface.IPv4Mode != "static" || iface.IPv4Address == "" {
			continue
		}
		di := buildDnsmasqInterface(iface.Name, iface.IPv4Address, "", "")
		if di != nil {
			ifaces = append(ifaces, *di)
		}
	}

	for _, vlan := range cfg.Network.VLANs {
		if vlan.Subnet == "" {
			continue
		}
		ifName := fmt.Sprintf("%s.%d", vlan.Parent, vlan.ID)
		di := buildDnsmasqInterface(ifName, vlan.Subnet, vlan.DHCPRange.Start, vlan.DHCPRange.End)
		if di != nil {
			ifaces = append(ifaces, *di)
		}
	}

	return ifaces
}

func buildDnsmasqInterface(name, cidr, rangeStart, rangeEnd string) *DnsmasqInterface {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return nil
	}

	mask := ipnet.Mask
	netmask := fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])

	if rangeStart == "" {
		rangeStart = fmt.Sprintf("%d.%d.%d.100", ip4[0], ip4[1], ip4[2])
	}
	if rangeEnd == "" {
		rangeEnd = fmt.Sprintf("%d.%d.%d.253", ip4[0], ip4[1], ip4[2])
	}

	return &DnsmasqInterface{
		Name:      name,
		IP:        ip.String(),
		DHCPStart: rangeStart,
		DHCPEnd:   rangeEnd,
		Netmask:   netmask,
		LeaseTime: "12h",
	}
}
