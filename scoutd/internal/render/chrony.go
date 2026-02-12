package render

import (
	"net"

	"github.com/scoututm/scoutd/internal/config"
)

// ChronyData holds template data for chrony.conf.
type ChronyData struct {
	NTPServers []string
	AllowNets  []string
}

// RenderChrony renders chrony.conf from the config.
func (e *Engine) RenderChrony(cfg *config.Config) error {
	data := ChronyData{
		NTPServers: cfg.System.NTPServers,
		AllowNets:  chronyAllowNets(cfg),
	}
	return e.renderTemplate("chrony.conf.tmpl", e.outputPath("chrony/chrony.conf"), data)
}

// chronyAllowNets returns the networks that should be allowed to query NTP.
func chronyAllowNets(cfg *config.Config) []string {
	var nets []string
	for _, iface := range cfg.Network.Interfaces {
		if iface.Zone == "wan" {
			continue
		}
		if iface.IPv4Mode == "static" && iface.IPv4Address != "" {
			_, ipnet, err := net.ParseCIDR(iface.IPv4Address)
			if err == nil {
				nets = append(nets, ipnet.String())
			}
		}
	}
	for _, vlan := range cfg.Network.VLANs {
		if vlan.Subnet != "" {
			_, ipnet, err := net.ParseCIDR(vlan.Subnet)
			if err == nil {
				nets = append(nets, ipnet.String())
			}
		}
	}
	return nets
}
