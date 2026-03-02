package parental

import (
	"testing"
	"time"

	"github.com/scoututm/scoutd/internal/config"
)

func makeTime(weekday time.Weekday, hour, min int) time.Time {
	// Use a known date: 2024-01-01 is Monday.
	// Adjust to desired weekday.
	base := time.Date(2024, 1, 1, hour, min, 0, 0, time.UTC) // Monday
	offset := int(weekday) - int(time.Monday)
	if offset < 0 {
		offset += 7
	}
	return base.AddDate(0, 0, offset)
}

func TestIsScheduleActive_Disabled(t *testing.T) {
	sched := config.Schedule{Enabled: false, StartTime: "21:00", EndTime: "07:00"}
	now := makeTime(time.Monday, 22, 0)
	if IsScheduleActive(sched, now) {
		t.Error("disabled schedule should not be active")
	}
}

func TestIsScheduleActive_OvernightActive(t *testing.T) {
	sched := config.Schedule{Enabled: true, StartTime: "21:00", EndTime: "07:00"}
	// 22:00 should be blocked.
	now := makeTime(time.Monday, 22, 0)
	if !IsScheduleActive(sched, now) {
		t.Error("22:00 should be within 21:00-07:00 overnight window")
	}
}

func TestIsScheduleActive_OvernightMorning(t *testing.T) {
	sched := config.Schedule{Enabled: true, StartTime: "21:00", EndTime: "07:00"}
	// 05:00 should be blocked (still in overnight window).
	now := makeTime(time.Monday, 5, 0)
	if !IsScheduleActive(sched, now) {
		t.Error("05:00 should be within 21:00-07:00 overnight window")
	}
}

func TestIsScheduleActive_OvernightExpired(t *testing.T) {
	sched := config.Schedule{Enabled: true, StartTime: "21:00", EndTime: "07:00"}
	// 08:00 should NOT be blocked.
	now := makeTime(time.Monday, 8, 0)
	if IsScheduleActive(sched, now) {
		t.Error("08:00 should not be within 21:00-07:00 overnight window")
	}
}

func TestIsScheduleActive_SameDayWindow(t *testing.T) {
	sched := config.Schedule{Enabled: true, StartTime: "09:00", EndTime: "17:00"}
	// 12:00 should be blocked.
	now := makeTime(time.Monday, 12, 0)
	if !IsScheduleActive(sched, now) {
		t.Error("12:00 should be within 09:00-17:00 window")
	}
}

func TestIsScheduleActive_SameDayExpired(t *testing.T) {
	sched := config.Schedule{Enabled: true, StartTime: "09:00", EndTime: "17:00"}
	// 18:00 should NOT be blocked.
	now := makeTime(time.Monday, 18, 0)
	if IsScheduleActive(sched, now) {
		t.Error("18:00 should not be within 09:00-17:00 window")
	}
}

func TestIsScheduleActive_WrongDay(t *testing.T) {
	sched := config.Schedule{
		Enabled:   true,
		StartTime: "21:00",
		EndTime:   "07:00",
		Days:      []string{"monday", "tuesday"},
	}
	// Wednesday 22:00 — not a scheduled day.
	now := makeTime(time.Wednesday, 22, 0)
	if IsScheduleActive(sched, now) {
		t.Error("should not be active on Wednesday when days are Monday+Tuesday")
	}
}

func TestIsScheduleActive_CorrectDay(t *testing.T) {
	sched := config.Schedule{
		Enabled:   true,
		StartTime: "21:00",
		EndTime:   "07:00",
		Days:      []string{"monday", "tuesday"},
	}
	// Monday 22:00 — scheduled day.
	now := makeTime(time.Monday, 22, 0)
	if !IsScheduleActive(sched, now) {
		t.Error("should be active on Monday at 22:00")
	}
}

func TestIsScheduleActive_EmptyDays(t *testing.T) {
	sched := config.Schedule{
		Enabled:   true,
		StartTime: "21:00",
		EndTime:   "07:00",
		Days:      []string{},
	}
	// Any day at 22:00 — empty days means every day.
	now := makeTime(time.Friday, 22, 0)
	if !IsScheduleActive(sched, now) {
		t.Error("empty days should mean every day")
	}
}

func TestBlockedDeviceIPs(t *testing.T) {
	cfg := &config.Config{
		Parental: config.ParentalConfig{
			Profiles: []config.ParentalProfile{
				{
					ID:   "prof1",
					Name: "Kids",
					Schedule: config.Schedule{
						Enabled:   true,
						StartTime: "21:00",
						EndTime:   "07:00",
					},
				},
				{
					ID:   "prof2",
					Name: "Teens",
					Schedule: config.Schedule{
						Enabled: false,
					},
				},
			},
			Devices: []config.ParentalDevice{
				{MAC: "aa:bb:cc:dd:ee:01", ProfileID: "prof1", IP: "192.168.1.101"},
				{MAC: "aa:bb:cc:dd:ee:02", ProfileID: "prof2", IP: "192.168.1.102"},
				{MAC: "aa:bb:cc:dd:ee:03", ProfileID: "prof1", IP: "192.168.1.103"},
			},
		},
	}

	// At 22:00 — prof1 schedule is active, prof2 is disabled.
	now := makeTime(time.Monday, 22, 0)
	blocked := BlockedDeviceIPs(cfg, now)

	if len(blocked) != 2 {
		t.Fatalf("expected 2 blocked devices, got %d", len(blocked))
	}

	expected := map[string]bool{"192.168.1.101": true, "192.168.1.103": true}
	for _, ip := range blocked {
		if !expected[ip] {
			t.Errorf("unexpected blocked IP: %s", ip)
		}
	}
}

func TestBlockedDeviceIPs_NoScheduleActive(t *testing.T) {
	cfg := &config.Config{
		Parental: config.ParentalConfig{
			Profiles: []config.ParentalProfile{
				{
					ID:   "prof1",
					Name: "Kids",
					Schedule: config.Schedule{
						Enabled:   true,
						StartTime: "21:00",
						EndTime:   "07:00",
					},
				},
			},
			Devices: []config.ParentalDevice{
				{MAC: "aa:bb:cc:dd:ee:01", ProfileID: "prof1", IP: "192.168.1.101"},
			},
		},
	}

	// At 12:00 — schedule is not active.
	now := makeTime(time.Monday, 12, 0)
	blocked := BlockedDeviceIPs(cfg, now)

	if len(blocked) != 0 {
		t.Fatalf("expected 0 blocked devices at midday, got %d", len(blocked))
	}
}

func TestParseHHMM_InvalidDigits(t *testing.T) {
	_, _, ok := parseHHMM("ab:cd")
	if ok {
		t.Error("parseHHMM should reject non-digit characters")
	}
}

func TestBlockedDeviceIPs_SkipEmptyIP(t *testing.T) {
	cfg := &config.Config{
		Parental: config.ParentalConfig{
			Profiles: []config.ParentalProfile{
				{
					ID:       "prof1",
					Name:     "Kids",
					Schedule: config.Schedule{Enabled: true, StartTime: "21:00", EndTime: "07:00"},
				},
			},
			Devices: []config.ParentalDevice{
				{MAC: "aa:bb:cc:dd:ee:01", ProfileID: "prof1", IP: ""},
			},
		},
	}

	now := makeTime(time.Monday, 22, 0)
	blocked := BlockedDeviceIPs(cfg, now)

	if len(blocked) != 0 {
		t.Fatalf("expected 0 blocked devices with empty IP, got %d", len(blocked))
	}
}
