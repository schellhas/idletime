// Package integration_test exercises the PostgreSQL repositories and the schema against a
// real database. See package dbtest for how to run it.
package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/schellhas/idletime/backend/internal/activity"
	"github.com/schellhas/idletime/backend/internal/category"
	"github.com/schellhas/idletime/backend/internal/timeentry"
)

var ctx = context.Background()

func ptr[T any](v T) *T { return &v }

func mustCategory(t *testing.T, repo *category.PostgresRepository, userID int64, name string, parent *int64) category.Category {
	t.Helper()
	c, err := repo.Create(ctx, category.Category{UserID: userID, ParentID: parent, Name: name, Multiplier: 1})
	if err != nil {
		t.Fatalf("create category %q: %v", name, err)
	}
	return c
}

func mustActivity(t *testing.T, repo *activity.PostgresRepository, userID, categoryID int64, name string) activity.Activity {
	t.Helper()
	a, err := repo.Create(ctx, activity.Activity{UserID: userID, CategoryID: categoryID, Name: name, Multiplier: 1})
	if err != nil {
		t.Fatalf("create activity %q: %v", name, err)
	}
	return a
}

func mustEntry(t *testing.T, repo *timeentry.PostgresRepository, userID, activityID int64, minutes int, at time.Time) timeentry.Entry {
	t.Helper()
	e, err := repo.Create(ctx, timeentry.Entry{UserID: userID, ActivityID: activityID, Minutes: minutes, LoggedAt: at})
	if err != nil {
		t.Fatalf("create time entry: %v", err)
	}
	return e
}

func count(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx, query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}
