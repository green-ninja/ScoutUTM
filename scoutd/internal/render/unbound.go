package render

import (
	"net"

	"github.com/scoututm/scoutd/internal/config"
)

// UnboundData holds template data for unbound.conf.
type UnboundData struct {
	ListenAddr   string
	Port         int
	AccessNets   []string
	Upstreams    []UnboundUpstream
	UseTLS       bool
	DNSSEC       bool
	QueryLogging bool
	RPZZones     []UnboundRPZZone
	CustomBlock  []string
	CustomAllow  []string
}

// UnboundUpstream represents a DNS forwarder.
type UnboundUpstream struct {
	IP   string
	Port int
}

// UnboundRPZZone represents a response policy zone for blocklists.
type UnboundRPZZone struct {
	Name     string
	ZoneFile string
}

// RenderUnbound renders unbound.conf from the config.
func (e *Engine) RenderUnbound(cfg *config.Config) error {
	data := buildUnboundData(cfg)
	return e.renderTemplate("unbound.conf.tmpl", e.outputPath("unbound/unbound.conf"), data)
}

// buildUnboundData constructs the template data from config.
func buildUnboundData(cfg *config.Config) UnboundData {
	port := 53
	if cfg.DNS.UseDOT {
		port = 853
	}

	var upstreams []UnboundUpstream
	for _, ip := range cfg.DNS.Upstream {
		upstreams = append(upstreams, UnboundUpstream{
			IP:   ip,
			Port: port,
		})
	}

	var rpzZones []UnboundRPZZone
	if cfg.DNS.BlockMalware {
		rpzZones = append(rpzZones, UnboundRPZZone{
			Name:     "rpz-malware",
			ZoneFile: "/var/lib/scout/blocklists/rpz-malware.zone",
		})
	}
	if cfg.DNS.BlockPhishing {
		rpzZones = append(rpzZones, UnboundRPZZone{
			Name:     "rpz-phishing",
			ZoneFile: "/var/lib/scout/blocklists/rpz-phishing.zone",
		})
	}
	if cfg.DNS.BlockAds {
		rpzZones = append(rpzZones, UnboundRPZZone{
			Name:     "rpz-ads",
			ZoneFile: "/var/lib/scout/blocklists/rpz-ads.zone",
		})
	}

	return UnboundData{
		ListenAddr:   "0.0.0.0",
		Port:         53,
		AccessNets:   unboundAccessNets(cfg),
		Upstreams:    upstreams,
		UseTLS:       cfg.DNS.UseDOT,
		DNSSEC:       true,
		QueryLogging: cfg.DNS.QueryLogging,
		RPZZones:     rpzZones,
		CustomBlock:  cfg.DNS.CustomBlock,
		CustomAllow:  cfg.DNS.CustomAllow,
	}
}

// unboundAccessNets returns the subnets allowed to query Unbound.
// Same logic as chronyAllowNets: non-WAN interfaces + VLANs.
func unboundAccessNets(cfg *config.Config) []string {
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
