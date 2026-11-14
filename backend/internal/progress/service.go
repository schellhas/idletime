package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/schellhas/idletime/backend/internal/activity"
	"github.com/schellhas/idletime/backend/internal/category"
	"github.com/schellhas/idletime/backend/internal/timeentry"
	"github.com/schellhas/idletime/backend/internal/validation"
)

const (
	defaultStatsDays = 30
	maxStatsDays     = 365
	dateLayout       = "2006-01-02"
)

// CategorySource lists a user's categories.
type CategorySource interface {
	List(ctx context.Context, userID int64) ([]category.Category, error)
}

// ActivitySource lists a user's activities with their tracked minutes.
type ActivitySource interface {
	List(ctx context.Context, userID int64) ([]activity.Activity, error)
}

// DailySource sums a user's logged time per calendar day.
type DailySource interface {
	MinutesPerDay(ctx context.Context, userID int64, timeZone string, since time.Time) ([]timeentry.DayMinutes, error)
}

// Service answers "what should I do next?" and "how am I doing?".
type Service struct {
	categories CategorySource
	activities ActivitySource
	daily      DailySource
	now        func() time.Time
}

// NewService creates a Service.
func NewService(categories CategorySource, activities ActivitySource, daily DailySource) *Service {
	return &Service{categories: categories, activities: activities, daily: daily, now: time.Now}
}

// Recommend returns the activity to do next. ok is false when nothing fits.
func (s *Service) Recommend(ctx context.Context, userID int64, o Options) (Recommendation, bool, error) {
	plan, err := s.plan(ctx, userID)
	if err != nil {
		return Recommendation{}, false, err
	}
	return plan.Recommend(o)
}

// Stats summarises the user's progress.
type Stats struct {
	// TotalMinutes is all time ever tracked.
	TotalMinutes int `json:"totalMinutes"`
	// PeriodMinutes is the time tracked within the last Days days.
	PeriodMinutes int `json:"periodMinutes"`
	Days          int `json:"days"`
	// Shares compares target and actual time distribution over all time.
	Shares []Share `json:"shares"`
	// Daily has one entry per day of the period, oldest first, including days without time.
	Daily []timeentry.DayMinutes `json:"daily"`
}

// Stats computes the user's statistics. days is the length of the daily chart (1-365,
// default 30) and timeZone an IANA name such as "Europe/Berlin" (default UTC).
func (s *Service) Stats(ctx context.Context, userID int64, days int, timeZone string) (Stats, error) {
	if days == 0 {
		days = defaultStatsDays
	}
	if days < 1 || days > maxStatsDays {
		return Stats{}, validation.Errorf("days must be between 1 and %d", maxStatsDays)
	}
	if timeZone == "" {
		timeZone = "UTC"
	}
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return Stats{}, validation.Errorf("unknown time zone %q", timeZone)
	}

	plan, err := s.plan(ctx, userID)
	if err != nil {
		return Stats{}, err
	}

	today := s.now().In(location)
	first := time.Date(today.Year(), today.Month(), today.Day()-(days-1), 0, 0, 0, 0, location)
	logged, err := s.daily.MinutesPerDay(ctx, userID, timeZone, first)
	if err != nil {
		return Stats{}, fmt.Errorf("daily minutes: %w", err)
	}

	daily, period := fillDays(first, days, logged)
	return Stats{
		TotalMinutes:  plan.TotalMinutes(),
		PeriodMinutes: period,
		Days:          days,
		Shares:        plan.Shares(),
		Daily:         daily,
	}, nil
}

// fillDays returns one entry per day starting at first, with zero for days without entries.
func fillDays(first time.Time, days int, logged []timeentry.DayMinutes) ([]timeentry.DayMinutes, int) {
	byDate := make(map[string]int, len(logged))
	for _, d := range logged {
		byDate[d.Date] = d.Minutes
	}

	result := make([]timeentry.DayMinutes, days)
	total := 0
	for i := range result {
		date := first.AddDate(0, 0, i).Format(dateLayout)
		result[i] = timeentry.DayMinutes{Date: date, Minutes: byDate[date]}
		total += byDate[date]
	}
	return result, total
}

func (s *Service) plan(ctx context.Context, userID int64) (*Plan, error) {
	categories, err := s.categories.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	activities, err := s.activities.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list activities: %w", err)
	}
	return NewPlan(categories, activities), nil
}
