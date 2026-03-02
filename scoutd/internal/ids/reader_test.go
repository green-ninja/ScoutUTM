package ids

import (
	"encoding/json"
	"testing"
)

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer()
	alerts := rb.All()
	if len(alerts) != 0 {
		t.Errorf("empty ring buffer returned %d alerts, want 0", len(alerts))
	}
}

func TestRingBuffer_AddAndRetrieve(t *testing.T) {
	rb := NewRingBuffer()
	rb.Add(Alert{SrcIP: "10.0.0.1", Alert: AlertDetail{SignatureID: 1}})
	rb.Add(Alert{SrcIP: "10.0.0.2", Alert: AlertDetail{SignatureID: 2}})

	alerts := rb.All()
	if len(alerts) != 2 {
		t.Fatalf("got %d alerts, want 2", len(alerts))
	}
	if alerts[0].SrcIP != "10.0.0.1" {
		t.Errorf("alerts[0].SrcIP = %q, want 10.0.0.1", alerts[0].SrcIP)
	}
	if alerts[1].SrcIP != "10.0.0.2" {
		t.Errorf("alerts[1].SrcIP = %q, want 10.0.0.2", alerts[1].SrcIP)
	}
}

func TestRingBuffer_OverflowOrdering(t *testing.T) {
	rb := NewRingBuffer()

	// Fill beyond capacity.
	for i := 0; i < ringCapacity+10; i++ {
		rb.Add(Alert{Alert: AlertDetail{SignatureID: i}})
	}

	alerts := rb.All()
	if len(alerts) != ringCapacity {
		t.Fatalf("got %d alerts, want %d", len(alerts), ringCapacity)
	}

	// Oldest should be #10 (first 10 were overwritten).
	if alerts[0].Alert.SignatureID != 10 {
		t.Errorf("oldest alert SigID = %d, want 10", alerts[0].Alert.SignatureID)
	}
	// Newest should be #109.
	if alerts[ringCapacity-1].Alert.SignatureID != ringCapacity+9 {
		t.Errorf("newest alert SigID = %d, want %d", alerts[ringCapacity-1].Alert.SignatureID, ringCapacity+9)
	}
}

func TestProcessLine_ValidAlert(t *testing.T) {
	r := NewReader("", "", true)

	ev := eveEvent{
		Timestamp: "2025-01-15T10:30:00.123456+0000",
		EventType: "alert",
		SrcIP:     "192.168.1.100",
		SrcPort:   54321,
		DestIP:    "10.0.0.1",
		DestPort:  80,
		Proto:     "TCP",
		Alert: AlertDetail{
			Action:      "allowed",
			GID:         1,
			SignatureID: 2001,
			Rev:         3,
			Signature:   "ET SCAN Test",
			Category:    "Attempted Information Leak",
			Severity:    2,
		},
	}
	line, _ := json.Marshal(ev)
	r.processLine(line)

	alerts := r.alerts.All()
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Alert.SignatureID != 2001 {
		t.Errorf("SignatureID = %d, want 2001", alerts[0].Alert.SignatureID)
	}
	if alerts[0].SrcIP != "192.168.1.100" {
		t.Errorf("SrcIP = %q, want 192.168.1.100", alerts[0].SrcIP)
	}
}

func TestProcessLine_NonAlertIgnored(t *testing.T) {
	r := NewReader("", "", true)

	ev := eveEvent{
		EventType: "stats",
		Timestamp: "2025-01-15T10:30:00.000000+0000",
	}
	line, _ := json.Marshal(ev)
	r.processLine(line)

	alerts := r.alerts.All()
	if len(alerts) != 0 {
		t.Errorf("non-alert event should be ignored, got %d alerts", len(alerts))
	}
}

func TestProcessLine_InvalidJSONIgnored(t *testing.T) {
	r := NewReader("", "", true)
	r.processLine([]byte("not json"))

	alerts := r.alerts.All()
	if len(alerts) != 0 {
		t.Errorf("invalid JSON should be ignored, got %d alerts", len(alerts))
	}
}
