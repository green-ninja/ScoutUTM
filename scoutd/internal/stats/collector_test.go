package stats

import "testing"

func TestCollector_ReturnsData(t *testing.T) {
	c := NewCollector("eth0")
	snap, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if snap.Timestamp == 0 {
		t.Error("Timestamp should be non-zero")
	}
	if snap.MemTotalMB == 0 {
		t.Error("MemTotalMB should be non-zero")
	}
}

func TestCollector_CPUNonNegative(t *testing.T) {
	c := NewCollector("eth0")

	for i := 0; i < 3; i++ {
		snap, err := c.Collect()
		if err != nil {
			t.Fatalf("Collect: %v", err)
		}
		if snap.CPUPercent < 0 || snap.CPUPercent > 100 {
			t.Errorf("CPUPercent = %f, want 0-100", snap.CPUPercent)
		}
	}
}

func TestCollector_UptimeNonZero(t *testing.T) {
	c := NewCollector("eth0")
	snap, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	// System has been up for at least a second.
	if snap.UptimeSeconds == 0 {
		t.Error("UptimeSeconds should be non-zero")
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
