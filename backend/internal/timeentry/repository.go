package timeentry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pgForeignKeyViolation = "23503"

const entryColumns = "id, user_id, activity_id, minutes, COALESCE(note, ''), logged_at, created_at"

// PostgresRepository stores time entries in PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a repository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Create inserts e. The composite foreign key on (activity_id, user_id) rejects
// activities that belong to another user.
func (r *PostgresRepository) Create(ctx context.Context, e Entry) (Entry, error) {
	return r.write(ctx,
		"INSERT INTO time_entries (user_id, activity_id, minutes, note, logged_at) VALUES ($1, $2, $3, NULLIF($4, ''), $5) RETURNING "+entryColumns,
		e.UserID, e.ActivityID, e.Minutes, e.Note, e.LoggedAt)
}

// Get returns one entry of the user, or ErrNotFound.
func (r *PostgresRepository) Get(ctx context.Context, userID, id int64) (Entry, error) {
	return r.write(ctx, "SELECT "+entryColumns+" FROM time_entries WHERE id = $1 AND user_id = $2", id, userID)
}

// Update stores the mutable fields of e.
func (r *PostgresRepository) Update(ctx context.Context, e Entry) (Entry, error) {
	return r.write(ctx,
		"UPDATE time_entries SET minutes = $3, note = NULLIF($4, ''), logged_at = $5 WHERE id = $1 AND user_id = $2 RETURNING "+entryColumns,
		e.ID, e.UserID, e.Minutes, e.Note, e.LoggedAt)
}

// Delete removes an entry.
func (r *PostgresRepository) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM time_entries WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return fmt.Errorf("delete time entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// List returns entries newest first. A negative limit means no limit.
func (r *PostgresRepository) List(ctx context.Context, userID int64, f ListFilter) ([]Entry, error) {
	var limit any // NULL means no LIMIT in PostgreSQL
	if f.Limit >= 0 {
		limit = f.Limit
	}
	rows, err := r.pool.Query(ctx,
		"SELECT "+entryColumns+` FROM time_entries
		 WHERE user_id = $1 AND ($2::bigint IS NULL OR activity_id = $2)
		 ORDER BY logged_at DESC, id DESC LIMIT $3`,
		userID, f.ActivityID, limit)
	if err != nil {
		return nil, fmt.Errorf("query time entries: %w", err)
	}
	entries, err := pgx.CollectRows(rows, scanEntry)
	if err != nil {
		return nil, fmt.Errorf("collect time entries: %w", err)
	}
	return entries, nil
}

// MinutesPerDay sums minutes per calendar day in timeZone, oldest day first. A nil
// activityIDs counts every activity, otherwise only the listed ones.
func (r *PostgresRepository) MinutesPerDay(ctx context.Context, userID int64, timeZone string, since time.Time, activityIDs []int64) ([]DayMinutes, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT to_char((logged_at AT TIME ZONE $2::text)::date, 'YYYY-MM-DD') AS day, SUM(minutes)::int
		FROM time_entries
		WHERE user_id = $1 AND logged_at >= $3 AND ($4::bigint[] IS NULL OR activity_id = ANY($4))
		GROUP BY day ORDER BY day`, userID, timeZone, since, activityIDs)
	if err != nil {
		return nil, fmt.Errorf("query minutes per day: %w", err)
	}
	days, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DayMinutes, error) {
		var d DayMinutes
		err := row.Scan(&d.Date, &d.Minutes)
		return d, err
	})
	if err != nil {
		return nil, fmt.Errorf("collect minutes per day: %w", err)
	}
	return days, nil
}

func (r *PostgresRepository) write(ctx context.Context, query string, args ...any) (Entry, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return Entry{}, mapError(err)
	}
	e, err := pgx.CollectOneRow(rows, scanEntry)
	if err != nil {
		return Entry{}, mapError(err)
	}
	return e, nil
}

func scanEntry(row pgx.CollectableRow) (Entry, error) {
	var e Entry
	err := row.Scan(&e.ID, &e.UserID, &e.ActivityID, &e.Minutes, &e.Note, &e.LoggedAt, &e.CreatedAt)
	return e, err
}

func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		return ErrActivityNotFound
	}
	return fmt.Errorf("time entry query: %w", err)
}
