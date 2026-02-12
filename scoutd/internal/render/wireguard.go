package render

import (
	"github.com/scoututm/scoutd/internal/config"
)

// WireGuardData holds template data for wireguard.conf.
type WireGuardData struct {
	Enabled    bool
	PrivateKey string
	ListenPort int
	Address    string
	Peers      []config.WireGuardPeer
}

// RenderWireGuard renders wireguard.conf from the config.
func (e *Engine) RenderWireGuard(cfg *config.Config) error {
	data := WireGuardData{
		Enabled:    cfg.VPN.WireGuard.Enabled,
		PrivateKey: cfg.VPN.WireGuard.PrivateKey,
		ListenPort: cfg.VPN.WireGuard.ListenPort,
		Address:    cfg.VPN.WireGuard.Address,
		Peers:      cfg.VPN.WireGuard.Peers,
	}
	return e.renderTemplate("wireguard.conf.tmpl", e.outputPath("wireguard.conf"), data)
}
