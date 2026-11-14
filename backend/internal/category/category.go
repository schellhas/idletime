// Package category implements the category domain: model, business rules,
// persistence and HTTP transport.
package category

import (
	"errors"
	"time"
)

// Category groups activities and can be nested inside another category.
type Category struct {
	ID         int64     `json:"id"`
	ParentID   *int64    `json:"parentId"`
	Name       string    `json:"name"`
	Multiplier float64   `json:"multiplier"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Domain errors returned by the service and repository.
var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrParentNotFound = errors.New("parent category not found")
	ErrDuplicateName  = errors.New("category name already exists in this parent")
)
