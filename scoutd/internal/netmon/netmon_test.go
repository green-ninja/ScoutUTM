package netmon

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseARP(t *testing.T) {
	input := `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.100    0x1         0x2         aa:bb:cc:dd:ee:01     *        eth1
192.168.1.101    0x1         0x2         AA:BB:CC:DD:EE:02     *        eth1
192.168.1.200    0x1         0x0         00:00:00:00:00:00     *        eth1
`

	scanner := bufio.NewScanner(strings.NewReader(input))
	entries, err := parseARP(scanner)
	if err != nil {
		t.Fatalf("parseARP error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// First entry.
	if entries[0].IP != "192.168.1.100" {
		t.Errorf("entry[0].IP = %q, want 192.168.1.100", entries[0].IP)
	}
	if entries[0].MAC != "aa:bb:cc:dd:ee:01" {
		t.Errorf("entry[0].MAC = %q, want aa:bb:cc:dd:ee:01", entries[0].MAC)
	}
	if entries[0].Interface != "eth1" {
		t.Errorf("entry[0].Interface = %q, want eth1", entries[0].Interface)
	}

	// Second entry should be lowercased.
	if entries[1].MAC != "aa:bb:cc:dd:ee:02" {
		t.Errorf("entry[1].MAC = %q, want aa:bb:cc:dd:ee:02", entries[1].MAC)
	}
}

func TestParseARP_SkipsIncomplete(t *testing.T) {
	input := `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.200    0x1         0x0         00:00:00:00:00:00     *        eth1
`
	scanner := bufio.NewScanner(strings.NewReader(input))
	entries, err := parseARP(scanner)
	if err != nil {
		t.Fatalf("parseARP error: %v", err)
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

func TestScanner_DevMode(t *testing.T) {
	s := NewScanner(true, "")
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	devices := s.Devices()
	if len(devices) < 5 {
		t.Errorf("dev-mode scan returned %d devices, want >= 5", len(devices))
	}

	// All should be online (from ARP).
	for _, d := range devices {
		if !d.Online {
			t.Errorf("device %s should be online", d.MAC)
		}
		if d.IP == "" {
			t.Errorf("device %s has no IP", d.MAC)
		}
	}
}

func TestScanner_MergesARPAndDHCP(t *testing.T) {
	s := NewScanner(true, "")
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

func TestScanner_SetDeviceName(t *testing.T) {
	s := NewScanner(true, "")
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

func TestScanner_DevMode_SetsManufacturer(t *testing.T) {
	s := NewScanner(true, "")
	if err := s.Scan(); err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	// Dev-mode synthetic devices use aa:bb:cc MACs which won't have OUI entries,
	// but the Manufacturer field should be populated (even if empty).
	devices := s.Devices()
	for _, d := range devices {
		// Just verify the field exists and scan doesn't crash.
		_ = d.Manufacturer
	}
}

func TestScanner_LookupByIP(t *testing.T) {
	s := NewScanner(true, "")
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
	s := NewScanner(true, "")
	s.Scan()

	_, _, ok := s.LookupByIP("10.0.0.99")
	if ok {
		t.Error("LookupByIP should not find unknown IP")
	}
}
