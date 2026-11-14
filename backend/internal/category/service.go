package category

import (
	"context"
	"fmt"

	"github.com/schellhas/idletime/backend/internal/optional"
	"github.com/schellhas/idletime/backend/internal/validation"
)

const defaultMultiplier = 1.0

// Repository is the persistence port required by the Service.
// Every operation is scoped to the owning user.
type Repository interface {
	List(ctx context.Context, userID int64) ([]Category, error)
	Create(ctx context.Context, c Category) (Category, error)
	Update(ctx context.Context, c Category) (Category, error)
	Delete(ctx context.Context, userID, id int64) error
}

// CreateInput carries the user-supplied fields of a new category.
type CreateInput struct {
	ParentID   *int64
	Name       string
	Multiplier *float64
}

// UpdateInput describes a partial update; nil / unset fields stay unchanged.
type UpdateInput struct {
	Name       *string
	Multiplier *float64
	// ParentID set to null moves the category to the top level.
	ParentID optional.Nullable[int64]
}

// Service contains the business rules for categories.
type Service struct {
	repo Repository
}

// NewService creates a Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns all categories of the user.
func (s *Service) List(ctx context.Context, userID int64) ([]Category, error) {
	categories, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return categories, nil
}

// Create validates the input and stores a new category for the user.
func (s *Service) Create(ctx context.Context, userID int64, in CreateInput) (Category, error) {
	name, err := validation.Name(in.Name)
	if err != nil {
		return Category{}, err
	}

	multiplier := defaultMultiplier
	if in.Multiplier != nil {
		multiplier = *in.Multiplier
	}
	if err := validation.Multiplier(multiplier); err != nil {
		return Category{}, err
	}

	created, err := s.repo.Create(ctx, Category{UserID: userID, ParentID: in.ParentID, Name: name, Multiplier: multiplier})
	if err != nil {
		return Category{}, fmt.Errorf("create category: %w", err)
	}
	return created, nil
}

// Update applies a partial update. A category cannot be moved below itself.
func (s *Service) Update(ctx context.Context, userID, id int64, in UpdateInput) (Category, error) {
	all, err := s.repo.List(ctx, userID)
	if err != nil {
		return Category{}, fmt.Errorf("list categories: %w", err)
	}
	current, ok := find(all, id)
	if !ok {
		return Category{}, ErrNotFound
	}

	if in.Name != nil {
		if current.Name, err = validation.Name(*in.Name); err != nil {
			return Category{}, err
		}
	}
	if in.Multiplier != nil {
		if err := validation.Multiplier(*in.Multiplier); err != nil {
			return Category{}, err
		}
		current.Multiplier = *in.Multiplier
	}
	if in.ParentID.Set {
		if parent := in.ParentID.Value; parent != nil && isSelfOrDescendant(all, id, *parent) {
			return Category{}, validation.Errorf("a category cannot be moved into itself or one of its subcategories")
		}
		current.ParentID = in.ParentID.Value
	}

	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return Category{}, fmt.Errorf("update category: %w", err)
	}
	return updated, nil
}

// Delete removes a category together with its subcategories and activities.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}

func find(categories []Category, id int64) (Category, bool) {
	for _, c := range categories {
		if c.ID == id {
			return c, true
		}
	}
	return Category{}, false
}

// isSelfOrDescendant reports whether candidate is ancestorID itself or lies in its subtree,
// by walking from candidate up to the top level.
func isSelfOrDescendant(categories []Category, ancestorID, candidate int64) bool {
	parents := make(map[int64]*int64, len(categories))
	for _, c := range categories {
		parents[c.ID] = c.ParentID
	}
	for current, steps := &candidate, 0; current != nil && steps <= len(categories); steps++ {
		if *current == ancestorID {
			return true
		}
		current = parents[*current]
	}
	return false
}
