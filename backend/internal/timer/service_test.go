package timer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/schellhas/idletime/backend/internal/timeentry"
)

func TestRoundedMinutes(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want int
	}{
		{0, 1},                // a started timer always counts
		{20 * time.Second, 1}, // short timers count as one minute
		{89 * time.Second, 1}, // rounds to nearest: 1.48 -> 1
		{90 * time.Second, 2}, // 1.5 -> 2
		{47*time.Minute + 10*time.Second, 47},
		{48 * time.Hour, timeentry.MaxMinutes}, // capped at one day
		{-time.Minute, 1},                      // clock skew never yields zero or negative
	}
	for _, tt := range tests {
		if got := RoundedMinutes(tt.d); got != tt.want {
			t.Errorf("RoundedMinutes(%v) = %d, want %d", tt.d, got, tt.want)
		}
	}
}

type fakeRepository struct {
	timer   *Timer
	logged  *timeentry.Entry
	startAt time.Time
}

func (f *fakeRepository) Start(_ context.Context, _, activityID int64, at time.Time) (Timer, error) {
	if f.timer != nil {
		return Timer{}, ErrAlreadyRunning
	}
	f.timer = &Timer{ActivityID: activityID, StartedAt: at}
	return *f.timer, nil
}

func (f *fakeRepository) Get(context.Context, int64) (Timer, error) {
	if f.timer == nil {
		return Timer{}, ErrNotRunning
	}
	return *f.timer, nil
}

func (f *fakeRepository) Cancel(context.Context, int64) error {
	if f.timer == nil {
		return ErrNotRunning
	}
	f.timer = nil
	return nil
}

func (f *fakeRepository) Finish(_ context.Context, userID int64, minutesFor func(time.Time) int) (timeentry.Entry, error) {
	if f.timer == nil {
		return timeentry.Entry{}, ErrNotRunning
	}
	e := timeentry.Entry{UserID: userID, ActivityID: f.timer.ActivityID, Minutes: minutesFor(f.timer.StartedAt), LoggedAt: f.timer.StartedAt}
	f.timer, f.logged = nil, &e
	return e, nil
}

func TestServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	now := start
	svc := NewService(&fakeRepository{})
	svc.now = func() time.Time { return now }

	if _, running, _ := svc.Current(ctx, 1); running {
		t.Fatal("no timer should be running initially")
	}
	if _, err := svc.Start(ctx, 1, 7); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if _, err := svc.Start(ctx, 1, 8); !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("second Start() error = %v, want ErrAlreadyRunning", err)
	}
	if current, running, _ := svc.Current(ctx, 1); !running || current.ActivityID != 7 {
		t.Errorf("Current() = %+v, running=%v", current, running)
	}

	now = start.Add(32*time.Minute + 20*time.Second)
	entry, err := svc.Stop(ctx, 1)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if entry.Minutes != 32 || entry.ActivityID != 7 || !entry.LoggedAt.Equal(start) {
		t.Errorf("Stop() logged %+v, want 32 minutes for activity 7 starting at %v", entry, start)
	}
	if _, err := svc.Stop(ctx, 1); !errors.Is(err, ErrNotRunning) {
		t.Errorf("second Stop() error = %v, want ErrNotRunning", err)
	}
}

func TestCancelLogsNothing(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepository{}
	svc := NewService(repo)

	_, _ = svc.Start(ctx, 1, 7)
	if err := svc.Cancel(ctx, 1); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if repo.logged != nil {
		t.Error("cancelling must not log time")
	}
	if err := svc.Cancel(ctx, 1); !errors.Is(err, ErrNotRunning) {
		t.Errorf("second Cancel() error = %v, want ErrNotRunning", err)
	}
}
