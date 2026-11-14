package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	pgUniqueViolation = "23505"

	usernameConstraint = "users_username_lower_uq"
	emailConstraint    = "users_email_lower_uq"

	userColumns = "id, username, email, email_verified_at IS NOT NULL, created_at"
)

// Repository stores users in PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a Repository backed by pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new, unverified user.
func (r *Repository) Create(ctx context.Context, username, email, passwordHash string) (User, error) {
	return scanUser(r.pool.QueryRow(ctx,
		"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING "+userColumns,
		username, email, passwordHash))
}

// ByID returns the user with the given id.
func (r *Repository) ByID(ctx context.Context, id int64) (User, error) {
	return scanUser(r.pool.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE id = $1", id))
}

// RecordByID returns the user including credentials.
func (r *Repository) RecordByID(ctx context.Context, id int64) (Record, error) {
	return scanRecord(r.pool.QueryRow(ctx,
		"SELECT "+userColumns+", password_hash FROM users WHERE id = $1", id))
}

// RecordByEmail returns the user with the given (normalized) email including credentials.
func (r *Repository) RecordByEmail(ctx context.Context, email string) (Record, error) {
	return scanRecord(r.pool.QueryRow(ctx,
		"SELECT "+userColumns+", password_hash FROM users WHERE LOWER(email) = LOWER($1)", email))
}

// RecordByIdentifier finds a user by email or username (case-insensitive).
// Usernames cannot contain '@', so an identifier never matches both.
func (r *Repository) RecordByIdentifier(ctx context.Context, identifier string) (Record, error) {
	return scanRecord(r.pool.QueryRow(ctx,
		"SELECT "+userColumns+", password_hash FROM users WHERE LOWER(email) = LOWER($1) OR LOWER(username) = LOWER($1) LIMIT 1",
		identifier))
}

// MarkVerified records that the user's email address was confirmed.
func (r *Repository) MarkVerified(ctx context.Context, id int64) error {
	return r.exec(ctx,
		"UPDATE users SET email_verified_at = COALESCE(email_verified_at, now()), updated_at = now() WHERE id = $1", id)
}

// UpdateUsername changes the username and returns the updated user.
func (r *Repository) UpdateUsername(ctx context.Context, id int64, username string) (User, error) {
	return scanUser(r.pool.QueryRow(ctx,
		"UPDATE users SET username = $2, updated_at = now() WHERE id = $1 RETURNING "+userColumns, id, username))
}

// UpdatePassword replaces the stored password hash.
func (r *Repository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	return r.exec(ctx, "UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1", id, passwordHash)
}

// Delete removes the user and, through cascading foreign keys, all of their data.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	return r.exec(ctx, "DELETE FROM users WHERE id = $1", id)
}

func (r *Repository) exec(ctx context.Context, query string, args ...any) error {
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.EmailVerified, &u.CreatedAt)
	return u, mapError(err)
}

func scanRecord(row pgx.Row) (Record, error) {
	var rec Record
	err := row.Scan(&rec.ID, &rec.Username, &rec.Email, &rec.EmailVerified, &rec.CreatedAt, &rec.PasswordHash)
	return rec, mapError(err)
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		switch pgErr.ConstraintName {
		case usernameConstraint:
			return ErrDuplicateUsername
		case emailConstraint:
			return ErrDuplicateEmail
		}
	}
	return fmt.Errorf("user query: %w", err)
}
