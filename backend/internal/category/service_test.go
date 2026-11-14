package category

import (
	"context"
	"errors"
	"testing"

	"github.com/schellhas/idletime/backend/internal/optional"
	"github.com/schellhas/idletime/backend/internal/validation"
)

type fakeRepository struct {
	rows []Category
	err  error
}

func (f *fakeRepository) List(_ context.Context, userID int64) ([]Category, error) {
	var own []Category
	for _, c := range f.rows {
		if c.UserID == userID {
			own = append(own, c)
		}
	}
	return own, f.err
}

func (f *fakeRepository) Create(_ context.Context, c Category) (Category, error) {
	if f.err != nil {
		return Category{}, f.err
	}
	c.ID = int64(len(f.rows) + 1)
	f.rows = append(f.rows, c)
	return c, nil
}

func (f *fakeRepository) Update(_ context.Context, c Category) (Category, error) {
	for i := range f.rows {
		if f.rows[i].ID == c.ID && f.rows[i].UserID == c.UserID {
			f.rows[i] = c
			return c, nil
		}
	}
	return Category{}, ErrNotFound
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
	two := 2.0
	zero := 0.0

	tests := []struct {
		name           string
		in             CreateInput
		repoErr        error
		wantErr        error
		wantName       string
		wantMultiplier float64
	}{
		{name: "trims name and defaults multiplier", in: CreateInput{Name: "  Sports "}, wantName: "Sports", wantMultiplier: 1.0},
		{name: "keeps explicit multiplier", in: CreateInput{Name: "Sports", Multiplier: &two}, wantName: "Sports", wantMultiplier: 2.0},
		{name: "rejects blank name", in: CreateInput{Name: "   "}, wantErr: validation.ErrInvalid},
		{name: "rejects overlong name", in: CreateInput{Name: string(make([]rune, validation.MaxNameLength+1))}, wantErr: validation.ErrInvalid},
		{name: "rejects non-positive multiplier", in: CreateInput{Name: "Sports", Multiplier: &zero}, wantErr: validation.ErrInvalid},
		{name: "propagates repository errors", in: CreateInput{Name: "Sports"}, repoErr: ErrDuplicateName, wantErr: ErrDuplicateName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&fakeRepository{err: tt.repoErr})

			got, err := svc.Create(context.Background(), 1, tt.in)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != tt.wantName || got.Multiplier != tt.wantMultiplier {
				t.Errorf("got %q/%v, want %q/%v", got.Name, got.Multiplier, tt.wantName, tt.wantMultiplier)
			}
		})
	}
}

func TestServiceOwnsCategoriesPerUser(t *testing.T) {
	ctx := context.Background()
	svc := NewService(&fakeRepository{})

	created, err := svc.Create(ctx, 1, CreateInput{Name: "Private"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	own, _ := svc.List(ctx, 1)
	other, _ := svc.List(ctx, 2)
	if len(own) != 1 || len(other) != 0 {
		t.Errorf("user 1 sees %d, user 2 sees %d categories; want 1 and 0", len(own), len(other))
	}
	if _, err := svc.Update(ctx, 2, created.ID, UpdateInput{Name: ptr("Stolen")}); !errors.Is(err, ErrNotFound) {
		t.Errorf("another user's update: error = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, 2, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("another user's delete: error = %v, want ErrNotFound", err)
	}
}

func TestServiceUpdate(t *testing.T) {
	ctx := context.Background()

	// Tree: 1 (top) -> 2 -> 3, and 4 (top)
	newTree := func() *Service {
		return NewService(&fakeRepository{rows: []Category{
			{ID: 1, UserID: 1, Name: "A", Multiplier: 1},
			{ID: 2, UserID: 1, Name: "B", Multiplier: 1, ParentID: ptr(int64(1))},
			{ID: 3, UserID: 1, Name: "C", Multiplier: 1, ParentID: ptr(int64(2))},
			{ID: 4, UserID: 1, Name: "D", Multiplier: 1},
		}})
	}

	t.Run("renames and reweights only the given fields", func(t *testing.T) {
		got, err := newTree().Update(ctx, 1, 2, UpdateInput{Name: ptr(" Renamed "), Multiplier: ptr(3.0)})
		if err != nil || got.Name != "Renamed" || got.Multiplier != 3 || got.ParentID == nil || *got.ParentID != 1 {
			t.Fatalf("Update() = %+v, %v", got, err)
		}
	})

	t.Run("moves to another parent and to the top level", func(t *testing.T) {
		moved, err := newTree().Update(ctx, 1, 3, UpdateInput{ParentID: optional.Nullable[int64]{Set: true, Value: ptr(int64(4))}})
		if err != nil || moved.ParentID == nil || *moved.ParentID != 4 {
			t.Fatalf("move below 4: %+v, %v", moved, err)
		}
		top, err := newTree().Update(ctx, 1, 3, UpdateInput{ParentID: optional.Nullable[int64]{Set: true}})
		if err != nil || top.ParentID != nil {
			t.Fatalf("move to top level: %+v, %v", top, err)
		}
	})

	t.Run("refuses cycles", func(t *testing.T) {
		for name, parent := range map[string]int64{"itself": 1, "child": 2, "grandchild": 3} {
			_, err := newTree().Update(ctx, 1, 1, UpdateInput{ParentID: optional.Nullable[int64]{Set: true, Value: ptr(parent)}})
			if !errors.Is(err, validation.ErrInvalid) {
				t.Errorf("moving into %s: error = %v, want validation error", name, err)
			}
		}
	})

	t.Run("reports unknown categories", func(t *testing.T) {
		if _, err := newTree().Update(ctx, 1, 99, UpdateInput{Name: ptr("x")}); !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
}
