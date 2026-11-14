package timeentry

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/schellhas/idletime/backend/internal/validation"
)

const (
	// MaxMinutes is the longest single entry: one day.
	MaxMinutes   = 24 * 60
	maxNoteChars = 500
	// futureTolerance absorbs clock differences between client and server.
	futureTolerance = 5 * time.Minute

	defaultListLimit = 50
	maxListLimit     = 500
)

// ListFilter narrows a listing.
type ListFilter struct {
	ActivityID *int64
	// Limit caps the number of entries, newest first. Zero means the default.
	Limit int
}

// Repository is the persistence port required by the Service.
// Every operation is scoped to the owning user.
type Repository interface {
	Create(ctx context.Context, e Entry) (Entry, error)
	Get(ctx context.Context, userID, id int64) (Entry, error)
	Update(ctx context.Context, e Entry) (Entry, error)
	Delete(ctx context.Context, userID, id int64) error
	List(ctx context.Context, userID int64, f ListFilter) ([]Entry, error)
	// MinutesPerDay sums minutes per day; a nil activityIDs means all activities.
	MinutesPerDay(ctx context.Context, userID int64, timeZone string, since time.Time, activityIDs []int64) ([]DayMinutes, error)
}

// CreateInput carries the user-supplied fields of a new entry.
type CreateInput struct {
	ActivityID int64
	Minutes    int
	Note       string
	LoggedAt   *time.Time // defaults to now
}

// UpdateInput describes a partial update; nil fields stay unchanged.
type UpdateInput struct {
	Minutes  *int
	Note     *string
	LoggedAt *time.Time
}

// Service contains the business rules for time entries.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService creates a Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Create validates the input and stores a new entry.
func (s *Service) Create(ctx context.Context, userID int64, in CreateInput) (Entry, error) {
	e := Entry{UserID: userID, ActivityID: in.ActivityID, Minutes: in.Minutes, Note: in.Note, LoggedAt: s.now()}
	if in.LoggedAt != nil {
		e.LoggedAt = *in.LoggedAt
	}
	if err := s.validate(&e); err != nil {
		return Entry{}, err
	}

	created, err := s.repo.Create(ctx, e)
	if err != nil {
		return Entry{}, fmt.Errorf("create time entry: %w", err)
	}
	return created, nil
}

// Update applies a partial update.
func (s *Service) Update(ctx context.Context, userID, id int64, in UpdateInput) (Entry, error) {
	e, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return Entry{}, fmt.Errorf("get time entry: %w", err)
	}
	if in.Minutes != nil {
		e.Minutes = *in.Minutes
	}
	if in.Note != nil {
		e.Note = *in.Note
	}
	if in.LoggedAt != nil {
		e.LoggedAt = *in.LoggedAt
	}
	if err := s.validate(&e); err != nil {
		return Entry{}, err
	}

	updated, err := s.repo.Update(ctx, e)
	if err != nil {
		return Entry{}, fmt.Errorf("update time entry: %w", err)
	}
	return updated, nil
}

// Delete removes an entry.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return fmt.Errorf("delete time entry: %w", err)
	}
	return nil
}

// List returns the user's entries, newest first.
func (s *Service) List(ctx context.Context, userID int64, f ListFilter) ([]Entry, error) {
	switch {
	case f.Limit <= 0:
		f.Limit = defaultListLimit
	case f.Limit > maxListLimit:
		f.Limit = maxListLimit
	}
	entries, err := s.repo.List(ctx, userID, f)
	if err != nil {
		return nil, fmt.Errorf("list time entries: %w", err)
	}
	return entries, nil
}

// ListAll returns every entry of the user, for the data export.
func (s *Service) ListAll(ctx context.Context, userID int64) ([]Entry, error) {
	entries, err := s.repo.List(ctx, userID, ListFilter{Limit: -1})
	if err != nil {
		return nil, fmt.Errorf("list time entries: %w", err)
	}
	return entries, nil
}

// MinutesPerDay sums the user's logged time per calendar day since the given instant.
// Days are cut in the given IANA time zone (e.g. "Europe/Berlin"). A nil activityIDs counts
// every activity; otherwise only the listed ones.
func (s *Service) MinutesPerDay(ctx context.Context, userID int64, timeZone string, since time.Time, activityIDs []int64) ([]DayMinutes, error) {
	days, err := s.repo.MinutesPerDay(ctx, userID, timeZone, since, activityIDs)
	if err != nil {
		return nil, fmt.Errorf("minutes per day: %w", err)
	}
	return days, nil
}

func (s *Service) validate(e *Entry) error {
	if e.Minutes < 1 || e.Minutes > MaxMinutes {
		return validation.Errorf("minutes must be between 1 and %d", MaxMinutes)
	}
	e.Note = strings.TrimSpace(e.Note)
	if utf8.RuneCountInString(e.Note) > maxNoteChars {
		return validation.Errorf("note must be at most %d characters", maxNoteChars)
	}
	if e.LoggedAt.After(s.now().Add(futureTolerance)) {
		return validation.Errorf("the time cannot be in the future")
	}
	return nil
}
