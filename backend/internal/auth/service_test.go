package auth

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/schellhas/idletime/backend/internal/mail"
	"github.com/schellhas/idletime/backend/internal/user"
	"github.com/schellhas/idletime/backend/internal/validation"
)

// --- in-memory fakes -------------------------------------------------------------

type memUsers struct {
	records map[int64]*user.Record
	nextID  int64
}

func newMemUsers() *memUsers { return &memUsers{records: map[int64]*user.Record{}, nextID: 1} }

func (m *memUsers) Create(_ context.Context, username, email, hash string) (user.User, error) {
	for _, r := range m.records {
		if strings.EqualFold(r.Email, email) {
			return user.User{}, user.ErrDuplicateEmail
		}
		if strings.EqualFold(r.Username, username) {
			return user.User{}, user.ErrDuplicateUsername
		}
	}
	u := user.User{ID: m.nextID, Username: username, Email: email}
	m.records[u.ID] = &user.Record{User: u, PasswordHash: hash}
	m.nextID++
	return u, nil
}

func (m *memUsers) find(match func(*user.Record) bool) (user.Record, error) {
	for _, r := range m.records {
		if match(r) {
			return *r, nil
		}
	}
	return user.Record{}, user.ErrNotFound
}

func (m *memUsers) RecordByIdentifier(_ context.Context, id string) (user.Record, error) {
	return m.find(func(r *user.Record) bool { return strings.EqualFold(r.Email, id) || strings.EqualFold(r.Username, id) })
}

func (m *memUsers) RecordByEmail(_ context.Context, email string) (user.Record, error) {
	return m.find(func(r *user.Record) bool { return r.Email == email })
}

func (m *memUsers) MarkVerified(_ context.Context, id int64) error {
	m.records[id].EmailVerified = true
	return nil
}

func (m *memUsers) UpdatePassword(_ context.Context, id int64, hash string) error {
	m.records[id].PasswordHash = hash
	return nil
}

type memSessions struct {
	users    *memUsers
	sessions map[string]int64 // token hash -> user id
}

func (m *memSessions) Create(_ context.Context, s NewSession) error {
	m.sessions[s.TokenHash] = s.UserID
	return nil
}

func (m *memSessions) Lookup(_ context.Context, hash string) (user.User, error) {
	id, ok := m.sessions[hash]
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	return m.users.records[id].User, nil
}

func (m *memSessions) Revoke(_ context.Context, hash string) error {
	delete(m.sessions, hash)
	return nil
}

func (m *memSessions) RevokeAll(_ context.Context, userID int64, except string) error {
	for hash, id := range m.sessions {
		if id == userID && hash != except {
			delete(m.sessions, hash)
		}
	}
	return nil
}

type memTokens struct {
	tokens map[string]struct { // purpose+hash
		userID  int64
		expires time.Time
		used    bool
	}
}

func (m *memTokens) Issue(_ context.Context, userID int64, purpose, hash string, expires time.Time) error {
	m.tokens[purpose+hash] = struct {
		userID  int64
		expires time.Time
		used    bool
	}{userID: userID, expires: expires}
	return nil
}

func (m *memTokens) Consume(_ context.Context, purpose, hash string) (int64, error) {
	t, ok := m.tokens[purpose+hash]
	if !ok || t.used || time.Now().After(t.expires) {
		return 0, ErrInvalidToken
	}
	t.used = true
	m.tokens[purpose+hash] = t
	return t.userID, nil
}

type recordingMailer struct{ sent []mail.Message }

func (r *recordingMailer) Send(_ context.Context, m mail.Message) error {
	r.sent = append(r.sent, m)
	return nil
}

// --- helpers ---------------------------------------------------------------------

const testPassword = "correct horse battery"

type fixture struct {
	service  *Service
	users    *memUsers
	sessions *memSessions
	mailer   *recordingMailer
}

