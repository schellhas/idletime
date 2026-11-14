package activity

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgreSQL error codes handled explicitly.
const (
	pgForeignKeyViolation = "23503"
	pgUniqueViolation     = "23505"
)

// selectActivity reads activities together with the total minutes tracked for them.
const selectActivity = `
	SELECT a.id, a.user_id, a.category_id, a.name, a.multiplier::float8, a.minimum_minutes,
	       COALESCE(SUM(t.minutes), 0)::int, a.created_at
	FROM activities a
	LEFT JOIN time_entries t ON t.activity_id = a.id`

// PostgresRepository stores activities in PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a repository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// List returns the user's activities ordered by name.
func (r *PostgresRepository) List(ctx context.Context, userID int64) ([]Activity, error) {
	rows, err := r.pool.Query(ctx, selectActivity+" WHERE a.user_id = $1 GROUP BY a.id ORDER BY LOWER(a.name), a.id", userID)
	if err != nil {
		return nil, fmt.Errorf("query activities: %w", err)
	}
	activities, err := pgx.CollectRows(rows, scanActivity)
	if err != nil {
		return nil, fmt.Errorf("collect activities: %w", err)
	}
	return activities, nil
}

// Get returns one activity of the user, or ErrNotFound.
func (r *PostgresRepository) Get(ctx context.Context, userID, id int64) (Activity, error) {
	rows, err := r.pool.Query(ctx, selectActivity+" WHERE a.id = $1 AND a.user_id = $2 GROUP BY a.id", id, userID)
	if err != nil {
		return Activity{}, fmt.Errorf("query activity: %w", err)
	}
	a, err := pgx.CollectOneRow(rows, scanActivity)
	if errors.Is(err, pgx.ErrNoRows) {
		return Activity{}, ErrNotFound
	}
	if err != nil {
		return Activity{}, fmt.Errorf("collect activity: %w", err)
	}
	return a, nil
}

// Create inserts a. The composite foreign key on (category_id, user_id) rejects
// categories that belong to another user.
func (r *PostgresRepository) Create(ctx context.Context, a Activity) (Activity, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		"INSERT INTO activities (user_id, category_id, name, multiplier, minimum_minutes) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		a.UserID, a.CategoryID, a.Name, a.Multiplier, a.MinimumMinutes).Scan(&id)
	if err != nil {
		return Activity{}, mapWriteError(err)
	}
	return r.Get(ctx, a.UserID, id)
}

// Update stores the mutable fields of a.
func (r *PostgresRepository) Update(ctx context.Context, a Activity) (Activity, error) {
	tag, err := r.pool.Exec(ctx,
		"UPDATE activities SET category_id = $3, name = $4, multiplier = $5, minimum_minutes = $6 WHERE id = $1 AND user_id = $2",
		a.ID, a.UserID, a.CategoryID, a.Name, a.Multiplier, a.MinimumMinutes)
	if err != nil {
		return Activity{}, mapWriteError(err)
	}
	if tag.RowsAffected() == 0 {
		return Activity{}, ErrNotFound
	}
	return r.Get(ctx, a.UserID, a.ID)
}

// Delete removes an activity; its time entries follow through cascading keys.
func (r *PostgresRepository) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM activities WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return fmt.Errorf("delete activity: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanActivity(row pgx.CollectableRow) (Activity, error) {
	var a Activity
	err := row.Scan(&a.ID, &a.UserID, &a.CategoryID, &a.Name, &a.Multiplier, &a.MinimumMinutes, &a.TrackedMinutes, &a.CreatedAt)
	return a, err
}

func mapWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgForeignKeyViolation:
			return ErrCategoryNotFound
		case pgUniqueViolation:
			return ErrDuplicateName
		}
	}
	return fmt.Errorf("write activity: %w", err)
}
