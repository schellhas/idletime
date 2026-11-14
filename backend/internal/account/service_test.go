package account

import (
	"context"
	"errors"
	"testing"

	"github.com/schellhas/idletime/backend/internal/category"
	"github.com/schellhas/idletime/backend/internal/password"
	"github.com/schellhas/idletime/backend/internal/user"
	"github.com/schellhas/idletime/backend/internal/validation"
)

const currentPassword = "correct horse battery"

type fakeUsers struct {
	record  user.Record
	deleted bool
}

func (f *fakeUsers) RecordByID(context.Context, int64) (user.Record, error) { return f.record, nil }

func (f *fakeUsers) UpdateUsername(_ context.Context, _ int64, username string) (user.User, error) {
	f.record.Username = username
	return f.record.User, nil
}

func (f *fakeUsers) UpdatePassword(_ context.Context, _ int64, hash string) error {
	f.record.PasswordHash = hash
	return nil
}

func (f *fakeUsers) Delete(context.Context, int64) error {
	f.deleted = true
	return nil
}

type fakeSessions struct{ keptHash string }

func (f *fakeSessions) RevokeAll(_ context.Context, _ int64, except string) error {
	f.keptHash = except
	return nil
}

type fakeCategories struct{ list []category.Category }

func (f fakeCategories) List(context.Context, int64) ([]category.Category, error) { return f.list, nil }

func newService(t *testing.T) (*Service, *fakeUsers, *fakeSessions) {
	t.Helper()
	hash, err := password.Hash(currentPassword)
	if err != nil {
		t.Fatal(err)
	}
	users := &fakeUsers{record: user.Record{User: user.User{ID: 1, Username: "alice"}, PasswordHash: hash}}
	sessions := &fakeSessions{}
	return NewService(users, sessions, fakeCategories{}), users, sessions
}

func TestUpdateUsername(t *testing.T) {
	svc, _, _ := newService(t)

	updated, err := svc.UpdateUsername(context.Background(), 1, "  alice2 ")
	if err != nil || updated.Username != "alice2" {
		t.Fatalf("UpdateUsername() = %+v, %v", updated, err)
	}
	if _, err := svc.UpdateUsername(context.Background(), 1, "x"); !errors.Is(err, validation.ErrInvalid) {
		t.Errorf("error = %v, want validation error", err)
	}
}

func TestChangePassword(t *testing.T) {
	ctx := context.Background()

	t.Run("requires the current password", func(t *testing.T) {
		svc, users, _ := newService(t)
		before := users.record.PasswordHash

		err := svc.ChangePassword(ctx, 1, "not my password", "a brand new password", "keep")

		if !errors.Is(err, ErrWrongPassword) || users.record.PasswordHash != before {
			t.Errorf("error = %v, hash changed = %v", err, users.record.PasswordHash != before)
		}
	})

	t.Run("updates the hash and keeps only the current session", func(t *testing.T) {
		svc, users, sessions := newService(t)

		if err := svc.ChangePassword(ctx, 1, currentPassword, "a brand new password", "keep-me"); err != nil {
			t.Fatalf("error = %v", err)
		}
		if !password.Matches(users.record.PasswordHash, "a brand new password") {
			t.Error("new password was not stored")
		}
		if sessions.keptHash != "keep-me" {
			t.Errorf("kept session = %q, want keep-me", sessions.keptHash)
		}
	})

	t.Run("enforces the password policy", func(t *testing.T) {
		svc, _, _ := newService(t)
		if err := svc.ChangePassword(ctx, 1, currentPassword, "short", ""); !errors.Is(err, validation.ErrInvalid) {
			t.Errorf("error = %v, want validation error", err)
		}
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	svc, users, _ := newService(t)

	if err := svc.Delete(ctx, 1, "wrong"); !errors.Is(err, ErrWrongPassword) || users.deleted {
		t.Fatalf("a wrong password must not delete the account (err=%v, deleted=%v)", err, users.deleted)
	}
	if err := svc.Delete(ctx, 1, currentPassword); err != nil || !users.deleted {
		t.Fatalf("Delete() error = %v, deleted = %v", err, users.deleted)
	}
}

func TestExportReturnsEmptyListNotNull(t *testing.T) {
	svc, users, _ := newService(t)

	data, err := svc.Export(context.Background(), users.record.User)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if data.Categories == nil {
		t.Error("categories must be an empty slice so it serialises as []")
	}
}
