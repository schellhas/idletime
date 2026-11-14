package integration_test

import (
	"database/sql"
	"strconv"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"

	"github.com/schellhas/idletime/backend/internal/activity"
	"github.com/schellhas/idletime/backend/internal/auth"
	"github.com/schellhas/idletime/backend/internal/category"
	"github.com/schellhas/idletime/backend/internal/database"
	"github.com/schellhas/idletime/backend/internal/dbtest"
	"github.com/schellhas/idletime/backend/internal/timeentry"
	"github.com/schellhas/idletime/backend/internal/timer"
	"github.com/schellhas/idletime/backend/internal/user"
)

func TestMigrationsCanBeRolledBackAndReapplied(t *testing.T) {
	db := dbtest.New(t) // already migrated up

	sqlDB, err := sql.Open("pgx", db.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, database.Migrations())
	if err != nil {
		t.Fatalf("provider: %v", err)
	}

	if _, err := provider.DownTo(ctx, 0); err != nil {
		t.Fatalf("rolling back all migrations: %v", err)
	}
	if n := count(t, db.Pool, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('users', 'categories', 'activities')"); n != 0 {
		t.Errorf("%d application tables remain after rolling back", n)
	}

	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("re-applying migrations: %v", err)
	}
	if n := count(t, db.Pool, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('users', 'categories', 'activities', 'time_entries', 'running_timers', 'sessions', 'user_tokens')"); n != 7 {
		t.Errorf("%d of 7 application tables exist after re-applying", n)
	}
}

func TestDeletingAUserRemovesAllTheirData(t *testing.T) {
	db := dbtest.New(t)
	alice := dbtest.CreateUser(t, db.Pool, "alice")
	bob := dbtest.CreateUser(t, db.Pool, "bob")

	categories := category.NewPostgresRepository(db.Pool)
	activities := activity.NewPostgresRepository(db.Pool)
	entries := timeentry.NewPostgresRepository(db.Pool)
	timers := timer.NewPostgresRepository(db.Pool)
	sessions := auth.NewSessionStore(db.Pool)
	tokens := auth.NewTokenStore(db.Pool)

	seed := func(owner int64, label string) {
		cat := mustCategory(t, categories, owner, label+"-category", nil)
		act := mustActivity(t, activities, owner, cat.ID, label+"-activity")
		mustEntry(t, entries, owner, act.ID, 10, time.Now().Add(-time.Hour))
		if _, err := timers.Start(ctx, owner, act.ID, time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := sessions.Create(ctx, auth.NewSession{UserID: owner, TokenHash: label + "-session", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
		if err := tokens.Issue(ctx, owner, auth.PurposeEmailVerification, label+"-token", time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	seed(alice.ID, "alice")
	seed(bob.ID, "bob")

	if err := user.NewRepository(db.Pool).Delete(ctx, alice.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	for _, table := range []string{"categories", "activities", "time_entries", "running_timers", "sessions", "user_tokens"} {
		if n := count(t, db.Pool, "SELECT count(*) FROM "+table+" WHERE user_id = $1", alice.ID); n != 0 {
			t.Errorf("%s: %d rows of the deleted user remain", table, n)
		}
		if n := count(t, db.Pool, "SELECT count(*) FROM "+table+" WHERE user_id = $1", bob.ID); n != 1 {
			t.Errorf("%s: other user has %d rows, want 1 (must be untouched)", table, n)
		}
	}
}

func TestSchemaRejectsInvalidValues(t *testing.T) {
	db := dbtest.New(t)
	alice := dbtest.CreateUser(t, db.Pool, "alice")
	cat := mustCategory(t, category.NewPostgresRepository(db.Pool), alice.ID, "Life", nil)
	act := mustActivity(t, activity.NewPostgresRepository(db.Pool), alice.ID, cat.ID, "Reading")

	// The constraints are the last line of defence if application validation is ever bypassed.
	statements := map[string]string{
		"non-positive category multiplier": "UPDATE categories SET multiplier = 0",
		"negative minimum minutes":         "UPDATE activities SET minimum_minutes = -1",
		"minimum over one day":             "UPDATE activities SET minimum_minutes = 1441",
		"zero-minute entry":                "INSERT INTO time_entries (user_id, activity_id, minutes) VALUES (" + id(alice.ID) + ", " + id(act.ID) + ", 0)",
		"entry longer than a day":          "INSERT INTO time_entries (user_id, activity_id, minutes) VALUES (" + id(alice.ID) + ", " + id(act.ID) + ", 1441)",
		"too short username":               "UPDATE users SET username = 'ab'",
	}
	for name, statement := range statements {
		if _, err := db.Pool.Exec(ctx, statement); err == nil {
			t.Errorf("%s: the database accepted %q", name, statement)
		}
	}
}

func id(n int64) string { return strconv.FormatInt(n, 10) }
