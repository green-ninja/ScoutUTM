package stats

import "testing"

func TestHistoryRecordAndQuery(t *testing.T) {
	h := NewHistory(10)

	// Record 5 samples, 10s apart.
	for i := 0; i < 5; i++ {
		h.Record(Snapshot{
			Timestamp: int64(1000 + i*10),
			WANRxBps:  uint64(i * 100),
			WANTxBps:  uint64(i * 50),
		})
	}

	// Query last 25 seconds (should get samples at t=1020, 1030, 1040).
	samples := h.Query(25)
	if len(samples) != 3 {
		t.Fatalf("expected 3 samples, got %d", len(samples))
	}
	if samples[0].Timestamp != 1020 {
		t.Errorf("expected first timestamp 1020, got %d", samples[0].Timestamp)
	}
	if samples[2].RxBps != 400 {
		t.Errorf("expected last RxBps 400, got %d", samples[2].RxBps)
	}

	// Query all (large window).
	all := h.Query(9999)
	if len(all) != 5 {
		t.Fatalf("expected 5 samples, got %d", len(all))
	}
}

func TestHistoryWraparound(t *testing.T) {
	h := NewHistory(4)

	// Record 6 samples into a buffer of size 4.
	for i := 0; i < 6; i++ {
		h.Record(Snapshot{
			Timestamp: int64(100 + i*10),
			WANRxBps:  uint64(i),
		})
	}

	// Should only have the last 4.
	all := h.Query(9999)
	if len(all) != 4 {
		t.Fatalf("expected 4 samples, got %d", len(all))
	}
	if all[0].Timestamp != 120 {
		t.Errorf("expected oldest timestamp 120, got %d", all[0].Timestamp)
	}
	if all[3].Timestamp != 150 {
		t.Errorf("expected newest timestamp 150, got %d", all[3].Timestamp)
	}
}

func TestHistoryEmpty(t *testing.T) {
	h := NewHistory(10)
	samples := h.Query(300)
	if samples != nil {
		t.Errorf("expected nil for empty history, got %v", samples)
	}
}
