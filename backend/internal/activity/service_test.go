package activity

import (
	"context"
	"errors"
	"testing"

	"github.com/schellhas/idletime/backend/internal/validation"
)

type fakeRepository struct{ rows []Activity }

func (f *fakeRepository) List(_ context.Context, userID int64) ([]Activity, error) {
	var own []Activity
	for _, a := range f.rows {
		if a.UserID == userID {
			own = append(own, a)
		}
	}
	return own, nil
}

func (f *fakeRepository) Get(_ context.Context, userID, id int64) (Activity, error) {
	for _, a := range f.rows {
		if a.ID == id && a.UserID == userID {
			return a, nil
		}
	}
	return Activity{}, ErrNotFound
}

func (f *fakeRepository) Create(_ context.Context, a Activity) (Activity, error) {
	a.ID = int64(len(f.rows) + 1)
	f.rows = append(f.rows, a)
	return a, nil
}

func (f *fakeRepository) Update(_ context.Context, a Activity) (Activity, error) {
	for i := range f.rows {
		if f.rows[i].ID == a.ID && f.rows[i].UserID == a.UserID {
			f.rows[i] = a
			return a, nil
		}
	}
	return Activity{}, ErrNotFound
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

func ptr[T any](v T) *T { return &v }

func TestServiceCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("applies defaults and trims the name", func(t *testing.T) {
		got, err := NewService(&fakeRepository{}).Create(ctx, 1, CreateInput{CategoryID: 5, Name: "  Swimming "})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if got.Name != "Swimming" || got.Multiplier != 1 || got.MinimumMinutes != 0 || got.CategoryID != 5 {
			t.Errorf("got %+v", got)
		}
	})

	invalid := map[string]CreateInput{
		"blank name":         {CategoryID: 1, Name: " "},
		"zero multiplier":    {CategoryID: 1, Name: "x", Multiplier: ptr(0.0)},
		"negative minimum":   {CategoryID: 1, Name: "x", MinimumMinutes: ptr(-1)},
		"minimum over a day": {CategoryID: 1, Name: "x", MinimumMinutes: ptr(24*60 + 1)},
		"huge multiplier":    {CategoryID: 1, Name: "x", Multiplier: ptr(validation.MaxMultiplier + 1)},
	}
	for name, in := range invalid {
		t.Run("rejects "+name, func(t *testing.T) {
			if _, err := NewService(&fakeRepository{}).Create(ctx, 1, in); !errors.Is(err, validation.ErrInvalid) {
				t.Errorf("error = %v, want validation error", err)
			}
		})
	}
}

func TestServiceUpdate(t *testing.T) {
	ctx := context.Background()
	newService := func() *Service {
		return NewService(&fakeRepository{rows: []Activity{{ID: 1, UserID: 1, CategoryID: 5, Name: "Swim", Multiplier: 2, MinimumMinutes: 30}}})
	}

	t.Run("changes only the given fields", func(t *testing.T) {
		got, err := newService().Update(ctx, 1, 1, UpdateInput{MinimumMinutes: ptr(20)})
		if err != nil || got.MinimumMinutes != 20 || got.Name != "Swim" || got.Multiplier != 2 {
			t.Fatalf("got %+v, %v", got, err)
		}
	})

	t.Run("can move to another category", func(t *testing.T) {
		got, err := newService().Update(ctx, 1, 1, UpdateInput{CategoryID: ptr(int64(9))})
		if err != nil || got.CategoryID != 9 {
			t.Fatalf("got %+v, %v", got, err)
		}
	})

	t.Run("validates the result", func(t *testing.T) {
		if _, err := newService().Update(ctx, 1, 1, UpdateInput{Multiplier: ptr(-1.0)}); !errors.Is(err, validation.ErrInvalid) {
			t.Errorf("error = %v, want validation error", err)
		}
	})

	t.Run("is scoped to the owner", func(t *testing.T) {
		if _, err := newService().Update(ctx, 2, 1, UpdateInput{Name: ptr("Stolen")}); !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
		if err := newService().Delete(ctx, 2, 1); !errors.Is(err, ErrNotFound) {
			t.Errorf("delete: error = %v, want ErrNotFound", err)
		}
	})
}
