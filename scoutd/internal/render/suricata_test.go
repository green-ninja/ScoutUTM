package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderSuricata_DefaultConfig(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderSuricata(cfg); err != nil {
		t.Fatalf("RenderSuricata: %v", err)
	}

	path := filepath.Join(e.outputDir, "suricata", "suricata.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	content := string(data)

	// Verify HOME_NET from config.
	if !strings.Contains(content, "192.168.1.0/24") {
		t.Error("suricata.yaml missing HOME_NET 192.168.1.0/24")
	}

	// Verify fail-open.
	if !strings.Contains(content, "fail-open: yes") {
		t.Error("suricata.yaml missing fail-open: yes")
	}

	// Verify detect profile.
	if !strings.Contains(content, "profile: low") {
		t.Error("suricata.yaml missing detect profile: low")
	}

	// Verify EVE socket path.
	if !strings.Contains(content, "/var/run/suricata/eve.sock") {
		t.Error("suricata.yaml missing EVE socket path")
	}

	// Verify rule files.
	if !strings.Contains(content, "suricata.rules") {
		t.Error("suricata.yaml missing rule file suricata.rules")
	}

	// Verify runmode.
	if !strings.Contains(content, "runmode: autofp") {
		t.Error("suricata.yaml missing runmode: autofp")
	}

	// Verify unix_dgram output.
	if !strings.Contains(content, "filetype: unix_dgram") {
		t.Error("suricata.yaml missing unix_dgram filetype")
	}
}

func TestRenderSuricata_CustomHomeNet(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.IDS.HomeNet = []string{"10.0.0.0/8", "172.16.0.0/12"}

	if err := e.RenderSuricata(cfg); err != nil {
		t.Fatalf("RenderSuricata: %v", err)
	}

	path := filepath.Join(e.outputDir, "suricata", "suricata.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "10.0.0.0/8") {
		t.Error("suricata.yaml missing custom HomeNet 10.0.0.0/8")
	}
	if !strings.Contains(content, "172.16.0.0/12") {
		t.Error("suricata.yaml missing custom HomeNet 172.16.0.0/12")
	}
}

func TestRenderSuricata_Suppressed(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.IDS.Suppressed = []int{2001, 2002}

	if err := e.RenderSuricata(cfg); err != nil {
		t.Fatalf("RenderSuricata: %v", err)
	}

	// Verify threshold.config was rendered.
	threshPath := filepath.Join(e.outputDir, "suricata", "threshold.config")
	data, err := os.ReadFile(threshPath)
	if err != nil {
		t.Fatalf("reading threshold.config: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "suppress gen_id 1, sig_id 2001") {
		t.Error("threshold.config missing SID 2001")
	}
	if !strings.Contains(content, "suppress gen_id 1, sig_id 2002") {
		t.Error("threshold.config missing SID 2002")
	}

	// Verify suricata.yaml references threshold file.
	suricataPath := filepath.Join(e.outputDir, "suricata", "suricata.yaml")
	sData, _ := os.ReadFile(suricataPath)
	if !strings.Contains(string(sData), "threshold-file") {
		t.Error("suricata.yaml missing threshold-file reference")
	}
}

func TestRenderSuricata_SuppressedRemoved_CleansThresholdFile(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	// First render with suppressed SIDs.
	cfg.IDS.Suppressed = []int{2001}
	if err := e.RenderSuricata(cfg); err != nil {
		t.Fatalf("RenderSuricata with suppressed: %v", err)
	}
	threshPath := filepath.Join(e.outputDir, "suricata", "threshold.config")
	if _, err := os.Stat(threshPath); os.IsNotExist(err) {
		t.Fatal("threshold.config should exist after rendering with suppressed SIDs")
	}

	// Re-render with empty suppressed list.
	cfg.IDS.Suppressed = []int{}
	if err := e.RenderSuricata(cfg); err != nil {
		t.Fatalf("RenderSuricata without suppressed: %v", err)
	}
	if _, err := os.Stat(threshPath); !os.IsNotExist(err) {
		t.Error("threshold.config should be removed when suppressed list is empty")
	}
}

func TestRuleFilesForRulesets(t *testing.T) {
	tests := []struct {
		name     string
		rulesets []string
		want     []string
	}{
		{"et-open", []string{"et-open"}, []string{"suricata.rules"}},
		{"unknown", []string{"custom"}, []string{"custom.rules"}},
		{"empty", nil, []string{"suricata.rules"}},
		{"mixed", []string{"et-open", "local"}, []string{"suricata.rules", "local.rules"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ruleFilesForRulesets(tt.rulesets)
			if len(got) != len(tt.want) {
				t.Fatalf("ruleFilesForRulesets(%v) = %v, want %v", tt.rulesets, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ruleFilesForRulesets(%v)[%d] = %q, want %q", tt.rulesets, i, got[i], tt.want[i])
				}
			}
		})
	}
}
