// Package user implements the user account model and its persistence.
package user

import (
	"errors"
	"time"
)

// User is the public view of an account.
type User struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"emailVerified"`
	CreatedAt     time.Time `json:"createdAt"`
}

// Record is a User together with its secret credentials. It never leaves the backend.
type Record struct {
	User
	PasswordHash string
}

// Errors returned by the repository.
var (
	ErrNotFound          = errors.New("user not found")
	ErrDuplicateUsername = errors.New("username already taken")
	ErrDuplicateEmail    = errors.New("email already registered")
)
