package category

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

const categoryColumns = "id, user_id, parent_id, name, multiplier::float8, created_at"

// PostgresRepository stores categories in PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a repository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// List returns the user's categories ordered by name.
func (r *PostgresRepository) List(ctx context.Context, userID int64) ([]Category, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT "+categoryColumns+" FROM categories WHERE user_id = $1 ORDER BY LOWER(name), id", userID)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	categories, err := pgx.CollectRows(rows, scanCategory)
	if err != nil {
		return nil, fmt.Errorf("collect categories: %w", err)
	}
	return categories, nil
}

// Create inserts c and returns the stored row. The composite foreign key on
// (parent_id, user_id) rejects parents that belong to another user.
func (r *PostgresRepository) Create(ctx context.Context, c Category) (Category, error) {
	return r.write(ctx,
		"INSERT INTO categories (user_id, parent_id, name, multiplier) VALUES ($1, $2, $3, $4) RETURNING "+categoryColumns,
		c.UserID, c.ParentID, c.Name, c.Multiplier)
}

// Update stores the mutable fields of c and returns the stored row.
func (r *PostgresRepository) Update(ctx context.Context, c Category) (Category, error) {
	return r.write(ctx,
		"UPDATE categories SET parent_id = $3, name = $4, multiplier = $5 WHERE id = $1 AND user_id = $2 RETURNING "+categoryColumns,
		c.ID, c.UserID, c.ParentID, c.Name, c.Multiplier)
}

// Delete removes a category; subcategories and activities follow through cascading keys.
func (r *PostgresRepository) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM categories WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) write(ctx context.Context, query string, args ...any) (Category, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return Category{}, mapWriteError(err)
	}
	c, err := pgx.CollectOneRow(rows, scanCategory)
	if err != nil {
		return Category{}, mapWriteError(err)
	}
	return c, nil
}

func scanCategory(row pgx.CollectableRow) (Category, error) {
	var c Category
	err := row.Scan(&c.ID, &c.UserID, &c.ParentID, &c.Name, &c.Multiplier, &c.CreatedAt)
	return c, err
}

func mapWriteError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgForeignKeyViolation:
			return ErrParentNotFound
		case pgUniqueViolation:
			return ErrDuplicateName
		}
	}
	return fmt.Errorf("write category: %w", err)
}
