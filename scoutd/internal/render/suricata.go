package render

import (
	"fmt"
	"strings"

	"github.com/scoututm/scoutd/internal/config"
)

// SuricataData holds template data for suricata.yaml.
type SuricataData struct {
	HomeNet       string
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

	data := SuricataData{
		HomeNet:       "[" + strings.Join(homeNets, ", ") + "]",
		RulePath:      "/var/lib/suricata/rules",
		RuleFiles:     ruleFilesForRulesets(cfg.IDS.Rulesets),
		EveSocketPath: "/var/run/suricata/eve.sock",
		DetectProfile: "low",
		Suppressed:    cfg.IDS.Suppressed,
	}

	if err := e.renderTemplate("suricata.yaml.tmpl", e.outputPath("suricata/suricata.yaml"), data); err != nil {
		return err
	}

	// Render threshold.config for suppressed SIDs.
	if len(cfg.IDS.Suppressed) > 0 {
		return e.renderTemplate("threshold.config.tmpl", e.outputPath("suricata/threshold.config"), data)
	}

	return nil
}

// ruleFilesForRulesets maps ruleset names to rule file names.
func ruleFilesForRulesets(rulesets []string) []string {
	mapping := map[string]string{
		"et-open": "suricata.rules",
	}
	var files []string
	for _, rs := range rulesets {
		if f, ok := mapping[rs]; ok {
			files = append(files, f)
		} else {
			files = append(files, fmt.Sprintf("%s.rules", rs))
		}
	}
	if len(files) == 0 {
		files = []string{"suricata.rules"}
	}
	return files
}
