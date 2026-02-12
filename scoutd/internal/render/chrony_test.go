package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderChrony_Default(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderChrony(cfg); err != nil {
		t.Fatalf("RenderChrony: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(e.outputDir, "chrony", "chrony.conf"))
	if err != nil {
		t.Fatalf("reading chrony.conf: %v", err)
	}
	content := string(data)

	// NTP servers with iburst.
	if !strings.Contains(content, "server pool.ntp.org iburst") {
		t.Error("expected 'server pool.ntp.org iburst'")
	}

	// Allow LAN network.
	if !strings.Contains(content, "allow 192.168.1.0/24") {
		t.Error("expected 'allow 192.168.1.0/24'")
	}

	// Standard chrony directives.
	if !strings.Contains(content, "driftfile") {
		t.Error("expected driftfile directive")
	}
	if !strings.Contains(content, "makestep") {
		t.Error("expected makestep directive")
	}
	if !strings.Contains(content, "rtcsync") {
		t.Error("expected rtcsync directive")
	}
}

func TestRenderChrony_MultipleNTPServers(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()
	cfg.System.NTPServers = []string{"0.pool.ntp.org", "1.pool.ntp.org", "time.google.com"}

	if err := e.RenderChrony(cfg); err != nil {
		t.Fatalf("RenderChrony: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "chrony", "chrony.conf"))
	content := string(data)

	for _, srv := range cfg.System.NTPServers {
		expected := "server " + srv + " iburst"
		if !strings.Contains(content, expected) {
			t.Errorf("expected %q in output", expected)
		}
	}
}

func TestRenderChrony_NoWANAllow(t *testing.T) {
	e := testEngine(t)
	cfg := testConfig()

	if err := e.RenderChrony(cfg); err != nil {
		t.Fatalf("RenderChrony: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(e.outputDir, "chrony", "chrony.conf"))
	content := string(data)

	// WAN interface (eth0, DHCP) should NOT have an allow line.
	// Only LAN (192.168.1.0/24) should be allowed.
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "allow ") {
			net := strings.TrimPrefix(line, "allow ")
			if net != "192.168.1.0/24" {
				t.Errorf("unexpected allow network: %s", net)
			}
		}
	}
}
