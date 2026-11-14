package integration_test

import (
	"errors"
	"testing"
	"time"

	"github.com/schellhas/idletime/backend/internal/auth"
	"github.com/schellhas/idletime/backend/internal/dbtest"
	"github.com/schellhas/idletime/backend/internal/user"
)

func TestUserRepository(t *testing.T) {
	db := dbtest.New(t)
	repo := user.NewRepository(db.Pool)

	alice, err := repo.Create(ctx, "alice", "alice@example.com", "hash-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if alice.EmailVerified {
		t.Error("a new user must be unverified")
	}

	t.Run("usernames and emails are unique regardless of case", func(t *testing.T) {
		if _, err := repo.Create(ctx, "ALICE", "other@example.com", "h"); !errors.Is(err, user.ErrDuplicateUsername) {
			t.Errorf("duplicate username: error = %v, want ErrDuplicateUsername", err)
		}
		if _, err := repo.Create(ctx, "someone", "ALICE@Example.com", "h"); !errors.Is(err, user.ErrDuplicateEmail) {
			t.Errorf("duplicate email: error = %v, want ErrDuplicateEmail", err)
		}
	})

	t.Run("finds users by email or username, case-insensitively", func(t *testing.T) {
		for _, identifier := range []string{"alice", "ALICE", "alice@example.com", "Alice@EXAMPLE.com"} {
			rec, err := repo.RecordByIdentifier(ctx, identifier)
			if err != nil || rec.ID != alice.ID || rec.PasswordHash != "hash-1" {
				t.Errorf("RecordByIdentifier(%q) = %+v, %v", identifier, rec, err)
			}
		}
		if _, err := repo.RecordByIdentifier(ctx, "nobody"); !errors.Is(err, user.ErrNotFound) {
			t.Errorf("unknown identifier: error = %v, want ErrNotFound", err)
		}
		if rec, err := repo.RecordByEmail(ctx, "alice@example.com"); err != nil || rec.ID != alice.ID {
			t.Errorf("RecordByEmail() = %+v, %v", rec, err)
		}
	})

	t.Run("verification is idempotent", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			if err := repo.MarkVerified(ctx, alice.ID); err != nil {
				t.Fatalf("MarkVerified() error = %v", err)
			}
		}
		if got, _ := repo.ByID(ctx, alice.ID); !got.EmailVerified {
			t.Error("user should be verified")
		}
	})

	t.Run("username and password can be changed", func(t *testing.T) {
		bob, _ := repo.Create(ctx, "bob", "bob@example.com", "h")
		if _, err := repo.UpdateUsername(ctx, bob.ID, "Alice"); !errors.Is(err, user.ErrDuplicateUsername) {
			t.Errorf("taking a name: error = %v, want ErrDuplicateUsername", err)
		}
		if updated, err := repo.UpdateUsername(ctx, bob.ID, "robert"); err != nil || updated.Username != "robert" {
			t.Errorf("UpdateUsername() = %+v, %v", updated, err)
		}
		if err := repo.UpdatePassword(ctx, bob.ID, "hash-2"); err != nil {
			t.Fatalf("UpdatePassword() error = %v", err)
		}
		if rec, _ := repo.RecordByID(ctx, bob.ID); rec.PasswordHash != "hash-2" {
			t.Errorf("password hash = %q, want hash-2", rec.PasswordHash)
		}
	})

	t.Run("deleting removes the user once", func(t *testing.T) {
		carol, _ := repo.Create(ctx, "carol", "carol@example.com", "h")
		if err := repo.Delete(ctx, carol.ID); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if _, err := repo.ByID(ctx, carol.ID); !errors.Is(err, user.ErrNotFound) {
			t.Errorf("ByID after delete: error = %v, want ErrNotFound", err)
		}
		if err := repo.Delete(ctx, carol.ID); !errors.Is(err, user.ErrNotFound) {
			t.Errorf("second Delete: error = %v, want ErrNotFound", err)
		}
	})
}

