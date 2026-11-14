package integration_test

import (
	"errors"
	"testing"
	"time"

	"github.com/schellhas/idletime/backend/internal/activity"
	"github.com/schellhas/idletime/backend/internal/category"
	"github.com/schellhas/idletime/backend/internal/dbtest"
	"github.com/schellhas/idletime/backend/internal/timeentry"
	"github.com/schellhas/idletime/backend/internal/timer"
)

type trackingFixture struct {
	db         *dbtest.DB
	alice, bob int64
	swimming   activity.Activity
	reading    activity.Activity
	bobs       activity.Activity
	entries    *timeentry.PostgresRepository
	activities *activity.PostgresRepository
}

func newTrackingFixture(t *testing.T) trackingFixture {
	t.Helper()
	db := dbtest.New(t)
	alice := dbtest.CreateUser(t, db.Pool, "alice")
	bob := dbtest.CreateUser(t, db.Pool, "bob")

	categories := category.NewPostgresRepository(db.Pool)
	activities := activity.NewPostgresRepository(db.Pool)
	aliceCat := mustCategory(t, categories, alice.ID, "Life", nil)
	bobCat := mustCategory(t, categories, bob.ID, "Life", nil)

	return trackingFixture{
		db: db, alice: alice.ID, bob: bob.ID,
		swimming:   mustActivity(t, activities, alice.ID, aliceCat.ID, "Swimming"),
		reading:    mustActivity(t, activities, alice.ID, aliceCat.ID, "Reading"),
		bobs:       mustActivity(t, activities, bob.ID, bobCat.ID, "Bobs activity"),
		entries:    timeentry.NewPostgresRepository(db.Pool),
		activities: activities,
	}
}

func TestTimeEntryRepository(t *testing.T) {
	f := newTrackingFixture(t)
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	t.Run("an entry cannot reference another user's activity", func(t *testing.T) {
		_, err := f.entries.Create(ctx, timeentry.Entry{UserID: f.alice, ActivityID: f.bobs.ID, Minutes: 10, LoggedAt: base})
		if !errors.Is(err, timeentry.ErrActivityNotFound) {
			t.Errorf("error = %v, want ErrActivityNotFound", err)
		}
	})

	first := mustEntry(t, f.entries, f.alice, f.swimming.ID, 30, base)
	second := mustEntry(t, f.entries, f.alice, f.swimming.ID, 45, base.Add(24*time.Hour))
	third := mustEntry(t, f.entries, f.alice, f.reading.ID, 20, base.Add(48*time.Hour))
	mustEntry(t, f.entries, f.bob, f.bobs.ID, 99, base)

	t.Run("notes round-trip and empty notes stay empty", func(t *testing.T) {
		withNote, err := f.entries.Create(ctx, timeentry.Entry{UserID: f.alice, ActivityID: f.reading.ID, Minutes: 5, Note: "chapter 3", LoggedAt: base})
		if err != nil || withNote.Note != "chapter 3" {
			t.Errorf("note = %q, %v", withNote.Note, err)
		}
		if first.Note != "" {
			t.Errorf("note = %q, want empty", first.Note)
		}
	})

	t.Run("lists newest first, can filter by activity and limit", func(t *testing.T) {
		all, err := f.entries.List(ctx, f.alice, timeentry.ListFilter{Limit: -1})
		if err != nil || len(all) != 4 || all[0].ID != third.ID {
			t.Fatalf("List(all) = %+v, %v", all, err)
		}
		onlySwimming, _ := f.entries.List(ctx, f.alice, timeentry.ListFilter{ActivityID: &f.swimming.ID, Limit: -1})
		if len(onlySwimming) != 2 || onlySwimming[0].ID != second.ID || onlySwimming[1].ID != first.ID {
			t.Errorf("List(swimming) = %+v", onlySwimming)
		}
		limited, _ := f.entries.List(ctx, f.alice, timeentry.ListFilter{Limit: 1})
		if len(limited) != 1 {
			t.Errorf("List(limit 1) returned %d entries", len(limited))
		}
	})

	t.Run("activities report their tracked minutes", func(t *testing.T) {
		got, err := f.activities.Get(ctx, f.alice, f.swimming.ID)
		if err != nil || got.TrackedMinutes != 75 {
			t.Errorf("tracked = %d, %v; want 75", got.TrackedMinutes, err)
		}
	})

	t.Run("update and delete respect ownership", func(t *testing.T) {
		changed := first
		changed.Minutes, changed.Note = 40, "longer"
		if got, err := f.entries.Update(ctx, changed); err != nil || got.Minutes != 40 || got.Note != "longer" {
			t.Fatalf("Update() = %+v, %v", got, err)
		}
		foreign := changed
		foreign.UserID = f.bob
		if _, err := f.entries.Update(ctx, foreign); !errors.Is(err, timeentry.ErrNotFound) {
			t.Errorf("foreign update: error = %v, want ErrNotFound", err)
		}
		if err := f.entries.Delete(ctx, f.bob, first.ID); !errors.Is(err, timeentry.ErrNotFound) {
			t.Errorf("foreign delete: error = %v, want ErrNotFound", err)
		}
		if err := f.entries.Delete(ctx, f.alice, first.ID); err != nil {
			t.Errorf("Delete() error = %v", err)
		}
		if _, err := f.entries.Get(ctx, f.alice, first.ID); !errors.Is(err, timeentry.ErrNotFound) {
			t.Errorf("Get after delete: error = %v, want ErrNotFound", err)
		}
	})

	t.Run("deleting an activity removes its entries", func(t *testing.T) {
		if err := f.activities.Delete(ctx, f.alice, f.reading.ID); err != nil {
			t.Fatal(err)
		}
		if n := count(t, f.db.Pool, "SELECT count(*) FROM time_entries WHERE activity_id = $1", f.reading.ID); n != 0 {
			t.Errorf("%d entries survived", n)
		}
	})
}

