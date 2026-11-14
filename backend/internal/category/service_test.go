package category

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct {
	created []Category
	err     error
}

func (f *fakeRepository) List(context.Context) ([]Category, error) { return f.created, f.err }

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
		{name: "rejects blank name", in: CreateInput{Name: "   "}, wantErr: ErrInvalidInput},
		{name: "rejects overlong name", in: CreateInput{Name: string(make([]rune, maxNameLength+1))}, wantErr: ErrInvalidInput},
		{name: "rejects non-positive multiplier", in: CreateInput{Name: "Sports", Multiplier: &zero}, wantErr: ErrInvalidInput},
		{name: "propagates repository errors", in: CreateInput{Name: "Sports"}, repoErr: ErrDuplicateName, wantErr: ErrDuplicateName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&fakeRepository{err: tt.repoErr})

			got, err := svc.Create(context.Background(), tt.in)

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
