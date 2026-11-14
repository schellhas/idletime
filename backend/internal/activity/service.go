package activity

import (
	"context"
	"fmt"

	"github.com/schellhas/idletime/backend/internal/validation"
)

const (
	defaultMultiplier = 1.0
	maxMinimumMinutes = 24 * 60
)

// Repository is the persistence port required by the Service.
// Every operation is scoped to the owning user.
type Repository interface {
	List(ctx context.Context, userID int64) ([]Activity, error)
	Get(ctx context.Context, userID, id int64) (Activity, error)
	Create(ctx context.Context, a Activity) (Activity, error)
	Update(ctx context.Context, a Activity) (Activity, error)
	Delete(ctx context.Context, userID, id int64) error
}

// CreateInput carries the user-supplied fields of a new activity.
type CreateInput struct {
	CategoryID     int64
	Name           string
	Multiplier     *float64
	MinimumMinutes *int
}

// UpdateInput describes a partial update; nil fields stay unchanged.
type UpdateInput struct {
	CategoryID     *int64
	Name           *string
	Multiplier     *float64
	MinimumMinutes *int
}

// Service contains the business rules for activities.
type Service struct {
	repo Repository
}

// NewService creates a Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns all activities of the user including their tracked time.
func (s *Service) List(ctx context.Context, userID int64) ([]Activity, error) {
	activities, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list activities: %w", err)
	}
	return activities, nil
}

// Create validates the input and stores a new activity.
func (s *Service) Create(ctx context.Context, userID int64, in CreateInput) (Activity, error) {
	a := Activity{UserID: userID, CategoryID: in.CategoryID, Multiplier: defaultMultiplier}
	if in.Multiplier != nil {
		a.Multiplier = *in.Multiplier
	}
	if in.MinimumMinutes != nil {
		a.MinimumMinutes = *in.MinimumMinutes
	}
	var err error
	if a.Name, err = validation.Name(in.Name); err != nil {
		return Activity{}, err
	}
	if err := validate(a); err != nil {
		return Activity{}, err
	}

	created, err := s.repo.Create(ctx, a)
	if err != nil {
		return Activity{}, fmt.Errorf("create activity: %w", err)
	}
	return created, nil
}

// Update applies a partial update.
func (s *Service) Update(ctx context.Context, userID, id int64, in UpdateInput) (Activity, error) {
	a, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return Activity{}, fmt.Errorf("get activity: %w", err)
	}

	if in.Name != nil {
		if a.Name, err = validation.Name(*in.Name); err != nil {
			return Activity{}, err
		}
	}
	if in.CategoryID != nil {
		a.CategoryID = *in.CategoryID
	}
	if in.Multiplier != nil {
		a.Multiplier = *in.Multiplier
	}
	if in.MinimumMinutes != nil {
		a.MinimumMinutes = *in.MinimumMinutes
	}
	if err := validate(a); err != nil {
		return Activity{}, err
	}

	updated, err := s.repo.Update(ctx, a)
	if err != nil {
		return Activity{}, fmt.Errorf("update activity: %w", err)
	}
	return updated, nil
}

// Delete removes an activity together with its time entries.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return fmt.Errorf("delete activity: %w", err)
	}
	return nil
}

func validate(a Activity) error {
	if err := validation.Multiplier(a.Multiplier); err != nil {
		return err
	}
	if a.MinimumMinutes < 0 || a.MinimumMinutes > maxMinimumMinutes {
		return validation.Errorf("minimum time must be between 0 and %d minutes", maxMinimumMinutes)
	}
	return nil
}
