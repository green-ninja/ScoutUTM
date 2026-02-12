package render

import (
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
		AllowNets:  localSubnets(cfg),
	}
	return e.renderTemplate("chrony.conf.tmpl", e.outputPath("chrony/chrony.conf"), data)
}
