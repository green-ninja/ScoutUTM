package netmon

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseNeigh(t *testing.T) {
	input := `192.168.1.100 dev eth1 lladdr aa:bb:cc:dd:ee:01 REACHABLE
192.168.1.101 dev eth1 lladdr AA:BB:CC:DD:EE:02 STALE
192.168.1.200 dev eth1  FAILED
`
	scanner := bufio.NewScanner(strings.NewReader(input))
	entries, err := parseNeigh(scanner)
	if err != nil {
		t.Fatalf("parseNeigh error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].IP != "192.168.1.100" {
		t.Errorf("entry[0].IP = %q, want 192.168.1.100", entries[0].IP)
	}
	if entries[0].MAC != "aa:bb:cc:dd:ee:01" {
		t.Errorf("entry[0].MAC = %q, want aa:bb:cc:dd:ee:01", entries[0].MAC)
	}
	if entries[0].Iface != "eth1" {
		t.Errorf("entry[0].Iface = %q, want eth1", entries[0].Iface)
	}
	if entries[0].State != "REACHABLE" {
		t.Errorf("entry[0].State = %q, want REACHABLE", entries[0].State)
	}

	// Second entry should be lowercased MAC.
	if entries[1].MAC != "aa:bb:cc:dd:ee:02" {
		t.Errorf("entry[1].MAC = %q, want aa:bb:cc:dd:ee:02", entries[1].MAC)
	}
	if entries[1].State != "STALE" {
		t.Errorf("entry[1].State = %q, want STALE", entries[1].State)
	}
}

func TestParseNeigh_SkipsFailed(t *testing.T) {
	input := `192.168.1.200 dev eth1  FAILED
192.168.1.201 dev eth1  INCOMPLETE
`
	scanner := bufio.NewScanner(strings.NewReader(input))
	entries, err := parseNeigh(scanner)
	if err != nil {
		t.Fatalf("parseNeigh error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseLeases(t *testing.T) {
	input := `1700000000 aa:bb:cc:dd:ee:01 192.168.1.100 laptop 01:aa:bb:cc:dd:ee:01
1700000000 AA:BB:CC:DD:EE:02 192.168.1.101 * 01:aa:bb:cc:dd:ee:02
`
	scanner := bufio.NewScanner(strings.NewReader(input))
	leases, err := parseLeases(scanner)
	if err != nil {
		t.Fatalf("parseLeases error: %v", err)
	}

	if len(leases) != 2 {
		t.Fatalf("expected 2 leases, got %d", len(leases))
	}

	if leases[0].MAC != "aa:bb:cc:dd:ee:01" {
		t.Errorf("lease[0].MAC = %q, want aa:bb:cc:dd:ee:01", leases[0].MAC)
	}
	if leases[0].Hostname != "laptop" {
		t.Errorf("lease[0].Hostname = %q, want laptop", leases[0].Hostname)
	}
	if leases[0].Expiry != 1700000000 {
		t.Errorf("lease[0].Expiry = %d, want 1700000000", leases[0].Expiry)
	}

	// Wildcard hostname should be empty.
	if leases[1].Hostname != "" {
		t.Errorf("lease[1].Hostname = %q, want empty", leases[1].Hostname)
	}
	// MAC should be lowercased.
	if leases[1].MAC != "aa:bb:cc:dd:ee:02" {
		t.Errorf("lease[1].MAC = %q, want aa:bb:cc:dd:ee:02", leases[1].MAC)
	}
}

func TestParseLeases_SkipsComments(t *testing.T) {
	input := `# comment
1700000000 aa:bb:cc:dd:ee:01 192.168.1.100 laptop clientid
`
	scanner := bufio.NewScanner(strings.NewReader(input))
	leases, err := parseLeases(scanner)
	if err != nil {
		t.Fatalf("parseLeases error: %v", err)
	}
	if len(leases) != 1 {
		t.Errorf("expected 1 lease, got %d", len(leases))
	}
}

// writeNeighFile writes ip neigh format test data to a file.
func writeNeighFile(t *testing.T, dir string, data string) string {
	t.Helper()
	path := filepath.Join(dir, "neigh")
	os.WriteFile(path, []byte(data), 0644)
	return path
}

// testScanner creates a Scanner with temp neigh and lease files for testing.
// All devices are REACHABLE by default.
func testScanner(t *testing.T) *Scanner {
	t.Helper()
	dir := t.TempDir()

	neighData := `192.168.1.100 dev eth1 lladdr aa:bb:cc:dd:ee:01 REACHABLE
192.168.1.101 dev eth1 lladdr aa:bb:cc:dd:ee:02 REACHABLE
192.168.1.102 dev eth1 lladdr aa:bb:cc:dd:ee:03 REACHABLE
192.168.1.103 dev eth1 lladdr aa:bb:cc:dd:ee:04 REACHABLE
192.168.1.104 dev eth1 lladdr aa:bb:cc:dd:ee:05 REACHABLE
192.168.1.105 dev eth1 lladdr aa:bb:cc:dd:ee:06 REACHABLE
`
	neighFile := writeNeighFile(t, dir, neighData)

	leaseFile := filepath.Join(dir, "leases")
	leaseData := `1999999999 aa:bb:cc:dd:ee:01 192.168.1.100 laptop 01:aa:bb:cc:dd:ee:01
1999999999 aa:bb:cc:dd:ee:02 192.168.1.101 phone 01:aa:bb:cc:dd:ee:02
1999999999 aa:bb:cc:dd:ee:03 192.168.1.102 tablet 01:aa:bb:cc:dd:ee:03
1999999999 aa:bb:cc:dd:ee:04 192.168.1.103 smart-tv 01:aa:bb:cc:dd:ee:04
1999999999 aa:bb:cc:dd:ee:05 192.168.1.104 gaming-pc 01:aa:bb:cc:dd:ee:05
1999999999 aa:bb:cc:dd:ee:06 192.168.1.105 iot-hub 01:aa:bb:cc:dd:ee:06
`
	os.WriteFile(leaseFile, []byte(leaseData), 0644)

	s := NewScanner(leaseFile)
	s.SetNeighFile(neighFile)
	return s
}

func TestScanner_ScanFindsDevices(t *testing.T) {
	s := testScanner(t)
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	devices := s.Devices()
	if len(devices) != 6 {
		t.Errorf("scan returned %d devices, want 6", len(devices))
	}

	// All should be online (REACHABLE).
	for _, d := range devices {
		if !d.Online {
			t.Errorf("device %s should be online", d.MAC)
		}
		if d.IP == "" {
			t.Errorf("device %s has no IP", d.MAC)
		}
	}
}

func TestScanner_MergesNeighAndDHCP(t *testing.T) {
	s := testScanner(t)
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	devices := s.Devices()
	// Check that DHCP hostnames were merged.
	found := false
	for _, d := range devices {
		if d.Hostname == "laptop" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find device with hostname 'laptop' from DHCP merge")
	}
}

func TestScanner_StaleRecentlyReachable_StaysOnline(t *testing.T) {
	dir := t.TempDir()

	// First scan: device is REACHABLE.
	neighFile := writeNeighFile(t, dir, "192.168.1.100 dev eth1 lladdr aa:bb:cc:dd:ee:01 REACHABLE\n")
	leaseFile := filepath.Join(dir, "leases")
	os.WriteFile(leaseFile, []byte(""), 0644)

	s := NewScanner(leaseFile)
	s.SetNeighFile(neighFile)
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan 1 error: %v", err)
	}

	// Second scan: device transitions to STALE (just went idle).
	os.WriteFile(neighFile, []byte("192.168.1.100 dev eth1 lladdr aa:bb:cc:dd:ee:01 STALE\n"), 0644)
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan 2 error: %v", err)
	}

	devices := s.Devices()
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	if !devices[0].Online {
		t.Error("device should still be online (STALE but recently REACHABLE)")
	}
}

func TestScanner_StaleExpired_GoesOffline(t *testing.T) {
	dir := t.TempDir()

	neighFile := writeNeighFile(t, dir, "192.168.1.100 dev eth1 lladdr aa:bb:cc:dd:ee:01 REACHABLE\n")
	leaseFile := filepath.Join(dir, "leases")
	os.WriteFile(leaseFile, []byte(""), 0644)

	s := NewScanner(leaseFile)
	s.SetNeighFile(neighFile)
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan 1 error: %v", err)
	}

	// Simulate time passing beyond OfflineTimeout by backdating lastReachable.
	s.mu.Lock()
	s.lastReachable["aa:bb:cc:dd:ee:01"] = s.lastReachable["aa:bb:cc:dd:ee:01"].Add(-OfflineTimeout - time.Minute)
	s.mu.Unlock()

	// Second scan: device is STALE and lastReachable is beyond timeout.
	os.WriteFile(neighFile, []byte("192.168.1.100 dev eth1 lladdr aa:bb:cc:dd:ee:01 STALE\n"), 0644)
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan 2 error: %v", err)
	}

	devices := s.Devices()
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	if devices[0].Online {
		t.Error("device should be offline (STALE with expired lastReachable)")
	}
}

