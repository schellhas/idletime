package timer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/schellhas/idletime/backend/internal/timeentry"
)

// Repository is the persistence port required by the Service.
type Repository interface {
	// Start records a running timer, or returns ErrAlreadyRunning / ErrActivityNotFound.
	Start(ctx context.Context, userID, activityID int64, startedAt time.Time) (Timer, error)
	// Get returns the running timer, or ErrNotRunning.
	Get(ctx context.Context, userID int64) (Timer, error)
	// Cancel discards the running timer without logging time, or returns ErrNotRunning.
	Cancel(ctx context.Context, userID int64) error
	// Finish atomically removes the running timer and stores a time entry whose length is
	// minutesFor(startedAt). It returns ErrNotRunning if there is no timer.
	Finish(ctx context.Context, userID int64, minutesFor func(startedAt time.Time) int) (timeentry.Entry, error)
}

// Service contains the stopwatch use cases.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService creates a Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Start begins timing an activity. Only one timer can run at a time.
func (s *Service) Start(ctx context.Context, userID, activityID int64) (Timer, error) {
	t, err := s.repo.Start(ctx, userID, activityID, s.now())
	if err != nil {
		return Timer{}, fmt.Errorf("start timer: %w", err)
	}
	return t, nil
}

// Current returns the running timer. ok is false if none is running.
func (s *Service) Current(ctx context.Context, userID int64) (t Timer, ok bool, err error) {
	t, err = s.repo.Get(ctx, userID)
	if errors.Is(err, ErrNotRunning) {
		return Timer{}, false, nil
	}
	if err != nil {
		return Timer{}, false, fmt.Errorf("get timer: %w", err)
	}
	return t, true, nil
}

// Stop ends the timer and logs the elapsed time as a time entry.
func (s *Service) Stop(ctx context.Context, userID int64) (timeentry.Entry, error) {
	now := s.now()
	entry, err := s.repo.Finish(ctx, userID, func(startedAt time.Time) int {
		return RoundedMinutes(now.Sub(startedAt))
	})
	if err != nil {
		return timeentry.Entry{}, fmt.Errorf("stop timer: %w", err)
	}
	return entry, nil
}

// Cancel discards the running timer without logging anything.
func (s *Service) Cancel(ctx context.Context, userID int64) error {
	if err := s.repo.Cancel(ctx, userID); err != nil {
		return fmt.Errorf("cancel timer: %w", err)
	}
	return nil
}

// RoundedMinutes converts a duration to whole minutes: rounded to the nearest minute,
// but at least one minute (a started timer always counts) and at most one day.
func RoundedMinutes(d time.Duration) int {
	minutes := int(math.Round(d.Minutes()))
	return min(max(minutes, 1), timeentry.MaxMinutes)
}