func TestSessionStore(t *testing.T) {
	db := dbtest.New(t)
	alice := dbtest.CreateUser(t, db.Pool, "alice")
	store := auth.NewSessionStore(db.Pool)
	hour := time.Hour

	create := func(hash string, expiresIn time.Duration) {
		t.Helper()
		err := store.Create(ctx, auth.NewSession{UserID: alice.ID, TokenHash: hash, ExpiresAt: time.Now().Add(expiresIn)})
		if err != nil {
			t.Fatalf("Create(%s) error = %v", hash, err)
		}
	}

	t.Run("a valid session resolves to its user", func(t *testing.T) {
		create("valid", hour)
		got, err := store.Lookup(ctx, "valid")
		if err != nil || got.ID != alice.ID || got.Username != "alice" {
			t.Errorf("Lookup() = %+v, %v", got, err)
		}
	})

	t.Run("unknown and expired sessions are not found", func(t *testing.T) {
		create("expired", -hour)
		for _, hash := range []string{"expired", "unknown"} {
			if _, err := store.Lookup(ctx, hash); !errors.Is(err, user.ErrNotFound) {
				t.Errorf("Lookup(%s): error = %v, want ErrNotFound", hash, err)
			}
		}
	})

	t.Run("creating a session cleans up the user's expired ones", func(t *testing.T) {
		create("another", hour)
		if n := count(t, db.Pool, "SELECT count(*) FROM sessions WHERE token_hash = 'expired'"); n != 0 {
			t.Errorf("expired sessions left: %d", n)
		}
	})

	t.Run("revoking ends one session, revoke-all keeps the exception", func(t *testing.T) {
		if err := store.Revoke(ctx, "another"); err != nil {
			t.Fatalf("Revoke() error = %v", err)
		}
		if _, err := store.Lookup(ctx, "another"); !errors.Is(err, user.ErrNotFound) {
			t.Error("revoked session must not resolve")
		}

		create("keep", hour)
		create("drop", hour)
		if err := store.RevokeAll(ctx, alice.ID, "keep"); err != nil {
			t.Fatalf("RevokeAll() error = %v", err)
		}
		if _, err := store.Lookup(ctx, "keep"); err != nil {
			t.Errorf("the excepted session was revoked: %v", err)
		}
		for _, hash := range []string{"drop", "valid"} {
			if _, err := store.Lookup(ctx, hash); !errors.Is(err, user.ErrNotFound) {
				t.Errorf("session %q survived RevokeAll", hash)
			}
		}

		if err := store.RevokeAll(ctx, alice.ID, ""); err != nil {
			t.Fatalf("RevokeAll(all) error = %v", err)
		}
		if n := count(t, db.Pool, "SELECT count(*) FROM sessions"); n != 0 {
			t.Errorf("sessions left: %d", n)
		}
	})
}

func TestTokenStore(t *testing.T) {
	db := dbtest.New(t)
	alice := dbtest.CreateUser(t, db.Pool, "alice")
	store := auth.NewTokenStore(db.Pool)
	future, past := time.Now().Add(time.Hour), time.Now().Add(-time.Hour)

	issue := func(purpose, hash string, expires time.Time) {
		t.Helper()
		if err := store.Issue(ctx, alice.ID, purpose, hash, expires); err != nil {
			t.Fatalf("Issue(%s) error = %v", hash, err)
		}
	}

	t.Run("a token can be used exactly once", func(t *testing.T) {
		issue(auth.PurposeEmailVerification, "once", future)
		if id, err := store.Consume(ctx, auth.PurposeEmailVerification, "once"); err != nil || id != alice.ID {
			t.Fatalf("Consume() = %d, %v", id, err)
		}
		if _, err := store.Consume(ctx, auth.PurposeEmailVerification, "once"); !errors.Is(err, auth.ErrInvalidToken) {
			t.Errorf("second use: error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("tokens only work for their own purpose", func(t *testing.T) {
		issue(auth.PurposePasswordReset, "reset", future)
		if _, err := store.Consume(ctx, auth.PurposeEmailVerification, "reset"); !errors.Is(err, auth.ErrInvalidToken) {
			t.Errorf("wrong purpose: error = %v, want ErrInvalidToken", err)
		}
		if _, err := store.Consume(ctx, auth.PurposePasswordReset, "reset"); err != nil {
			t.Errorf("right purpose: error = %v", err)
		}
	})

	t.Run("expired and unknown tokens are rejected", func(t *testing.T) {
		issue(auth.PurposeEmailVerification, "old", past)
		for _, hash := range []string{"old", "never-issued"} {
			if _, err := store.Consume(ctx, auth.PurposeEmailVerification, hash); !errors.Is(err, auth.ErrInvalidToken) {
				t.Errorf("%s: error = %v, want ErrInvalidToken", hash, err)
			}
		}
	})

	t.Run("a newer token replaces the older unused one", func(t *testing.T) {
		issue(auth.PurposeEmailVerification, "first", future)
		issue(auth.PurposeEmailVerification, "second", future)
		if _, err := store.Consume(ctx, auth.PurposeEmailVerification, "first"); !errors.Is(err, auth.ErrInvalidToken) {
			t.Errorf("replaced token: error = %v, want ErrInvalidToken", err)
		}
		if _, err := store.Consume(ctx, auth.PurposeEmailVerification, "second"); err != nil {
			t.Errorf("newest token: error = %v", err)
		}
	})
}