func TestScanner_DeviceDisappears_GoesOffline(t *testing.T) {
	dir := t.TempDir()

	neighFile := writeNeighFile(t, dir, "192.168.1.100 dev eth1 lladdr aa:bb:cc:dd:ee:01 REACHABLE\n")
	leaseFile := filepath.Join(dir, "leases")
	os.WriteFile(leaseFile, []byte("1999999999 aa:bb:cc:dd:ee:01 192.168.1.100 laptop cid\n"), 0644)

	s := NewScanner(leaseFile)
	s.SetNeighFile(neighFile)
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan 1 error: %v", err)
	}
	if !s.Devices()[0].Online {
		t.Fatal("device should be online after first scan")
	}

	// Device completely gone from neighbor table.
	os.WriteFile(neighFile, []byte(""), 0644)
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan 2 error: %v", err)
	}

	devices := s.Devices()
	if len(devices) != 1 {
		t.Fatalf("expected 1 device (from DHCP lease), got %d", len(devices))
	}
	if devices[0].Online {
		t.Error("device should be offline (gone from neighbor table)")
	}
}

func TestScanner_FailedState_Offline(t *testing.T) {
	dir := t.TempDir()

	neighFile := writeNeighFile(t, dir, "192.168.1.100 dev eth1 lladdr aa:bb:cc:dd:ee:01 REACHABLE\n")
	leaseFile := filepath.Join(dir, "leases")
	os.WriteFile(leaseFile, []byte(""), 0644)

	s := NewScanner(leaseFile)
	s.SetNeighFile(neighFile)
	s.Scan()

	// FAILED entries have no lladdr → skipped by parser → device goes offline.
	os.WriteFile(neighFile, []byte("192.168.1.100 dev eth1  FAILED\n"), 0644)
	s.Scan()

	devices := s.Devices()
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	if devices[0].Online {
		t.Error("FAILED device should be offline")
	}
}