func TestMinutesPerDay(t *testing.T) {
	f := newTrackingFixture(t)

	// 23:30 UTC on June 1st is already June 2nd (01:30) in Berlin, but still June 1st in UTC.
	lateEvening := time.Date(2026, 6, 1, 23, 30, 0, 0, time.UTC)
	nextMorning := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	mustEntry(t, f.entries, f.alice, f.swimming.ID, 30, lateEvening)
	mustEntry(t, f.entries, f.alice, f.reading.ID, 20, nextMorning)
	mustEntry(t, f.entries, f.bob, f.bobs.ID, 500, nextMorning)
	since := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)

	toMap := func(days []timeentry.DayMinutes) map[string]int {
		m := map[string]int{}
		for _, d := range days {
			m[d.Date] = d.Minutes
		}
		return m
	}

	t.Run("days are cut in the requested time zone", func(t *testing.T) {
		utc, err := f.entries.MinutesPerDay(ctx, f.alice, "UTC", since, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := toMap(utc); len(got) != 2 || got["2026-06-01"] != 30 || got["2026-06-02"] != 20 {
			t.Errorf("UTC = %v", got)
		}

		berlin, _ := f.entries.MinutesPerDay(ctx, f.alice, "Europe/Berlin", since, nil)
		if got := toMap(berlin); len(got) != 1 || got["2026-06-02"] != 50 {
			t.Errorf("Berlin = %v, want everything on 2026-06-02", got)
		}
	})

	t.Run("results are sorted by day and only count the user's own time", func(t *testing.T) {
		days, _ := f.entries.MinutesPerDay(ctx, f.alice, "UTC", since, nil)
		if len(days) != 2 || days[0].Date > days[1].Date {
			t.Errorf("days = %+v, want ascending order", days)
		}
		if toMap(days)["2026-06-02"] == 520 {
			t.Error("another user's minutes leaked into the sums")
		}
	})

	t.Run("an activity filter restricts the sum; an empty list matches nothing", func(t *testing.T) {
		onlyReading, _ := f.entries.MinutesPerDay(ctx, f.alice, "UTC", since, []int64{f.reading.ID})
		if got := toMap(onlyReading); len(got) != 1 || got["2026-06-02"] != 20 {
			t.Errorf("filtered = %v", got)
		}
		none, err := f.entries.MinutesPerDay(ctx, f.alice, "UTC", since, []int64{})
		if err != nil || len(none) != 0 {
			t.Errorf("empty filter = %+v, %v; want no days", none, err)
		}
	})

	t.Run("entries before the start date are left out", func(t *testing.T) {
		days, _ := f.entries.MinutesPerDay(ctx, f.alice, "UTC", nextMorning.Add(-time.Hour), nil)
		if got := toMap(days); len(got) != 1 || got["2026-06-02"] != 20 {
			t.Errorf("got %v", got)
		}
	})

	t.Run("an unknown time zone is an error, not a silent fallback", func(t *testing.T) {
		if _, err := f.entries.MinutesPerDay(ctx, f.alice, "Mars/Olympus", since, nil); err == nil {
			t.Error("expected an error")
		}
	})
}

