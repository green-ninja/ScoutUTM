package stats

import "testing"

func TestCollector_DevMode_ReturnsData(t *testing.T) {
	c := NewCollector(true, "eth0")
	snap, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if snap.Timestamp == 0 {
		t.Error("Timestamp should be non-zero")
	}
	if snap.MemTotalMB == 0 {
		t.Error("MemTotalMB should be non-zero in dev mode")
	}
	if snap.MemUsedMB == 0 {
		t.Error("MemUsedMB should be non-zero in dev mode")
	}
}

func TestCollector_DevMode_CPUInRange(t *testing.T) {
	c := NewCollector(true, "eth0")

	for i := 0; i < 10; i++ {
		snap, err := c.Collect()
		if err != nil {
			t.Fatalf("Collect: %v", err)
		}
		if snap.CPUPercent < 25 || snap.CPUPercent > 35 {
			t.Errorf("CPUPercent = %f, want 25-35", snap.CPUPercent)
		}
	}
}

func TestCollector_DevMode_UptimeIncreases(t *testing.T) {
	c := NewCollector(true, "eth0")
	snap, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	// Uptime should be >= 0 (just started).
	if snap.UptimeSeconds > 60 {
		t.Errorf("UptimeSeconds = %d, expected small value for fresh collector", snap.UptimeSeconds)
	}
}

func TestCollector_DevMode_DNSCountsAccumulate(t *testing.T) {
	c := NewCollector(true, "eth0")
	var prev Snapshot
	for i := 0; i < 5; i++ {
		snap, err := c.Collect()
		if err != nil {
			t.Fatalf("Collect: %v", err)
		}
		if i > 0 && snap.DNSQueries <= prev.DNSQueries {
			t.Errorf("DNSQueries should accumulate: got %d, prev %d", snap.DNSQueries, prev.DNSQueries)
		}
		prev = snap
	}
}

func TestParseUnboundStats(t *testing.T) {
	output := `thread0.num.queries=100
total.num.queries=12345
total.num.cachehits=5000
num.answer.rcode.NOERROR=10000
num.answer.rcode.NXDOMAIN=42
num.answer.rcode.SERVFAIL=3
`
	s := parseUnboundStats(output)
	if s.totalQueries != 12345 {
		t.Errorf("totalQueries = %d, want 12345", s.totalQueries)
	}
	if s.blockedRPZ != 42 {
		t.Errorf("blockedRPZ = %d, want 42", s.blockedRPZ)
	}
}

func TestParseUnboundStats_Empty(t *testing.T) {
	s := parseUnboundStats("")
	if s.totalQueries != 0 || s.blockedRPZ != 0 {
		t.Errorf("expected zeros for empty input, got queries=%d blocked=%d", s.totalQueries, s.blockedRPZ)
	}
}
