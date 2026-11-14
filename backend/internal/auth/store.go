package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/schellhas/idletime/backend/internal/user"
)

const maxStoredMetadataLength = 255

// PostgresSessionStore keeps sessions in PostgreSQL.
type PostgresSessionStore struct {
	pool *pgxpool.Pool
}

// NewSessionStore creates a PostgresSessionStore.
func NewSessionStore(pool *pgxpool.Pool) *PostgresSessionStore {
	return &PostgresSessionStore{pool: pool}
}

// Create stores the session and removes the user's expired ones.
func (s *PostgresSessionStore) Create(ctx context.Context, ns NewSession) error {
	_, err := s.pool.Exec(ctx,
		"INSERT INTO sessions (user_id, token_hash, expires_at, user_agent, ip_address) VALUES ($1, $2, $3, $4, $5)",
		ns.UserID, ns.TokenHash, ns.ExpiresAt, truncate(ns.UserAgent), truncate(ns.IPAddress))
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	if _, err := s.pool.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1 AND expires_at < now()", ns.UserID); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}

// Lookup returns the owner of a valid session, or user.ErrNotFound.
func (s *PostgresSessionStore) Lookup(ctx context.Context, tokenHash string) (user.User, error) {
	var u user.User
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.username, u.email, u.email_verified_at IS NOT NULL, u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`, tokenHash).
		Scan(&u.ID, &u.Username, &u.Email, &u.EmailVerified, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	if err != nil {
		return user.User{}, fmt.Errorf("query session: %w", err)
	}
	return u, nil
}

// Revoke deletes one session. Revoking an unknown session is not an error.
func (s *PostgresSessionStore) Revoke(ctx context.Context, tokenHash string) error {
	if _, err := s.pool.Exec(ctx, "DELETE FROM sessions WHERE token_hash = $1", tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// RevokeAll deletes all sessions of a user except the one with exceptTokenHash.
// An empty exceptTokenHash deletes every session.
func (s *PostgresSessionStore) RevokeAll(ctx context.Context, userID int64, exceptTokenHash string) error {
	if _, err := s.pool.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2", userID, exceptTokenHash); err != nil {
		return fmt.Errorf("delete sessions: %w", err)
	}
	return nil
}

// PostgresTokenStore keeps one-time email tokens in PostgreSQL.
type PostgresTokenStore struct {
	pool *pgxpool.Pool
}

// NewTokenStore creates a PostgresTokenStore.
func NewTokenStore(pool *pgxpool.Pool) *PostgresTokenStore {
	return &PostgresTokenStore{pool: pool}
}

// Issue stores a token and discards the user's older unused tokens of the same purpose.
func (s *PostgresTokenStore) Issue(ctx context.Context, userID int64, purpose, tokenHash string, expiresAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful commit

	if _, err := tx.Exec(ctx,
		"DELETE FROM user_tokens WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL", userID, purpose); err != nil {
		return fmt.Errorf("delete old tokens: %w", err)
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO user_tokens (user_id, purpose, token_hash, expires_at) VALUES ($1, $2, $3, $4)",
		userID, purpose, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("insert token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// Consume atomically marks a valid, unused token as used and returns its owner.
func (s *PostgresTokenStore) Consume(ctx context.Context, purpose, tokenHash string) (int64, error) {
	var userID int64
	err := s.pool.QueryRow(ctx, `
		UPDATE user_tokens SET used_at = now()
		WHERE purpose = $1 AND token_hash = $2 AND used_at IS NULL AND expires_at > now()
		RETURNING user_id`, purpose, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInvalidToken
	}
	if err != nil {
		return 0, fmt.Errorf("consume token: %w", err)
	}
	return userID, nil
}

func truncate(s string) string {
	if len(s) > maxStoredMetadataLength {
		return s[:maxStoredMetadataLength]
	}
	return s
}
