// Package activity implements the activities a user wants to spend time on. Activities
// belong to a category and are private to their owner.
package activity

import (
	"errors"
	"time"
)

// Activity is something to do, e.g. "swimming".
type Activity struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"-"`
	CategoryID int64  `json:"categoryId"`
	Name       string `json:"name"`
	// Multiplier is the weight relative to the activity's siblings.
	Multiplier float64 `json:"multiplier"`
	// MinimumMinutes is the shortest time slot in which the activity makes sense.
	MinimumMinutes int `json:"minimumMinutes"`
	// TrackedMinutes is the total time logged for this activity.
	TrackedMinutes int       `json:"trackedMinutes"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Domain errors returned by the service and repository.
var (
	ErrNotFound         = errors.New("activity not found")
	ErrCategoryNotFound = errors.New("category not found")
	ErrDuplicateName    = errors.New("an activity with this name already exists in the category")
)
