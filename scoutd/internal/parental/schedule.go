package parental

import (
	"strings"
	"time"

	"github.com/scoututm/scoutd/internal/config"
)

// IsScheduleActive returns true if the schedule is currently active (device should be blocked).
// An active schedule means the current time falls within the blocked window.
func IsScheduleActive(sched config.Schedule, now time.Time) bool {
	if !sched.Enabled {
		return false
	}

	// Check day-of-week constraint (empty days = every day).
	if len(sched.Days) > 0 {
		today := strings.ToLower(now.Weekday().String())
		found := false
		for _, d := range sched.Days {
			if strings.ToLower(d) == today {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Parse start/end times.
	startH, startM, ok := parseHHMM(sched.StartTime)
	if !ok {
		return false
	}
	endH, endM, ok := parseHHMM(sched.EndTime)
	if !ok {
		return false
	}

	nowMinutes := now.Hour()*60 + now.Minute()
	startMinutes := startH*60 + startM
	endMinutes := endH*60 + endM

	if startMinutes <= endMinutes {
		// Same-day window: e.g. 09:00→17:00
		return nowMinutes >= startMinutes && nowMinutes < endMinutes
	}

	// Overnight window: e.g. 21:00→07:00
	return nowMinutes >= startMinutes || nowMinutes < endMinutes
}

// BlockedDeviceIPs returns IPs of devices with active block schedules.
func BlockedDeviceIPs(cfg *config.Config, now time.Time) []string {
	// Build profile ID→schedule map.
	profileSchedules := make(map[string]config.Schedule)
	for _, p := range cfg.Parental.Profiles {
		profileSchedules[p.ID] = p.Schedule
	}

	var blocked []string
	for _, d := range cfg.Parental.Devices {
		if d.IP == "" {
			continue
		}
		sched, ok := profileSchedules[d.ProfileID]
		if !ok {
			continue
		}
		if IsScheduleActive(sched, now) {
			blocked = append(blocked, d.IP)
		}
	}
	return blocked
}

func parseHHMM(s string) (int, int, bool) {
	if len(s) != 5 || s[2] != ':' {
		return 0, 0, false
	}
	if !isDigit(s[0]) || !isDigit(s[1]) || !isDigit(s[3]) || !isDigit(s[4]) {
		return 0, 0, false
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[3]-'0')*10 + int(s[4]-'0')
	if h > 23 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