func TestTimerRepository(t *testing.T) {
	f := newTrackingFixture(t)
	repo := timer.NewPostgresRepository(f.db.Pool)
	started := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	if _, err := repo.Get(ctx, f.alice); !errors.Is(err, timer.ErrNotRunning) {
		t.Fatalf("Get() before start: error = %v, want ErrNotRunning", err)
	}

	t.Run("only one timer runs per user", func(t *testing.T) {
		got, err := repo.Start(ctx, f.alice, f.swimming.ID, started)
		if err != nil || got.ActivityID != f.swimming.ID || !got.StartedAt.Equal(started) {
			t.Fatalf("Start() = %+v, %v", got, err)
		}
		if _, err := repo.Start(ctx, f.alice, f.reading.ID, started); !errors.Is(err, timer.ErrAlreadyRunning) {
			t.Errorf("second Start: error = %v, want ErrAlreadyRunning", err)
		}
		if running, err := repo.Get(ctx, f.alice); err != nil || running.ActivityID != f.swimming.ID {
			t.Errorf("Get() = %+v, %v", running, err)
		}
	})

	t.Run("timers cannot use another user's activity", func(t *testing.T) {
		if _, err := repo.Start(ctx, f.bob, f.swimming.ID, started); !errors.Is(err, timer.ErrActivityNotFound) {
			t.Errorf("error = %v, want ErrActivityNotFound", err)
		}
	})

	t.Run("finishing logs the entry and removes the timer in one step", func(t *testing.T) {
		var seenStart time.Time
		entry, err := repo.Finish(ctx, f.alice, func(startedAt time.Time) int {
			seenStart = startedAt
			return 42
		})
		if err != nil {
			t.Fatalf("Finish() error = %v", err)
		}
		if !seenStart.Equal(started) || entry.Minutes != 42 || entry.ActivityID != f.swimming.ID || !entry.LoggedAt.Equal(started) {
			t.Errorf("entry = %+v (start seen: %v)", entry, seenStart)
		}
		if _, err := repo.Get(ctx, f.alice); !errors.Is(err, timer.ErrNotRunning) {
			t.Error("the timer is still running")
		}
		if n := count(t, f.db.Pool, "SELECT count(*) FROM time_entries WHERE user_id = $1", f.alice); n != 1 {
			t.Errorf("%d entries logged, want exactly 1", n)
		}
		if _, err := repo.Finish(ctx, f.alice, func(time.Time) int { return 1 }); !errors.Is(err, timer.ErrNotRunning) {
			t.Errorf("second Finish: error = %v, want ErrNotRunning", err)
		}
	})

	t.Run("cancelling discards the timer without logging time", func(t *testing.T) {
		if _, err := repo.Start(ctx, f.alice, f.reading.ID, started); err != nil {
			t.Fatal(err)
		}
		if err := repo.Cancel(ctx, f.alice); err != nil {
			t.Fatalf("Cancel() error = %v", err)
		}
		if n := count(t, f.db.Pool, "SELECT count(*) FROM time_entries WHERE user_id = $1", f.alice); n != 1 {
			t.Errorf("%d entries, want the earlier one only", n)
		}
		if err := repo.Cancel(ctx, f.alice); !errors.Is(err, timer.ErrNotRunning) {
			t.Errorf("second Cancel: error = %v, want ErrNotRunning", err)
		}
	})

	t.Run("deleting the activity removes its running timer", func(t *testing.T) {
		if _, err := repo.Start(ctx, f.alice, f.reading.ID, started); err != nil {
			t.Fatal(err)
		}
		if err := f.activities.Delete(ctx, f.alice, f.reading.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Get(ctx, f.alice); !errors.Is(err, timer.ErrNotRunning) {
			t.Errorf("error = %v, want the timer to be gone", err)
		}
	})
}
