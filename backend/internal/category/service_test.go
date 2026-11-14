package category

import (
	"context"
	"errors"
	"testing"

	"github.com/schellhas/idletime/backend/internal/validation"
)

type fakeRepository struct {
	created []Category
	err     error
}

func (f *fakeRepository) List(_ context.Context, userID int64) ([]Category, error) {
	var own []Category
	for _, c := range f.created {
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
	c.ID = int64(len(f.created) + 1)
	f.created = append(f.created, c)
	return c, nil
}

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
		{name: "rejects overlong name", in: CreateInput{Name: string(make([]rune, maxNameLength+1))}, wantErr: validation.ErrInvalid},
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

	if _, err := svc.Create(ctx, 1, CreateInput{Name: "Private"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	own, _ := svc.List(ctx, 1)
	other, _ := svc.List(ctx, 2)
	if len(own) != 1 || len(other) != 0 {
		t.Errorf("user 1 sees %d, user 2 sees %d categories; want 1 and 0", len(own), len(other))
	}
}
