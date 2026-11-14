package category

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxNameLength     = 100
	defaultMultiplier = 1.0
)

// Repository is the persistence port required by the Service.
type Repository interface {
	List(ctx context.Context) ([]Category, error)
	Create(ctx context.Context, c Category) (Category, error)
}

// CreateInput carries the user-supplied fields of a new category.
type CreateInput struct {
	ParentID   *int64
	Name       string
	Multiplier *float64
}

// Service contains the business rules for categories.
type Service struct {
	repo Repository
}

// NewService creates a Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns all categories.
func (s *Service) List(ctx context.Context) ([]Category, error) {
	categories, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return categories, nil
}

// Create validates the input and stores a new category.
func (s *Service) Create(ctx context.Context, in CreateInput) (Category, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > maxNameLength {
		return Category{}, fmt.Errorf("%w: name must be 1-%d characters", ErrInvalidInput, maxNameLength)
	}

	multiplier := defaultMultiplier
	if in.Multiplier != nil {
		multiplier = *in.Multiplier
	}
	if multiplier <= 0 {
		return Category{}, fmt.Errorf("%w: multiplier must be greater than 0", ErrInvalidInput)
	}

	created, err := s.repo.Create(ctx, Category{ParentID: in.ParentID, Name: name, Multiplier: multiplier})
	if err != nil {
		return Category{}, fmt.Errorf("create category: %w", err)
	}
	return created, nil
}