func newFixture() *fixture {
	users := newMemUsers()
	sessions := &memSessions{users: users, sessions: map[string]int64{}}
	tokens := &memTokens{tokens: map[string]struct {
		userID  int64
		expires time.Time
		used    bool
	}{}}
	mailer := &recordingMailer{}
	return &fixture{
		service:  NewService(users, sessions, tokens, mailer, "https://app.example.com/"),
		users:    users,
		sessions: sessions,
		mailer:   mailer,
	}
}

var tokenInBody = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

func (f *fixture) lastToken(t *testing.T) string {
	t.Helper()
	if len(f.mailer.sent) == 0 {
		t.Fatal("no email was sent")
	}
	match := tokenInBody.FindStringSubmatch(f.mailer.sent[len(f.mailer.sent)-1].Body)
	if match == nil {
		t.Fatal("email contains no token link")
	}
	return match[1]
}

func (f *fixture) registerAndVerify(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if err := f.service.Register(ctx, RegisterInput{Username: "alice", Email: "Alice@Example.com", Password: testPassword}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := f.service.VerifyEmail(ctx, f.lastToken(t)); err != nil {
		t.Fatalf("VerifyEmail() error = %v", err)
	}
}

// --- tests -----------------------------------------------------------------------

func TestRegister(t *testing.T) {
	ctx := context.Background()

	t.Run("creates an unverified user and sends the verification link", func(t *testing.T) {
		f := newFixture()
		if err := f.service.Register(ctx, RegisterInput{Username: "alice", Email: "Alice@Example.com", Password: testPassword}); err != nil {
			t.Fatalf("error = %v", err)
		}
		rec, err := f.users.RecordByEmail(ctx, "alice@example.com")
		if err != nil || rec.EmailVerified {
			t.Fatalf("expected a stored, unverified user; err=%v verified=%v", err, rec.EmailVerified)
		}
		if rec.PasswordHash == testPassword {
			t.Error("password must be stored hashed")
		}
		if len(f.mailer.sent) != 1 || !strings.HasPrefix(f.mailer.sent[0].Body[strings.Index(f.mailer.sent[0].Body, "https://"):], "https://app.example.com/verify-email?token=") {
			t.Errorf("unexpected emails: %+v", f.mailer.sent)
		}
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		for name, in := range map[string]RegisterInput{
			"username": {Username: "a", Email: "a@example.com", Password: testPassword},
			"email":    {Username: "alice", Email: "nope", Password: testPassword},
			"password": {Username: "alice", Email: "a@example.com", Password: "short"},
		} {
			if err := newFixture().service.Register(ctx, in); !errors.Is(err, validation.ErrInvalid) {
				t.Errorf("%s: error = %v, want validation error", name, err)
			}
		}
	})

	t.Run("reports a taken username", func(t *testing.T) {
		f := newFixture()
		f.registerAndVerify(t)
		err := f.service.Register(ctx, RegisterInput{Username: "ALICE", Email: "other@example.com", Password: testPassword})
		if !errors.Is(err, user.ErrDuplicateUsername) {
			t.Errorf("error = %v, want ErrDuplicateUsername", err)
		}
	})

	t.Run("does not reveal a registered email", func(t *testing.T) {
		f := newFixture()
		f.registerAndVerify(t)
		f.mailer.sent = nil

		err := f.service.Register(ctx, RegisterInput{Username: "mallory", Email: "alice@example.com", Password: testPassword})

		if err != nil {
			t.Fatalf("error = %v, want silent success", err)
		}
		if len(f.mailer.sent) != 1 || f.mailer.sent[0].To != "alice@example.com" || f.mailer.sent[0].Subject != mail.AccountExists("", "", "").Subject {
			t.Errorf("expected an account-exists notice to the owner, got %+v", f.mailer.sent)
		}
		if len(f.users.records) != 1 {
			t.Error("no second account may be created")
		}
	})
}

func TestVerifyEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects unknown tokens", func(t *testing.T) {
		if err := newFixture().service.VerifyEmail(ctx, "bogus"); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("tokens work only once", func(t *testing.T) {
		f := newFixture()
		f.registerAndVerify(t)
		if err := f.service.VerifyEmail(ctx, f.lastToken(t)); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("second use: error = %v, want ErrInvalidToken", err)
		}
	})
}

func TestLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("succeeds with email or username and opens a session", func(t *testing.T) {
		f := newFixture()
		f.registerAndVerify(t)

		for _, identifier := range []string{"alice", "ALICE@example.com"} {
			session, err := f.service.Login(ctx, LoginInput{Identifier: identifier, Password: testPassword})
			if err != nil {
				t.Fatalf("Login(%q) error = %v", identifier, err)
			}
			u, err := f.service.Authenticate(ctx, session.Token)
			if err != nil || u.Username != "alice" {
				t.Fatalf("Authenticate() = %+v, %v", u, err)
			}
		}
	})

	t.Run("rejects wrong passwords and unknown users identically", func(t *testing.T) {
		f := newFixture()
		f.registerAndVerify(t)
		for _, in := range []LoginInput{
			{Identifier: "alice", Password: "wrong password!"},
			{Identifier: "nobody", Password: testPassword},
		} {
			if _, err := f.service.Login(ctx, in); !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("Login(%q) error = %v, want ErrInvalidCredentials", in.Identifier, err)
			}
		}
	})

	t.Run("requires a verified email", func(t *testing.T) {
		f := newFixture()
		_ = f.service.Register(ctx, RegisterInput{Username: "alice", Email: "alice@example.com", Password: testPassword})
		if _, err := f.service.Login(ctx, LoginInput{Identifier: "alice", Password: testPassword}); !errors.Is(err, ErrEmailNotVerified) {
			t.Errorf("error = %v, want ErrEmailNotVerified", err)
		}
	})
}

func TestLogoutEndsTheSession(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	f.registerAndVerify(t)
	session, _ := f.service.Login(ctx, LoginInput{Identifier: "alice", Password: testPassword})

	if err := f.service.Logout(ctx, session.Token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := f.service.Authenticate(ctx, session.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("error = %v, want ErrUnauthenticated", err)
	}
}

func TestPasswordReset(t *testing.T) {
	ctx := context.Background()

	t.Run("is silent for unknown addresses", func(t *testing.T) {
		f := newFixture()
		if err := f.service.ForgotPassword(ctx, "nobody@example.com"); err != nil || len(f.mailer.sent) != 0 {
			t.Errorf("err = %v, emails = %d; want no error and no email", err, len(f.mailer.sent))
		}
	})

	t.Run("sets the new password and ends all sessions", func(t *testing.T) {
		f := newFixture()
		f.registerAndVerify(t)
		old, _ := f.service.Login(ctx, LoginInput{Identifier: "alice", Password: testPassword})

		if err := f.service.ForgotPassword(ctx, "alice@example.com"); err != nil {
			t.Fatalf("ForgotPassword() error = %v", err)
		}
		if err := f.service.ResetPassword(ctx, f.lastToken(t), "a brand new password"); err != nil {
			t.Fatalf("ResetPassword() error = %v", err)
		}

		if _, err := f.service.Authenticate(ctx, old.Token); !errors.Is(err, ErrUnauthenticated) {
			t.Error("existing sessions must be revoked")
		}
		if _, err := f.service.Login(ctx, LoginInput{Identifier: "alice", Password: testPassword}); !errors.Is(err, ErrInvalidCredentials) {
			t.Error("the old password must stop working")
		}
		if _, err := f.service.Login(ctx, LoginInput{Identifier: "alice", Password: "a brand new password"}); err != nil {
			t.Errorf("the new password must work: %v", err)
		}
	})

	t.Run("a weak password does not use up the token", func(t *testing.T) {
		f := newFixture()
		f.registerAndVerify(t)
		_ = f.service.ForgotPassword(ctx, "alice@example.com")
		resetToken := f.lastToken(t)

		if err := f.service.ResetPassword(ctx, resetToken, "short"); !errors.Is(err, validation.ErrInvalid) {
			t.Fatalf("error = %v, want validation error", err)
		}
		if err := f.service.ResetPassword(ctx, resetToken, "a brand new password"); err != nil {
			t.Errorf("token should still be valid: %v", err)
		}
	})
}
