package timer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/schellhas/idletime/backend/internal/timeentry"
)

// PostgreSQL error codes handled explicitly.
const (
	pgForeignKeyViolation = "23503"
	pgUniqueViolation     = "23505"
)

// PostgresRepository stores running timers in PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a repository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Start inserts the timer. user_id is the primary key, so a second timer is rejected,
// and the composite foreign key rejects activities of other users.
func (r *PostgresRepository) Start(ctx context.Context, userID, activityID int64, startedAt time.Time) (Timer, error) {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO running_timers (user_id, activity_id, started_at) VALUES ($1, $2, $3)",
		userID, activityID, startedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return Timer{}, ErrAlreadyRunning
		case pgForeignKeyViolation:
			return Timer{}, ErrActivityNotFound
		}
	}
	if err != nil {
		return Timer{}, fmt.Errorf("insert timer: %w", err)
	}
	return Timer{ActivityID: activityID, StartedAt: startedAt}, nil
}

// Get returns the running timer, or ErrNotRunning.
func (r *PostgresRepository) Get(ctx context.Context, userID int64) (Timer, error) {
	var t Timer
	err := r.pool.QueryRow(ctx,
		"SELECT activity_id, started_at FROM running_timers WHERE user_id = $1", userID).
		Scan(&t.ActivityID, &t.StartedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Timer{}, ErrNotRunning
	}
	if err != nil {
		return Timer{}, fmt.Errorf("query timer: %w", err)
	}
	return t, nil
}

// Cancel deletes the running timer.
func (r *PostgresRepository) Cancel(ctx context.Context, userID int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM running_timers WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("delete timer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotRunning
	}
	return nil
}

// Finish deletes the timer and inserts the resulting time entry in one transaction, so
// time is neither lost nor logged twice.
func (r *PostgresRepository) Finish(ctx context.Context, userID int64, minutesFor func(time.Time) int) (timeentry.Entry, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return timeentry.Entry{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful commit

	var activityID int64
	var startedAt time.Time
	err = tx.QueryRow(ctx,
		"DELETE FROM running_timers WHERE user_id = $1 RETURNING activity_id, started_at", userID).
		Scan(&activityID, &startedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return timeentry.Entry{}, ErrNotRunning
	}
	if err != nil {
		return timeentry.Entry{}, fmt.Errorf("delete timer: %w", err)
	}

	var e timeentry.Entry
	err = tx.QueryRow(ctx, `
		INSERT INTO time_entries (user_id, activity_id, minutes, logged_at) VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, activity_id, minutes, COALESCE(note, ''), logged_at, created_at`,
		userID, activityID, minutesFor(startedAt), startedAt).
		Scan(&e.ID, &e.UserID, &e.ActivityID, &e.Minutes, &e.Note, &e.LoggedAt, &e.CreatedAt)
	if err != nil {
		return timeentry.Entry{}, fmt.Errorf("insert time entry: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return timeentry.Entry{}, fmt.Errorf("commit: %w", err)
	}
	return e, nil
}
