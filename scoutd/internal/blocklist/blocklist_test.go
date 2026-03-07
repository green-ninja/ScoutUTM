package blocklist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteZoneFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rpz-parental-adult.zone")

	if err := WriteZoneFile(path, "adult"); err != nil {
		t.Fatalf("WriteZoneFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading zone file: %v", err)
	}
	content := string(data)

	// Must have RPZ header.
	if !strings.Contains(content, "$SOA") {
		t.Error("zone file missing $SOA header")
	}
	if !strings.Contains(content, "$NS") {
		t.Error("zone file missing $NS header")
	}

	// Must contain known adult domains.
	if !strings.Contains(content, "pornhub.com CNAME .") {
		t.Error("zone file missing pornhub.com entry")
	}
	// Must contain wildcard entries.
	if !strings.Contains(content, "*.pornhub.com CNAME .") {
		t.Error("zone file missing *.pornhub.com wildcard entry")
	}
}

func TestWriteZoneFile_UnknownCategory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rpz-parental-unknown.zone")

	if err := WriteZoneFile(path, "unknown"); err != nil {
		t.Fatalf("WriteZoneFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading zone file: %v", err)
	}
	content := string(data)

	// Should have header but no domain entries.
	if !strings.Contains(content, "$SOA") {
		t.Error("zone file missing $SOA header")
	}
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 header lines, got %d", len(lines))
	}
}

func TestWriteParentalZones(t *testing.T) {
	dir := t.TempDir()

	cats := []string{"adult", "gambling"}
	if err := WriteParentalZones(cats, dir); err != nil {
		t.Fatalf("WriteParentalZones: %v", err)
	}

	// Both files should exist with domain content.
	for _, cat := range cats {
		path := filepath.Join(dir, "rpz-parental-"+cat+".zone")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("zone file %s not created: %v", cat, err)
			continue
		}
		if !strings.Contains(string(data), "CNAME .") {
			t.Errorf("zone file %s has no CNAME entries", cat)
		}
	}
}

func TestWriteParentalZones_OverwritesStubs(t *testing.T) {
	dir := t.TempDir()

	// Write an empty stub first (like vagrant-provision.sh does).
	path := filepath.Join(dir, "rpz-parental-adult.zone")
	stub := []byte("$SOA 0 0 IN SOA localhost. root.localhost. 1 86400 7200 2592000 86400\n$NS 0 0 IN NS localhost.\n")
	if err := os.WriteFile(path, stub, 0644); err != nil {
		t.Fatal(err)
	}

	// WriteParentalZones should overwrite the stub with real domains.
	if err := WriteParentalZones([]string{"adult"}, dir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "pornhub.com CNAME .") {
		t.Error("WriteParentalZones did not overwrite stub with real domains")
	}
}

func TestCategories(t *testing.T) {
	cats := Categories()
	if len(cats) != 5 {
		t.Errorf("expected 5 categories, got %d", len(cats))
	}
}

func TestAllCategoriesHaveDomains(t *testing.T) {
	for _, cat := range Categories() {
		domains, ok := categoryDomains[cat]
		if !ok || len(domains) == 0 {
			t.Errorf("category %q has no domains", cat)
		}
	}
}
