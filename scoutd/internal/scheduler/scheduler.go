package scheduler

import (
	"context"
	"time"
)

// Task defines a periodic job.
type Task struct {
	Name     string
	Interval time.Duration
	Fn       func(ctx context.Context) error
}

// Scheduler runs periodic tasks.
type Scheduler struct {
	tasks []Task
}

// New creates a scheduler.
func New() *Scheduler {
	return &Scheduler{}
}

// Register adds a task to the scheduler.
func (s *Scheduler) Register(t Task) {
	s.tasks = append(s.tasks, t)
}

// Start is a stub that returns when the context is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	<-ctx.Done()
}
