package progress

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/schellhas/idletime/backend/internal/activity"
	"github.com/schellhas/idletime/backend/internal/category"
	"github.com/schellhas/idletime/backend/internal/timeentry"
	"github.com/schellhas/idletime/backend/internal/validation"
)

type fakeSources struct {
	days    []timeentry.DayMinutes
	gotZone string
	gotFrom time.Time
	gotIDs  []int64
	called  bool
}

func (f *fakeSources) categories() []category.Category {
	return []category.Category{{ID: 1, Name: "Sports", Multiplier: 1}}
}

type catSource struct{ f *fakeSources }

func (c catSource) List(context.Context, int64) ([]category.Category, error) {
	return c.f.categories(), nil
}

type actSource struct{}

func (actSource) List(context.Context, int64) ([]activity.Activity, error) {
	return []activity.Activity{{ID: 10, CategoryID: 1, Name: "Swimming", Multiplier: 1, TrackedMinutes: 90}}, nil
}

func (f *fakeSources) MinutesPerDay(_ context.Context, _ int64, zone string, since time.Time, activityIDs []int64) ([]timeentry.DayMinutes, error) {
	f.gotZone, f.gotFrom, f.gotIDs, f.called = zone, since, activityIDs, true
	return f.days, nil
}

func newService(f *fakeSources, now time.Time) *Service {
	s := NewService(catSource{f}, actSource{}, f)
	s.now = func() time.Time { return now }
	return s
}

func TestStatsFillsMissingDays(t *testing.T) {
	// 2026-06-10 12:00 UTC is already 14:00 on 2026-06-10 in Berlin.
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	f := &fakeSources{days: []timeentry.DayMinutes{{Date: "2026-06-08", Minutes: 30}, {Date: "2026-06-10", Minutes: 60}}}

	stats, err := newService(f, now).Stats(context.Background(), 1, 4, "Europe/Berlin", Scope{})
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	want := []timeentry.DayMinutes{{Date: "2026-06-07"}, {Date: "2026-06-08", Minutes: 30}, {Date: "2026-06-09"}, {Date: "2026-06-10", Minutes: 60}}
	if len(stats.Daily) != len(want) {
		t.Fatalf("daily = %+v, want %+v", stats.Daily, want)
	}
	for i := range want {
		if stats.Daily[i] != want[i] {
			t.Errorf("daily[%d] = %+v, want %+v", i, stats.Daily[i], want[i])
		}
	}
	if stats.PeriodMinutes != 90 || stats.TotalMinutes != 90 || stats.Days != 4 {
		t.Errorf("period = %d, total = %d, days = %d", stats.PeriodMinutes, stats.TotalMinutes, stats.Days)
	}
	if f.gotZone != "Europe/Berlin" || f.gotFrom.Format("2006-01-02 15:04") != "2026-06-07 00:00" {
		t.Errorf("repository asked for zone %q since %v", f.gotZone, f.gotFrom)
	}
}

func TestStatsDefaultsAndValidation(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	svc := newService(&fakeSources{}, now)
	ctx := context.Background()

	stats, err := svc.Stats(ctx, 1, 0, "", Scope{})
	if err != nil || stats.Days != defaultStatsDays || len(stats.Daily) != defaultStatsDays {
		t.Errorf("defaults: days = %d, daily = %d, err = %v", stats.Days, len(stats.Daily), err)
	}

	for name, args := range map[string]struct {
		days int
		zone string
	}{
		"too many days": {days: maxStatsDays + 1},
		"negative days": {days: -1},
		"unknown zone":  {days: 7, zone: "Mars/Olympus"},
	} {
		if _, err := svc.Stats(ctx, 1, args.days, args.zone, Scope{}); !errors.Is(err, validation.ErrInvalid) {
			t.Errorf("%s: error = %v, want validation error", name, err)
		}
	}
}

func TestRecommendUsesTheUsersData(t *testing.T) {
	rec, ok, err := newService(&fakeSources{}, time.Now()).Recommend(context.Background(), 1, Options{})
	if err != nil || !ok || rec.Activity.Name != "Swimming" || len(rec.Path) != 1 || rec.Path[0] != "Sports" {
		t.Errorf("Recommend() = %+v, %v, %v", rec, ok, err)
	}
}

func TestStatsScope(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()
	category, activity, unknown := int64(1), int64(10), int64(999)

	t.Run("without a scope all activities are summed", func(t *testing.T) {
		f := &fakeSources{}
		if _, err := newService(f, now).Stats(ctx, 1, 7, "", Scope{}); err != nil {
			t.Fatal(err)
		}
		if !f.called || f.gotIDs != nil {
			t.Errorf("called = %v, ids = %v; want a query without activity filter", f.called, f.gotIDs)
		}
	})

	t.Run("a category scope sums its activities", func(t *testing.T) {
		f := &fakeSources{}
		if _, err := newService(f, now).Stats(ctx, 1, 7, "", Scope{CategoryID: &category}); err != nil {
			t.Fatal(err)
		}
		if len(f.gotIDs) != 1 || f.gotIDs[0] != 10 {
			t.Errorf("ids = %v, want [10]", f.gotIDs)
		}
	})

	t.Run("an activity scope sums that activity", func(t *testing.T) {
		f := &fakeSources{}
		if _, err := newService(f, now).Stats(ctx, 1, 7, "", Scope{ActivityID: &activity}); err != nil {
			t.Fatal(err)
		}
		if len(f.gotIDs) != 1 || f.gotIDs[0] != 10 {
			t.Errorf("ids = %v, want [10]", f.gotIDs)
		}
	})

	t.Run("unknown scopes are reported", func(t *testing.T) {
		svc := newService(&fakeSources{}, now)
		if _, err := svc.Stats(ctx, 1, 7, "", Scope{CategoryID: &unknown}); !errors.Is(err, ErrCategoryNotFound) {
			t.Errorf("category: error = %v, want ErrCategoryNotFound", err)
		}
		if _, err := svc.Stats(ctx, 1, 7, "", Scope{ActivityID: &unknown}); !errors.Is(err, ErrActivityNotFound) {
			t.Errorf("activity: error = %v, want ErrActivityNotFound", err)
		}
	})

	t.Run("both a category and an activity is invalid", func(t *testing.T) {
		_, err := newService(&fakeSources{}, now).Stats(ctx, 1, 7, "", Scope{CategoryID: &category, ActivityID: &activity})
		if !errors.Is(err, validation.ErrInvalid) {
			t.Errorf("error = %v, want validation error", err)
		}
	})
}
