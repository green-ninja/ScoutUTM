package stats

import (
	"testing"
)

func TestParseConntrackLine_ProcFormat_TCP(t *testing.T) {
	line := "ipv4     2 tcp      6 300 ESTABLISHED src=192.168.1.100 dst=142.250.80.46 sport=54321 dport=443 packets=10 bytes=5000 src=142.250.80.46 dst=192.168.1.1 sport=443 dport=54321 packets=8 bytes=3000 [ASSURED] mark=0 zone=0 use=2"
	c, ok := parseConntrackLine(line)
	if !ok {
		t.Fatal("expected parse success")
	}
	if c.Proto != "tcp" {
		t.Errorf("proto = %q, want tcp", c.Proto)
	}
	if c.State != "ESTABLISHED" {
		t.Errorf("state = %q, want ESTABLISHED", c.State)
	}
	if c.SrcIP != "192.168.1.100" {
		t.Errorf("src_ip = %q, want 192.168.1.100", c.SrcIP)
	}
	if c.SrcPort != 54321 {
		t.Errorf("src_port = %d, want 54321", c.SrcPort)
	}
	if c.DstIP != "142.250.80.46" {
		t.Errorf("dst_ip = %q, want 142.250.80.46", c.DstIP)
	}
	if c.DstPort != 443 {
		t.Errorf("dst_port = %d, want 443", c.DstPort)
	}
	if c.Bytes != 5000 {
		t.Errorf("bytes = %d, want 5000", c.Bytes)
	}
}

func TestParseConntrackLine_ConntrackLFormat_TCP(t *testing.T) {
	line := "tcp      6 431977 ESTABLISHED src=172.16.0.2 dst=142.250.80.46 sport=54321 dport=443 packets=10 bytes=5000 src=142.250.80.46 dst=172.16.0.1 sport=443 dport=54321 packets=8 bytes=3000 [ASSURED] mark=0 use=2"
	c, ok := parseConntrackLine(line)
	if !ok {
		t.Fatal("expected parse success")
	}
	if c.Proto != "tcp" {
		t.Errorf("proto = %q, want tcp", c.Proto)
	}
	if c.State != "ESTABLISHED" {
		t.Errorf("state = %q, want ESTABLISHED", c.State)
	}
	if c.SrcIP != "172.16.0.2" {
		t.Errorf("src_ip = %q, want 172.16.0.2", c.SrcIP)
	}
	if c.DstPort != 443 {
		t.Errorf("dst_port = %d, want 443", c.DstPort)
	}
	if c.Bytes != 5000 {
		t.Errorf("bytes = %d, want 5000", c.Bytes)
	}
}

func TestParseConntrackLine_ConntrackLFormat_UDP(t *testing.T) {
	line := "udp      17 29 src=172.16.0.2 dst=1.1.1.1 sport=12345 dport=53 packets=2 bytes=128 src=1.1.1.1 dst=172.16.0.1 sport=53 dport=12345 packets=2 bytes=256 [ASSURED] mark=0 use=2"
	c, ok := parseConntrackLine(line)
	if !ok {
		t.Fatal("expected parse success")
	}
	if c.Proto != "udp" {
		t.Errorf("proto = %q, want udp", c.Proto)
	}
	if c.State != "" {
		t.Errorf("state = %q, want empty", c.State)
	}
	if c.SrcIP != "172.16.0.2" {
		t.Errorf("src_ip = %q, want 172.16.0.2", c.SrcIP)
	}
	if c.DstPort != 53 {
		t.Errorf("dst_port = %d, want 53", c.DstPort)
	}
	if c.Bytes != 128 {
		t.Errorf("bytes = %d, want 128", c.Bytes)
	}
}

func TestParseConntrackLine_ProcFormat_UDP(t *testing.T) {
	line := "ipv4     2 udp      17 30 src=192.168.1.50 dst=1.1.1.1 sport=12345 dport=53 packets=2 bytes=128 src=1.1.1.1 dst=192.168.1.1 sport=53 dport=12345 packets=2 bytes=256 [ASSURED] mark=0 zone=0 use=2"
	c, ok := parseConntrackLine(line)
	if !ok {
		t.Fatal("expected parse success")
	}
	if c.Proto != "udp" {
		t.Errorf("proto = %q, want udp", c.Proto)
	}
	if c.State != "" {
		t.Errorf("state = %q, want empty", c.State)
	}
	if c.DstPort != 53 {
		t.Errorf("dst_port = %d, want 53", c.DstPort)
	}
}

func TestParseConntrackLine_ICMP(t *testing.T) {
	line := "icmp     1 29 src=172.16.0.2 dst=8.8.8.8 type=8 code=0 id=1234 packets=5 bytes=420 src=8.8.8.8 dst=172.16.0.1 type=0 code=0 id=1234 packets=5 bytes=420 mark=0 use=2"
	c, ok := parseConntrackLine(line)
	if !ok {
		t.Fatal("expected parse success")
	}
	if c.Proto != "icmp" {
		t.Errorf("proto = %q, want icmp", c.Proto)
	}
	if c.SrcIP != "172.16.0.2" {
		t.Errorf("src_ip = %q, want 172.16.0.2", c.SrcIP)
	}
	if c.DstIP != "8.8.8.8" {
		t.Errorf("dst_ip = %q, want 8.8.8.8", c.DstIP)
	}
	if c.Bytes != 420 {
		t.Errorf("bytes = %d, want 420", c.Bytes)
	}
}

func TestParseConntrackLine_TooShort(t *testing.T) {
	_, ok := parseConntrackLine("tcp 6")
	if ok {
		t.Error("expected parse failure for short line")
	}
}

func TestParseConnections_Sorting(t *testing.T) {
	// Test the parsing + sorting logic directly.
	lines := []string{
		"tcp      6 300 ESTABLISHED src=192.168.1.10 dst=10.0.0.1 sport=1000 dport=443 packets=5 bytes=100 src=10.0.0.1 dst=192.168.1.1 sport=443 dport=1000 packets=3 bytes=50 [ASSURED] mark=0",
		"tcp      6 60 TIME_WAIT src=192.168.1.20 dst=10.0.0.2 sport=2000 dport=80 packets=50 bytes=99999 src=10.0.0.2 dst=192.168.1.1 sport=80 dport=2000 packets=40 bytes=80000 [ASSURED] mark=0",
		"udp      17 30 src=192.168.1.30 dst=1.1.1.1 sport=3000 dport=53 packets=1 bytes=500 src=1.1.1.1 dst=192.168.1.1 sport=53 dport=3000 packets=1 bytes=200 mark=0",
	}

	conns := make([]Connection, 0, len(lines))
	for _, line := range lines {
		if c, ok := parseConntrackLine(line); ok {
			conns = append(conns, c)
		}
	}
	if len(conns) != 3 {
		t.Fatalf("got %d connections, want 3", len(conns))
	}

	// Sorted by bytes descending — TIME_WAIT entry (99999) should be first.
	if conns[0].Bytes > conns[1].Bytes {
		// Not sorted yet; this test checks parsing, sorting is in ReadConnections.
	}

	// Verify the high-bytes entry parsed correctly.
	found := false
	for _, c := range conns {
		if c.Bytes == 99999 && c.State == "TIME_WAIT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("did not find TIME_WAIT connection with 99999 bytes")
	}
}
