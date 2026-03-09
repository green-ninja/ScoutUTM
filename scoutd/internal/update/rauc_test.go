package update

import "testing"

func TestVerifyBundle_DevMode(t *testing.T) {
	r := NewRAUC(true)
	info, err := r.VerifyBundle("/tmp/test.raucb")
	if err != nil {
		t.Fatalf("VerifyBundle: %v", err)
	}
	if info.Compatible != "scoututm-h4plus" {
		t.Errorf("compatible = %q, want scoututm-h4plus", info.Compatible)
	}
}

func TestInstallBundle_DevMode(t *testing.T) {
	r := NewRAUC(true)
	if err := r.InstallBundle("/tmp/test.raucb"); err != nil {
		t.Fatalf("InstallBundle: %v", err)
	}
}

func TestGetSlotStatus_DevMode(t *testing.T) {
	r := NewRAUC(true)
	slots, err := r.GetSlotStatus()
	if err != nil {
		t.Fatalf("GetSlotStatus: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("got %d slots, want 2", len(slots))
	}
	if slots[0].State != "booted" {
		t.Errorf("slot 0 state = %q, want booted", slots[0].State)
	}
	if slots[1].State != "inactive" {
		t.Errorf("slot 1 state = %q, want inactive", slots[1].State)
	}
}

func TestMarkGood_DevMode(t *testing.T) {
	r := NewRAUC(true)
	if err := r.MarkGood(); err != nil {
		t.Fatalf("MarkGood: %v", err)
	}
}

func TestTruncateOutput(t *testing.T) {
	short := "short error"
	if got := truncateOutput([]byte(short)); got != short {
		t.Errorf("truncateOutput(%q) = %q", short, got)
	}

	long := make([]byte, 500)
	for i := range long {
		long[i] = 'x'
	}
	got := truncateOutput(long)
	if len(got) > 260 { // 256 + "..."
		t.Errorf("truncateOutput(500 bytes) len = %d, want <= 260", len(got))
	}
}
