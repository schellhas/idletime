// Package account lets a logged-in user manage their own account and data.
package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/schellhas/idletime/backend/internal/category"
	"github.com/schellhas/idletime/backend/internal/password"
	"github.com/schellhas/idletime/backend/internal/user"
)

// ErrWrongPassword is returned when the confirmation password does not match.
var ErrWrongPassword = errors.New("wrong password")

// UserStore is the user persistence required by the Service.
type UserStore interface {
	RecordByID(ctx context.Context, id int64) (user.Record, error)
	UpdateUsername(ctx context.Context, id int64, username string) (user.User, error)
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	Delete(ctx context.Context, id int64) error
}

// SessionRevoker ends login sessions.
type SessionRevoker interface {
	RevokeAll(ctx context.Context, userID int64, exceptTokenHash string) error
}

// CategoryLister provides the user's categories for the data export.
type CategoryLister interface {
	List(ctx context.Context, userID int64) ([]category.Category, error)
}

// Export is a complete copy of everything stored about a user.
type Export struct {
	ExportedAt time.Time           `json:"exportedAt"`
	User       user.User           `json:"user"`
	Categories []category.Category `json:"categories"`
}

// Service contains the account-management use cases.
type Service struct {
	users      UserStore
	sessions   SessionRevoker
	categories CategoryLister
}

// NewService creates a Service.
func NewService(users UserStore, sessions SessionRevoker, categories CategoryLister) *Service {
	return &Service{users: users, sessions: sessions, categories: categories}
}

// UpdateUsername changes the user's username.
func (s *Service) UpdateUsername(ctx context.Context, userID int64, username string) (user.User, error) {
	username = user.NormalizeUsername(username)
	if err := user.ValidateUsername(username); err != nil {
		return user.User{}, err
	}
	updated, err := s.users.UpdateUsername(ctx, userID, username)
	if err != nil {
		return user.User{}, fmt.Errorf("update username: %w", err)
	}
	return updated, nil
}

// ChangePassword sets a new password after confirming the current one and ends all
// other sessions. keepTokenHash identifies the session that stays logged in.
func (s *Service) ChangePassword(ctx context.Context, userID int64, current, next, keepTokenHash string) error {
	if err := s.confirmPassword(ctx, userID, current); err != nil {
		return err
	}
	if err := password.Validate(next); err != nil {
		return err
	}
	hash, err := password.Hash(next)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.users.UpdatePassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if err := s.sessions.RevokeAll(ctx, userID, keepTokenHash); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}
	return nil
}

// Delete permanently removes the account and all its data after confirming the password.
func (s *Service) Delete(ctx context.Context, userID int64, confirmation string) error {
	if err := s.confirmPassword(ctx, userID, confirmation); err != nil {
		return err
	}
	if err := s.users.Delete(ctx, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// Export collects all data stored about the user.
func (s *Service) Export(ctx context.Context, u user.User) (Export, error) {
	categories, err := s.categories.List(ctx, u.ID)
	if err != nil {
		return Export{}, fmt.Errorf("export categories: %w", err)
	}
	if categories == nil {
		categories = []category.Category{}
	}
	return Export{ExportedAt: time.Now().UTC(), User: u, Categories: categories}, nil
}

func (s *Service) confirmPassword(ctx context.Context, userID int64, candidate string) error {
	rec, err := s.users.RecordByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("load user: %w", err)
	}
	if !password.Matches(rec.PasswordHash, candidate) {
		return ErrWrongPassword
	}
	return nil
}
