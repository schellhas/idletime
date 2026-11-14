// Package auth implements registration, email verification, login sessions and
// password reset.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/schellhas/idletime/backend/internal/mail"
	"github.com/schellhas/idletime/backend/internal/password"
	"github.com/schellhas/idletime/backend/internal/token"
	"github.com/schellhas/idletime/backend/internal/user"
)

const (
	sessionTTL      = 14 * 24 * time.Hour
	verificationTTL = 24 * time.Hour
	resetTTL        = time.Hour
)

// Token purposes stored in user_tokens.
const (
	PurposeEmailVerification = "email_verification"
	PurposePasswordReset     = "password_reset"
)

// Errors returned by the Service.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUnauthenticated    = errors.New("unauthenticated")
)

// UserStore is the user persistence required by the Service.
type UserStore interface {
	Create(ctx context.Context, username, email, passwordHash string) (user.User, error)
	RecordByIdentifier(ctx context.Context, identifier string) (user.Record, error)
	RecordByEmail(ctx context.Context, email string) (user.Record, error)
	MarkVerified(ctx context.Context, id int64) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
}

// NewSession describes a session to persist.
type NewSession struct {
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	UserAgent string
	IPAddress string
}

// SessionStore persists login sessions.
type SessionStore interface {
	Create(ctx context.Context, s NewSession) error
	Lookup(ctx context.Context, tokenHash string) (user.User, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAll(ctx context.Context, userID int64, exceptTokenHash string) error
}

// TokenStore persists one-time email tokens.
type TokenStore interface {
	// Issue stores a new token, replacing unused tokens of the same user and purpose.
	Issue(ctx context.Context, userID int64, purpose, tokenHash string, expiresAt time.Time) error
	// Consume marks a valid token as used and returns its owner, or ErrInvalidToken.
	Consume(ctx context.Context, purpose, tokenHash string) (userID int64, err error)
}

// Service contains the authentication use cases.
type Service struct {
	users    UserStore
	sessions SessionStore
	tokens   TokenStore
	mailer   mail.Sender
	baseURL  string
}

// NewService creates a Service. baseURL is the public address of the web app and is
// used to build the links in emails.
func NewService(users UserStore, sessions SessionStore, tokens TokenStore, mailer mail.Sender, baseURL string) *Service {
	return &Service{users: users, sessions: sessions, tokens: tokens, mailer: mailer, baseURL: strings.TrimRight(baseURL, "/")}
}

// RegisterInput carries the fields of a registration.
type RegisterInput struct {
	Username string
	Email    string
	Password string
}

// Register creates an unverified account and sends the verification email.
// It succeeds without revealing whether the email address is already registered;
// in that case the address owner is notified by email instead.
func (s *Service) Register(ctx context.Context, in RegisterInput) error {
	username := user.NormalizeUsername(in.Username)
	email := user.NormalizeEmail(in.Email)
	if err := user.ValidateUsername(username); err != nil {
		return err
	}
	if err := user.ValidateEmail(email); err != nil {
		return err
	}
	if err := password.Validate(in.Password); err != nil {
		return err
	}

	hash, err := password.Hash(in.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	created, err := s.users.Create(ctx, username, email, hash)
	switch {
	case errors.Is(err, user.ErrDuplicateEmail):
		return s.notifyAccountExists(ctx, email)
	case err != nil:
		return fmt.Errorf("create user: %w", err)
	}
	return s.sendVerification(ctx, created)
}

// VerifyEmail confirms an email address using the token from the verification link.
func (s *Service) VerifyEmail(ctx context.Context, rawToken string) error {
	userID, err := s.tokens.Consume(ctx, PurposeEmailVerification, token.Hash(rawToken))
	if err != nil {
		return err
	}
	if err := s.users.MarkVerified(ctx, userID); err != nil {
		return fmt.Errorf("mark verified: %w", err)
	}
	return nil
}

// ResendVerification sends a new verification email. It does nothing, silently, for
// unknown or already verified addresses.
func (s *Service) ResendVerification(ctx context.Context, email string) error {
	rec, err := s.users.RecordByEmail(ctx, user.NormalizeEmail(email))
	if errors.Is(err, user.ErrNotFound) || (err == nil && rec.EmailVerified) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	return s.sendVerification(ctx, rec.User)
}

// LoginInput carries the credentials and request metadata of a login attempt.
type LoginInput struct {
	Identifier string // username or email
	Password   string
	UserAgent  string
	IPAddress  string
}

// Session is an established login. Token is the raw value for the client's cookie.
type Session struct {
	Token     string
	ExpiresAt time.Time
	User      user.User
}

// Login checks the credentials and opens a session. Unknown users and wrong passwords
// are indistinguishable to the caller.
func (s *Service) Login(ctx context.Context, in LoginInput) (Session, error) {
	rec, err := s.users.RecordByIdentifier(ctx, strings.TrimSpace(in.Identifier))
	if errors.Is(err, user.ErrNotFound) {
		password.SpendHashingTime(in.Password)
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, fmt.Errorf("find user: %w", err)
	}
	if !password.Matches(rec.PasswordHash, in.Password) {
		return Session{}, ErrInvalidCredentials
	}
	if !rec.EmailVerified {
		return Session{}, ErrEmailNotVerified
	}

	raw, hash, err := token.New()
	if err != nil {
		return Session{}, err
	}
	expiresAt := time.Now().Add(sessionTTL)
	err = s.sessions.Create(ctx, NewSession{
		UserID: rec.ID, TokenHash: hash, ExpiresAt: expiresAt, UserAgent: in.UserAgent, IPAddress: in.IPAddress,
	})
	if err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	return Session{Token: raw, ExpiresAt: expiresAt, User: rec.User}, nil
}

// Logout ends the session identified by the raw cookie token.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if err := s.sessions.Revoke(ctx, token.Hash(rawToken)); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// Authenticate resolves the user for a raw session token, or returns ErrUnauthenticated.
func (s *Service) Authenticate(ctx context.Context, rawToken string) (user.User, error) {
	u, err := s.sessions.Lookup(ctx, token.Hash(rawToken))
	if errors.Is(err, user.ErrNotFound) {
		return user.User{}, ErrUnauthenticated
	}
	if err != nil {
		return user.User{}, fmt.Errorf("lookup session: %w", err)
	}
	return u, nil
}

// ForgotPassword sends a reset link. It does nothing, silently, for unknown addresses.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	rec, err := s.users.RecordByEmail(ctx, user.NormalizeEmail(email))
	if errors.Is(err, user.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	link, err := s.issueLink(ctx, rec.ID, PurposePasswordReset, "/reset-password", resetTTL)
	if err != nil {
		return err
	}
	return s.mailer.Send(ctx, mail.PasswordReset(rec.Email, rec.Username, link))
}

// ResetPassword sets a new password using a reset token and ends all sessions of the user.
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if err := password.Validate(newPassword); err != nil {
		return err // checked first so a typo does not use up the token
	}
	userID, err := s.tokens.Consume(ctx, PurposePasswordReset, token.Hash(rawToken))
	if err != nil {
		return err
	}
	hash, err := password.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.users.UpdatePassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if err := s.sessions.RevokeAll(ctx, userID, ""); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}
	return nil
}

func (s *Service) sendVerification(ctx context.Context, u user.User) error {
	link, err := s.issueLink(ctx, u.ID, PurposeEmailVerification, "/verify-email", verificationTTL)
	if err != nil {
		return err
	}
	return s.mailer.Send(ctx, mail.VerifyEmail(u.Email, u.Username, link))
}

func (s *Service) notifyAccountExists(ctx context.Context, email string) error {
	return s.mailer.Send(ctx, mail.AccountExists(email, s.baseURL+"/login", s.baseURL+"/forgot-password"))
}

// issueLink stores a fresh token and returns the web app link that carries it.
func (s *Service) issueLink(ctx context.Context, userID int64, purpose, path string, ttl time.Duration) (string, error) {
	raw, hash, err := token.New()
	if err != nil {
		return "", err
	}
	if err := s.tokens.Issue(ctx, userID, purpose, hash, time.Now().Add(ttl)); err != nil {
		return "", fmt.Errorf("store token: %w", err)
	}
	return s.baseURL + path + "?token=" + url.QueryEscape(raw), nil
}
