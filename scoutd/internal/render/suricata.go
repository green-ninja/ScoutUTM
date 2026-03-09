package render

import (
	"os"
	"strings"

	"github.com/scoututm/scoutd/internal/config"
)

// SuricataData holds template data for suricata.yaml.
type SuricataData struct {
	HomeNet       string
	Interface     string
	RulePath      string
	RuleFiles     []string
	EveSocketPath string
	DetectProfile string
	Suppressed    []int
}

// RenderSuricata renders suricata.yaml from the config.
func (e *Engine) RenderSuricata(cfg *config.Config) error {
	homeNets := cfg.IDS.HomeNet
	if len(homeNets) == 0 {
		homeNets = []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"}
	}

	// Suricata monitors the LAN interface for intrusion detection.
	lanIface := "eth1"
	for _, iface := range cfg.Network.Interfaces {
		if iface.Zone == "lan" {
			lanIface = iface.Name
			break
		}
	}

	data := SuricataData{
		HomeNet:       "[" + strings.Join(homeNets, ", ") + "]",
		Interface:     lanIface,
		RulePath:      "/var/lib/suricata/rules",
		RuleFiles:     ruleFilesForRulesets(cfg.IDS.Rulesets),
		EveSocketPath: "/var/run/suricata/eve.sock",
		DetectProfile: "low",
		Suppressed:    cfg.IDS.Suppressed,
	}

	if err := e.renderTemplate("suricata.yaml.tmpl", e.outputPath("suricata/suricata.yaml"), data); err != nil {
		return err
	}

	// Render threshold.config for suppressed SIDs, or remove stale file.
	thresholdPath := e.outputPath("suricata/threshold.config")
	if len(cfg.IDS.Suppressed) > 0 {
		return e.renderTemplate("threshold.config.tmpl", thresholdPath, data)
	}
	os.Remove(thresholdPath)

	return nil
}

// ruleFilesForRulesets maps known ruleset names to rule file names.
// Unknown rulesets are silently ignored (defense-in-depth; validation
// should reject them before they reach here).
func ruleFilesForRulesets(rulesets []string) []string {
	// abuse-ch rules are merged into suricata.rules by suricata-update
	// (via enable-source), so they don't need a separate file entry.
	mapping := map[string]string{
		"et-open": "suricata.rules",
		"et-pro":  "et-pro.rules",
	}
	var files []string
	for _, rs := range rulesets {
		if f, ok := mapping[rs]; ok {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		files = []string{"suricata.rules"}
	}
	return files
}
