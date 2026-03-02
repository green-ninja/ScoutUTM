package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestDevMode_TasksDoNotRun(t *testing.T) {
	s := New(true)
	var ran int32
	s.Register(Task{
		Name:     "test",
		Interval: 1 * time.Millisecond,
		Fn: func(ctx context.Context) error {
			atomic.AddInt32(&ran, 1)
			return nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	s.Start(ctx)

	if atomic.LoadInt32(&ran) != 0 {
		t.Error("dev-mode scheduler should not run tasks")
	}
}

func TestComputeNextRun_Interval(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	task := Task{Name: "test", Interval: 1 * time.Hour}

	next := computeNextRun(task, now)
	want := now.Add(1 * time.Hour)
	if !next.Equal(want) {
		t.Errorf("computeNextRun interval = %v, want %v", next, want)
	}
}

func TestComputeNextRun_RunAt_FutureToday(t *testing.T) {
	now := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)
	task := Task{Name: "test", RunAt: "03:00"}

	next := computeNextRun(task, now)
	want := time.Date(2025, 1, 15, 3, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("computeNextRun RunAt future = %v, want %v", next, want)
	}
}

func TestComputeNextRun_RunAt_Tomorrow(t *testing.T) {
	now := time.Date(2025, 1, 15, 4, 0, 0, 0, time.UTC)
	task := Task{Name: "test", RunAt: "03:00"}

	next := computeNextRun(task, now)
	want := time.Date(2025, 1, 16, 3, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("computeNextRun RunAt tomorrow = %v, want %v", next, want)
	}
}

func TestComputeNextRun_RunAt_Invalid(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	task := Task{Name: "test", RunAt: "bad"}

	next := computeNextRun(task, now)
	want := now.Add(24 * time.Hour)
	if !next.Equal(want) {
		t.Errorf("computeNextRun invalid RunAt = %v, want %v (24h fallback)", next, want)
	}
}

func TestParseHHMM(t *testing.T) {
	var h, m int

	ok := parseHHMM("03:00", &h, &m)
	if !ok || h != 3 || m != 0 {
		t.Errorf("parseHHMM(03:00) = %v, %d:%d", ok, h, m)
	}

	ok = parseHHMM("23:59", &h, &m)
	if !ok || h != 23 || m != 59 {
		t.Errorf("parseHHMM(23:59) = %v, %d:%d", ok, h, m)
	}

	ok = parseHHMM("bad", &h, &m)
	if ok {
		t.Error("parseHHMM(bad) should return false")
	}
}