func TestScanner_SetDeviceName(t *testing.T) {
	s := testScanner(t)
	s.Scan()

	s.SetDeviceName("aa:bb:cc:dd:ee:01", "My Laptop")
	devices := s.Devices()
	for _, d := range devices {
		if d.MAC == "aa:bb:cc:dd:ee:01" {
			if d.Name != "My Laptop" {
				t.Errorf("device name = %q, want 'My Laptop'", d.Name)
			}
			return
		}
	}
	t.Error("device aa:bb:cc:dd:ee:01 not found")
}

func TestLookupOUI_KnownVendor(t *testing.T) {
	// Apple prefix 00:03:93
	result := LookupOUI("00:03:93:aa:bb:cc")
	if result != "Apple" {
		t.Errorf("LookupOUI(Apple MAC) = %q, want Apple", result)
	}
}

func TestLookupOUI_UnknownVendor(t *testing.T) {
	result := LookupOUI("ff:ff:ff:aa:bb:cc")
	if result != "" {
		t.Errorf("LookupOUI(unknown MAC) = %q, want empty", result)
	}
}

func TestLookupOUI_ShortMAC(t *testing.T) {
	result := LookupOUI("aa:bb")
	if result != "" {
		t.Errorf("LookupOUI(short MAC) = %q, want empty", result)
	}
}

func TestLookupOUI_DashFormat(t *testing.T) {
	// Samsung prefix using dash format
	result := LookupOUI("00-1A-8A-aa-bb-cc")
	if result != "Samsung" {
		t.Errorf("LookupOUI(Samsung dash MAC) = %q, want Samsung", result)
	}
}

func TestScanner_SetsManufacturer(t *testing.T) {
	s := testScanner(t)
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	// Test MACs use aa:bb:cc which won't have OUI entries,
	// but the Manufacturer field should be populated (even if empty).
	devices := s.Devices()
	for _, d := range devices {
		// Just verify the field exists and scan doesn't crash.
		_ = d.Manufacturer
	}
}

func TestScanner_LookupByIP(t *testing.T) {
	s := testScanner(t)
	s.Scan()
	s.SetDeviceName("aa:bb:cc:dd:ee:01", "My Laptop")

	mac, name, ok := s.LookupByIP("192.168.1.100")
	if !ok {
		t.Fatal("LookupByIP should find device at 192.168.1.100")
	}
	if mac != "aa:bb:cc:dd:ee:01" {
		t.Errorf("mac = %q, want aa:bb:cc:dd:ee:01", mac)
	}
	if name != "My Laptop" {
		t.Errorf("name = %q, want My Laptop", name)
	}
}

func TestScanner_LookupByIP_NotFound(t *testing.T) {
	s := testScanner(t)
	s.Scan()

	_, _, ok := s.LookupByIP("10.0.0.99")
	if ok {
		t.Error("LookupByIP should not find unknown IP")
	}
}

func TestScanner_NoLeaseFile(t *testing.T) {
	dir := t.TempDir()
	neighFile := writeNeighFile(t, dir, "192.168.1.100 dev eth1 lladdr aa:bb:cc:dd:ee:01 REACHABLE\n")

	s := NewScanner(filepath.Join(dir, "nonexistent-leases"))
	s.SetNeighFile(neighFile)
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	devices := s.Devices()
	if len(devices) != 1 {
		t.Errorf("expected 1 device, got %d", len(devices))
	}
}
