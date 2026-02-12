package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

// testEngine creates a render engine pointing at the real templates dir and a temp output dir.
func testEngine(t *testing.T) *Engine {
	t.Helper()
	tmplDir := filepath.Join("..", "..", "templates")
	outputDir := t.TempDir()
	return NewEngine(tmplDir, outputDir, true)
}

func testConfig() *config.Config {
	return config.DefaultConfig()
}

func TestRenderAll(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderAll(cfg); err != nil {
		t.Fatalf("RenderAll: %v", err)
	}

	// Verify output files exist.
	for _, name := range []string{"nftables.conf", "dnsmasq.conf", "chrony/chrony.conf"} {
		path := filepath.Join(e.outputDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected output file %s to exist", name)
		}
	}
}

func TestRenderDaemons_Subset(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderDaemons(cfg, []string{"chrony"}); err != nil {
		t.Fatalf("RenderDaemons: %v", err)
	}

	// chrony should exist.
	chronyPath := filepath.Join(e.outputDir, "chrony", "chrony.conf")
	if _, err := os.Stat(chronyPath); os.IsNotExist(err) {
		t.Error("expected chrony.conf to exist")
	}

	// nftables should NOT exist (not in subset).
	nftPath := filepath.Join(e.outputDir, "nftables.conf")
	if _, err := os.Stat(nftPath); !os.IsNotExist(err) {
		t.Error("expected nftables.conf to NOT exist")
	}
}

func TestRenderDaemons_UnknownSkipped(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	// "unbound" has no renderer yet — should be skipped without error.
	if err := e.RenderDaemons(cfg, []string{"unbound"}); err != nil {
		t.Fatalf("RenderDaemons with unknown: %v", err)
	}
}

func TestTemplateFuncMap_CidrToIP(t *testing.T) {
	fm := templateFuncMap()
	fn := fm["cidrToIP"].(func(string) string)
	if got := fn("192.168.1.1/24"); got != "192.168.1.1" {
		t.Errorf("cidrToIP = %q, want 192.168.1.1", got)
	}
}

func TestTemplateFuncMap_CidrToNet(t *testing.T) {
	fm := templateFuncMap()
	fn := fm["cidrToNet"].(func(string) string)
	if got := fn("192.168.1.1/24"); got != "192.168.1.0/24" {
		t.Errorf("cidrToNet = %q, want 192.168.1.0/24", got)
	}
}

func TestTemplateFuncMap_DhcpStart(t *testing.T) {
	fm := templateFuncMap()
	fn := fm["dhcpStart"].(func(string) string)
	if got := fn("192.168.1.1/24"); got != "192.168.1.100" {
		t.Errorf("dhcpStart = %q, want 192.168.1.100", got)
	}
}

func TestTemplateFuncMap_DhcpEnd(t *testing.T) {
	fm := templateFuncMap()
	fn := fm["dhcpEnd"].(func(string) string)
	if got := fn("192.168.1.1/24"); got != "192.168.1.253" {
		t.Errorf("dhcpEnd = %q, want 192.168.1.253", got)
	}
}
