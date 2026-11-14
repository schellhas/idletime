// Package dbtest gives integration tests a real, freshly migrated PostgreSQL database.
//
// Tests need TEST_DATABASE_URL, pointing at a server on which the user may create
// databases (e.g. postgres://test:test@localhost:5432/postgres?sslmode=disable). Without
// it the tests are skipped, so a plain `go test ./...` keeps working without a database.
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/schellhas/idletime/backend/internal/database"
	"github.com/schellhas/idletime/backend/internal/user"
)

// EnvVar names the environment variable holding the admin connection URL.
const EnvVar = "TEST_DATABASE_URL"

// DB is a throwaway database that exists for the duration of one test.
type DB struct {
	Pool *pgxpool.Pool
	// URL connects to the throwaway database itself.
	URL string
}

// New creates a database with a random name, applies all migrations and removes the
// database when the test ends. Every test therefore starts from an empty, current schema.
func New(t testing.TB) *DB {
	t.Helper()

	adminURL := os.Getenv(EnvVar)
	if adminURL == "" {
		t.Skipf("%s is not set; skipping database integration test", EnvVar)
	}
	ctx := context.Background()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect to test server: %v", err)
	}

	name := "idletime_test_" + randomSuffix(t)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("create database: %v", err)
	}

	parsed, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("parse %s: %v", EnvVar, err)
	}
	parsed.Path = "/" + name
	testURL := parsed.String()

	if err := database.Migrate(ctx, testURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := database.Connect(ctx, testURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop database: %v", err)
		}
		_ = admin.Close(ctx)
	})
	return &DB{Pool: pool, URL: testURL}
}

// CreateUser stores a verified user named username with the email username@example.com.
func CreateUser(t testing.TB, pool *pgxpool.Pool, username string) user.User {
	t.Helper()
	repo := user.NewRepository(pool)
	created, err := repo.Create(context.Background(), username, username+"@example.com", "not-a-real-hash")
	if err != nil {
		t.Fatalf("create user %q: %v", username, err)
	}
	if err := repo.MarkVerified(context.Background(), created.ID); err != nil {
		t.Fatalf("verify user %q: %v", username, err)
	}
	return created
}

func randomSuffix(t testing.TB) string {
	t.Helper()
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("random: %v", err)
	}
	return hex.EncodeToString(buf)
}
