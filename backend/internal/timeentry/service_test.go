package timeentry

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/schellhas/idletime/backend/internal/validation"
)

var fixedNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

type fakeRepository struct{ rows []Entry }

func (f *fakeRepository) Create(_ context.Context, e Entry) (Entry, error) {
	e.ID = int64(len(f.rows) + 1)
	f.rows = append(f.rows, e)
	return e, nil
}

func (f *fakeRepository) Get(_ context.Context, userID, id int64) (Entry, error) {
	for _, e := range f.rows {
		if e.ID == id && e.UserID == userID {
			return e, nil
		}
	}
	return Entry{}, ErrNotFound
}

func (f *fakeRepository) Update(_ context.Context, e Entry) (Entry, error) {
	for i := range f.rows {
		if f.rows[i].ID == e.ID && f.rows[i].UserID == e.UserID {
			f.rows[i] = e
			return e, nil
		}
	}
	return Entry{}, ErrNotFound
}

func (f *fakeRepository) Delete(_ context.Context, userID, id int64) error {
	for i := range f.rows {
		if f.rows[i].ID == id && f.rows[i].UserID == userID {
			f.rows = append(f.rows[:i], f.rows[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (f *fakeRepository) List(_ context.Context, _ int64, filter ListFilter) ([]Entry, error) {
	lastFilter = filter
	return f.rows, nil
}

func (f *fakeRepository) MinutesPerDay(context.Context, int64, string, time.Time, []int64) ([]DayMinutes, error) {
	return nil, nil
}

var lastFilter ListFilter // the filter the service passed to the repository

func newService(repo Repository) *Service {
	s := NewService(repo)
	s.now = func() time.Time { return fixedNow }
	return s
}

func ptr[T any](v T) *T { return &v }

func TestCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("defaults the time to now and trims the note", func(t *testing.T) {
		got, err := newService(&fakeRepository{}).Create(ctx, 1, CreateInput{ActivityID: 3, Minutes: 25, Note: "  felt good "})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !got.LoggedAt.Equal(fixedNow) || got.Note != "felt good" || got.Minutes != 25 {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("accepts entries in the past", func(t *testing.T) {
		past := fixedNow.Add(-48 * time.Hour)
		if _, err := newService(&fakeRepository{}).Create(ctx, 1, CreateInput{ActivityID: 3, Minutes: 10, LoggedAt: &past}); err != nil {
			t.Errorf("error = %v", err)
		}
	})

	invalid := map[string]CreateInput{
		"zero minutes":     {ActivityID: 1, Minutes: 0},
		"negative minutes": {ActivityID: 1, Minutes: -5},
		"more than a day":  {ActivityID: 1, Minutes: MaxMinutes + 1},
		"long note":        {ActivityID: 1, Minutes: 5, Note: strings.Repeat("x", maxNoteChars+1)},
		"future time":      {ActivityID: 1, Minutes: 5, LoggedAt: ptr(fixedNow.Add(time.Hour))},
	}
	for name, in := range invalid {
		t.Run("rejects "+name, func(t *testing.T) {
			if _, err := newService(&fakeRepository{}).Create(ctx, 1, in); !errors.Is(err, validation.ErrInvalid) {
				t.Errorf("error = %v, want validation error", err)
			}
		})
	}
}

func TestUpdateAndDeleteAreScopedToTheOwner(t *testing.T) {
	ctx := context.Background()
	svc := newService(&fakeRepository{rows: []Entry{{ID: 1, UserID: 1, ActivityID: 3, Minutes: 30, LoggedAt: fixedNow}}})

	if _, err := svc.Update(ctx, 2, 1, UpdateInput{Minutes: ptr(5)}); !errors.Is(err, ErrNotFound) {
		t.Errorf("foreign update: error = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, 2, 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("foreign delete: error = %v, want ErrNotFound", err)
	}

	got, err := svc.Update(ctx, 1, 1, UpdateInput{Minutes: ptr(45)})
	if err != nil || got.Minutes != 45 || got.ActivityID != 3 {
		t.Errorf("own update: %+v, %v", got, err)
	}
}

func TestListClampsTheLimit(t *testing.T) {
	ctx := context.Background()
	svc := newService(&fakeRepository{})

	for in, want := range map[int]int{0: defaultListLimit, -3: defaultListLimit, 10: 10, 100000: maxListLimit} {
		if _, err := svc.List(ctx, 1, ListFilter{Limit: in}); err != nil {
			t.Fatal(err)
		}
		if lastFilter.Limit != want {
			t.Errorf("limit %d was passed on as %d, want %d", in, lastFilter.Limit, want)
		}
	}
}
