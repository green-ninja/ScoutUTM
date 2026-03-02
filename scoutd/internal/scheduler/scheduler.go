package scheduler

import (
	"context"
	"log"
	"time"
)

// Task defines a periodic job.
type Task struct {
	Name     string
	Interval time.Duration
	RunAt    string // Optional "HH:MM" for daily-at-time tasks.
	Fn       func(ctx context.Context) error
}

// Scheduler runs periodic tasks.
type Scheduler struct {
	tasks   []Task
	devMode bool
	nextRun map[string]time.Time
}

// New creates a scheduler.
func New(devMode bool) *Scheduler {
	return &Scheduler{
		devMode: devMode,
		nextRun: make(map[string]time.Time),
	}
}

// Register adds a task to the scheduler.
func (s *Scheduler) Register(t Task) {
	s.tasks = append(s.tasks, t)
}

// Start runs the scheduler loop. In dev-mode, blocks until context is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	if s.devMode {
		<-ctx.Done()
		return
	}

	// Compute initial next-run times.
	now := time.Now()
	for _, t := range s.tasks {
		s.nextRun[t.Name] = computeNextRun(t, now)
	}

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			for _, t := range s.tasks {
				next, ok := s.nextRun[t.Name]
				if !ok || now.Before(next) {
					continue
				}
				task := t
				go func() {
					if err := task.Fn(ctx); err != nil {
						log.Printf("scheduler: %s error: %v", task.Name, err)
					}
				}()
				s.nextRun[task.Name] = computeNextRun(task, now)
			}
		}
	}
}

// computeNextRun calculates when a task should next run after the given time.
func computeNextRun(t Task, after time.Time) time.Time {
	if t.RunAt != "" {
		return nextRunAt(t.RunAt, after)
	}
	return after.Add(t.Interval)
}

// nextRunAt computes the next occurrence of "HH:MM" after the given time.
func nextRunAt(runAt string, after time.Time) time.Time {
	var hour, min int
	if _, err := parseHHMM(runAt, &hour, &min); err != nil {
		return after.Add(24 * time.Hour)
	}

	loc := after.Location()
	candidate := time.Date(after.Year(), after.Month(), after.Day(), hour, min, 0, 0, loc)
	if !candidate.After(after) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}

// parseHHMM parses "HH:MM" into hour and minute.
func parseHHMM(s string, hour, min *int) (bool, error) {
	if len(s) != 5 || s[2] != ':' {
		return false, nil
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[3]-'0')*10 + int(s[4]-'0')
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return false, nil
	}
	*hour = h
	*min = m
	return true, nil
}
