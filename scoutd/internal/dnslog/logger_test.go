package dnslog

import (
	"os"
	"path/filepath"
	"testing"
)

type mockLookup struct {
	devices map[string]struct{ mac, name string }
}

func (m *mockLookup) LookupByIP(ip string) (string, string, bool) {
	d, ok := m.devices[ip]
	if !ok {
		return "", "", false
	}
	return d.mac, d.name, true
}

func TestParseUnboundLine_QueryOnly(t *testing.T) {
	lookup := &mockLookup{
		devices: map[string]struct{ mac, name string }{
			"192.168.1.100": {mac: "aa:bb:cc:dd:ee:01", name: "laptop"},
		},
	}
	logger := NewLogger("", lookup)

	// Query-only line (log-queries format, no response code).
	line := `[1705314645] unbound[1234:0] info: 192.168.1.100 www.google.com. A IN`
	entry, ok := logger.ParseUnboundLine(line)

	if !ok {
		t.Fatal("expected line to be parsed")
	}
	if entry.DeviceMAC != "aa:bb:cc:dd:ee:01" {
		t.Errorf("device MAC = %q, want aa:bb:cc:dd:ee:01", entry.DeviceMAC)
	}
	if entry.DeviceIP != "192.168.1.100" {
		t.Errorf("device IP = %q, want 192.168.1.100", entry.DeviceIP)
	}
	if entry.Domain != "www.google.com" {
		t.Errorf("domain = %q, want www.google.com", entry.Domain)
	}
	if entry.DeviceName != "laptop" {
		t.Errorf("device name = %q, want laptop", entry.DeviceName)
	}
	if entry.Action != "allow" {
		t.Errorf("action = %q, want allow", entry.Action)
	}
	if entry.Timestamp != "2024-01-15 10:30:45" {
		t.Errorf("timestamp = %q, want 2024-01-15 10:30:45", entry.Timestamp)
	}
}

func TestParseUnboundLine_ReplyAllowed(t *testing.T) {
	logger := NewLogger("", nil)

	line := `[1705314645] unbound[1234:0] info: 192.168.1.100 www.google.com. A IN NOERROR 0.012345 1 55`
	entry, ok := logger.ParseUnboundLine(line)

	if !ok {
		t.Fatal("expected reply line to be parsed")
	}
	if entry.Domain != "www.google.com" {
		t.Errorf("domain = %q, want www.google.com", entry.Domain)
	}
	if entry.Action != "allow" {
		t.Errorf("action = %q, want allow", entry.Action)
	}
}

func TestParseUnboundLine_ReplyBlocked(t *testing.T) {
	logger := NewLogger("", nil)

	line := `[1705314645] unbound[1234:0] info: 192.168.1.200 pornhub.com. A IN NXDOMAIN 0.000000 0 40`
	entry, ok := logger.ParseUnboundLine(line)

	if !ok {
		t.Fatal("expected reply line to be parsed")
	}
	if entry.Domain != "pornhub.com" {
		t.Errorf("domain = %q, want pornhub.com", entry.Domain)
	}
	if entry.Action != "block" {
		t.Errorf("action = %q, want block", entry.Action)
	}
}

func TestParseUnboundLine_ReplyRefused(t *testing.T) {
	logger := NewLogger("", nil)

	line := `[1705314645] unbound[1234:0] info: 192.168.1.200 example.com. A IN REFUSED 0.000000 0 40`
	entry, ok := logger.ParseUnboundLine(line)

	if !ok {
		t.Fatal("expected reply line to be parsed")
	}
	if entry.Action != "block" {
		t.Errorf("action = %q, want block", entry.Action)
	}
}

func TestParseUnboundLine_NonQueryLine(t *testing.T) {
	logger := NewLogger("", nil)
	line := `[1705314645] unbound[1234:0] notice: init module 0: validator`
	_, ok := logger.ParseUnboundLine(line)
	if ok {
		t.Error("non-query line should not be parsed")
	}
}

func TestParseUnboundLine_UnknownDevice(t *testing.T) {
	logger := NewLogger("", &mockLookup{devices: map[string]struct{ mac, name string }{}})

	line := `[1705314645] unbound[1234:0] info: 10.0.0.5 example.com. A IN`
	entry, ok := logger.ParseUnboundLine(line)

	if !ok {
		t.Fatal("expected line to be parsed")
	}
	if entry.DeviceMAC != "" {
		t.Errorf("device MAC = %q, want empty for unknown device", entry.DeviceMAC)
	}
	if entry.DeviceName != "" {
		t.Errorf("device name = %q, want empty for unknown device", entry.DeviceName)
	}
}

func TestRecentEntries_RingBuffer(t *testing.T) {
	logger := NewLogger("", nil)

	// Add more than MaxEntries.
	for i := 0; i < MaxEntries+10; i++ {
		logger.AddEntry(Entry{Domain: string(rune('A' + (i % 26)))})
	}

	entries := logger.RecentEntries(5)
	if len(entries) != 5 {
		t.Fatalf("got %d entries, want 5", len(entries))
	}

	// Should be the last 5 entries added.
	all := logger.RecentEntries(MaxEntries)
	if len(all) != MaxEntries {
		t.Fatalf("got %d entries, want %d", len(all), MaxEntries)
	}
}

func TestRecentEntries_Empty(t *testing.T) {
	logger := NewLogger("", nil)
	entries := logger.RecentEntries(10)
	if entries != nil {
		t.Errorf("expected nil for empty buffer, got %d entries", len(entries))
	}
}

func TestRotateLog_BelowMaxSize(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, DefaultLogFile)
	os.WriteFile(logPath, []byte("small"), 0644)

	err := RotateLog(dir, 1024, 7)
	if err != nil {
		t.Fatalf("RotateLog: %v", err)
	}

	// File should still exist (not rotated).
	if _, err := os.Stat(logPath); err != nil {
		t.Error("log file should still exist")
	}
}

func TestRotateLog_AboveMaxSize(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, DefaultLogFile)
	// Create a file larger than threshold.
	data := make([]byte, 2048)
	os.WriteFile(logPath, data, 0644)

	err := RotateLog(dir, 1024, 7)
	if err != nil {
		t.Fatalf("RotateLog: %v", err)
	}

	// Original file should be gone, .1 should exist.
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("original log file should have been renamed")
	}
	if _, err := os.Stat(logPath + ".1"); err != nil {
		t.Error("rotated log file .1 should exist")
	}
}

func TestRotateLog_NonExistent(t *testing.T) {
	dir := t.TempDir()
	err := RotateLog(dir, 1024, 7)
	if err != nil {
		t.Fatalf("RotateLog should not error on missing file: %v", err)
	}
}
