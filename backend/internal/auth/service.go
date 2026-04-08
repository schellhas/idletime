package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrDuplicateUser      = errors.New("username or email already in use")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrAlreadyVerified    = errors.New("email already verified")
	ErrUnauthenticated    = errors.New("unauthenticated")
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

const (
	defaultRootCategoryName       = "root"
	legacyDefaultRootCategoryName = "none"
)

type Mailer interface {
	SendVerificationEmail(ctx context.Context, toEmail, username, verificationURL string) error
}

type Service struct {
	db              *pgxpool.Pool
	mailer          Mailer
	frontendURL     string
	sessionTTL      time.Duration
	verificationTTL time.Duration
}

type User struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

type RegisterResult struct {
	User            User
	VerificationURL string
}

type LoginInput struct {
	Identifier string
	Password   string
	UserAgent  string
	RemoteAddr string
}

type LoginResult struct {
	User         User
	SessionToken string
	ExpiresAt    time.Time
}

type userRecord struct {
	ID              int64
	Username        string
	Email           string
	PasswordHash    string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
}

func NewService(db *pgxpool.Pool, mailer Mailer, frontendURL string, sessionTTL time.Duration) *Service {
	if sessionTTL <= 0 {
		sessionTTL = 7 * 24 * time.Hour
	}

	return &Service{
		db:              db,
		mailer:          mailer,
		frontendURL:     strings.TrimRight(frontendURL, "/"),
		sessionTTL:      sessionTTL,
		verificationTTL: 24 * time.Hour,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (RegisterResult, error) {
	username := strings.TrimSpace(input.Username)
	email := strings.ToLower(strings.TrimSpace(input.Email))

	if err := validateRegistration(username, email, input.Password); err != nil {
		return RegisterResult{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return RegisterResult{}, fmt.Errorf("hash password: %w", err)
	}

	var rec userRecord
	err = s.db.QueryRow(
		ctx,
		`INSERT INTO users (username, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, username, email, password_hash, email_verified_at, created_at`,
		username,
		email,
		string(passwordHash),
	).Scan(
		&rec.ID,
		&rec.Username,
		&rec.Email,
		&rec.PasswordHash,
		&rec.EmailVerifiedAt,
		&rec.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return RegisterResult{}, ErrDuplicateUser
		}
		return RegisterResult{}, fmt.Errorf("create user: %w", err)
	}

	if err := s.ensureDefaultRootCategory(ctx, rec.ID); err != nil {
		return RegisterResult{}, err
	}

	verificationURL, err := s.issueVerificationToken(ctx, rec.ID, rec.Username, rec.Email)
	if err != nil {
		return RegisterResult{}, err
	}

	return RegisterResult{
		User:            rec.public(),
		VerificationURL: verificationURL,
	}, nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string) (User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return User{}, fmt.Errorf("%w: token is required", ErrInvalidInput)
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return User{}, fmt.Errorf("begin verify transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var tokenID int64
	var rec userRecord
	var expiresAt time.Time
	var usedAt *time.Time

	err = tx.QueryRow(
		ctx,
		`SELECT evt.id, u.id, u.username, u.email, u.password_hash, u.email_verified_at, u.created_at, evt.expires_at, evt.used_at
		 FROM email_verification_tokens evt
		 JOIN users u ON u.id = evt.user_id
		 WHERE evt.token_hash = $1`,
		hashToken(token),
	).Scan(
		&tokenID,
		&rec.ID,
		&rec.Username,
		&rec.Email,
		&rec.PasswordHash,
		&rec.EmailVerifiedAt,
		&rec.CreatedAt,
		&expiresAt,
		&usedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidToken
	}
	if err != nil {
		return User{}, fmt.Errorf("load verification token: %w", err)
	}

	now := time.Now().UTC()
	if usedAt != nil || now.After(expiresAt) {
		return User{}, ErrInvalidToken
	}

	if rec.EmailVerifiedAt == nil {
		if _, err := tx.Exec(
			ctx,
			`UPDATE users SET email_verified_at = NOW(), updated_at = NOW() WHERE id = $1`,
			rec.ID,
		); err != nil {
			return User{}, fmt.Errorf("mark email verified: %w", err)
		}
		rec.EmailVerifiedAt = &now
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE email_verification_tokens SET used_at = NOW() WHERE id = $1`,
		tokenID,
	); err != nil {
		return User{}, fmt.Errorf("mark verification token used: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit verify transaction: %w", err)
	}

	return rec.public(), nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	identifier := strings.TrimSpace(input.Identifier)
	if identifier == "" || input.Password == "" {
		return LoginResult{}, fmt.Errorf("%w: identifier and password are required", ErrInvalidInput)
	}

	rec, err := s.findUserByIdentifier(ctx, identifier)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("load user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(rec.PasswordHash), []byte(input.Password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	if rec.EmailVerifiedAt == nil {
		return LoginResult{}, ErrEmailNotVerified
	}

	sessionToken, expiresAt, err := s.createSession(ctx, rec.ID, input.UserAgent, input.RemoteAddr)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		User:         rec.public(),
		SessionToken: sessionToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	sessionToken = strings.TrimSpace(sessionToken)
	if sessionToken == "" {
		return ErrUnauthenticated
	}

	_, err := s.db.Exec(
		ctx,
		`UPDATE sessions SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`,
		hashToken(sessionToken),
	)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (s *Service) CurrentUser(ctx context.Context, sessionToken string) (User, error) {
	sessionToken = strings.TrimSpace(sessionToken)
	if sessionToken == "" {
		return User{}, ErrUnauthenticated
	}

	var rec userRecord
	err := s.db.QueryRow(
		ctx,
		`SELECT u.id, u.username, u.email, u.password_hash, u.email_verified_at, u.created_at
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = $1
		   AND s.revoked_at IS NULL
		   AND s.expires_at > NOW()
		 LIMIT 1`,
		hashToken(sessionToken),
	).Scan(
		&rec.ID,
		&rec.Username,
		&rec.Email,
		&rec.PasswordHash,
		&rec.EmailVerifiedAt,
		&rec.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUnauthenticated
	}
	if err != nil {
		return User{}, fmt.Errorf("load current user: %w", err)
	}

	return rec.public(), nil
}

func (s *Service) ResendVerification(ctx context.Context, identifier string) (string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", fmt.Errorf("%w: identifier is required", ErrInvalidInput)
	}

	rec, err := s.findUserByIdentifier(ctx, identifier)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUnauthenticated
	}
	if err != nil {
		return "", fmt.Errorf("load user for resend verification: %w", err)
	}

	if rec.EmailVerifiedAt != nil {
		return "", ErrAlreadyVerified
	}

	return s.issueVerificationToken(ctx, rec.ID, rec.Username, rec.Email)
}

func (s *Service) findUserByIdentifier(ctx context.Context, identifier string) (userRecord, error) {
	var rec userRecord
	err := s.db.QueryRow(
		ctx,
		`SELECT id, username, email, password_hash, email_verified_at, created_at
		 FROM users
		 WHERE LOWER(email) = LOWER($1) OR LOWER(username) = LOWER($1)
		 LIMIT 1`,
		identifier,
	).Scan(
		&rec.ID,
		&rec.Username,
		&rec.Email,
		&rec.PasswordHash,
		&rec.EmailVerifiedAt,
		&rec.CreatedAt,
	)
	return rec, err
}

func (s *Service) ensureDefaultRootCategory(ctx context.Context, userID int64) error {
	if _, err := s.db.Exec(
		ctx,
		`UPDATE categories
		 SET name = $2,
		     parent_id = NULL
		 WHERE user_id = $1
		   AND LOWER(name) = LOWER($3)
		   AND NOT EXISTS (
		     SELECT 1 FROM categories existing
		     WHERE existing.user_id = $1 AND LOWER(existing.name) = LOWER($2)
		   )`,
		userID,
		defaultRootCategoryName,
		legacyDefaultRootCategoryName,
	); err != nil {
		return fmt.Errorf("rename legacy default category: %w", err)
	}

	if _, err := s.db.Exec(
		ctx,
		`INSERT INTO categories (user_id, parent_id, name, multiplier)
		 VALUES ($1, NULL, $2, 1.0)
		 ON CONFLICT (user_id, name) DO NOTHING`,
		userID,
		defaultRootCategoryName,
	); err != nil {
		return fmt.Errorf("create default category: %w", err)
	}

	if _, err := s.db.Exec(
		ctx,
		`UPDATE categories AS child
		 SET parent_id = root.id
		 FROM categories AS root
		 WHERE child.user_id = $1
		   AND root.user_id = $1
		   AND LOWER(root.name) = LOWER($2)
		   AND child.id <> root.id
		   AND child.parent_id IS NULL`,
		userID,
		defaultRootCategoryName,
	); err != nil {
		return fmt.Errorf("attach categories to root: %w", err)
	}

	return nil
}

func (s *Service) issueVerificationToken(ctx context.Context, userID int64, username, email string) (string, error) {
	rawToken, err := generateToken(32)
	if err != nil {
		return "", fmt.Errorf("generate verification token: %w", err)
	}

	if _, err := s.db.Exec(
		ctx,
		`UPDATE email_verification_tokens
		 SET used_at = NOW()
		 WHERE user_id = $1 AND used_at IS NULL`,
		userID,
	); err != nil {
		return "", fmt.Errorf("expire existing verification tokens: %w", err)
	}

	if _, err := s.db.Exec(
		ctx,
		`INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		userID,
		hashToken(rawToken),
		time.Now().UTC().Add(s.verificationTTL),
	); err != nil {
		return "", fmt.Errorf("store verification token: %w", err)
	}

	verificationURL := s.buildVerificationURL(rawToken)
	if s.mailer != nil {
		if err := s.mailer.SendVerificationEmail(ctx, email, username, verificationURL); err != nil {
			return "", fmt.Errorf("send verification email: %w", err)
		}
	}

	return verificationURL, nil
}

func (s *Service) createSession(ctx context.Context, userID int64, userAgent, remoteAddr string) (string, time.Time, error) {
	rawToken, err := generateToken(32)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate session token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(s.sessionTTL)
	_, err = s.db.Exec(
		ctx,
		`INSERT INTO sessions (user_id, token_hash, expires_at, user_agent, ip_address)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID,
		hashToken(rawToken),
		expiresAt,
		trimTo(userAgent, 512),
		trimTo(remoteAddr, 128),
	)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create session: %w", err)
	}

	return rawToken, expiresAt, nil
}

func (s *Service) buildVerificationURL(rawToken string) string {
	base := s.frontendURL
	if base == "" {
		base = "http://localhost:3000"
	}

	return base + "/verify-email?token=" + url.QueryEscape(rawToken)
}

func validateRegistration(username, email, password string) error {
	switch {
	case len(username) < 3 || len(username) > 32:
		return fmt.Errorf("%w: username must be between 3 and 32 characters", ErrInvalidInput)
	case !emailPattern.MatchString(email):
		return fmt.Errorf("%w: email address is not valid", ErrInvalidInput)
	case len(password) < 8:
		return fmt.Errorf("%w: password must be at least 8 characters", ErrInvalidInput)
	default:
		return nil
	}
}

func generateToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func trimTo(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		return value[:max]
	}
	return value
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (u userRecord) public() User {
	return User{
		ID:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		EmailVerified: u.EmailVerifiedAt != nil,
		CreatedAt:     u.CreatedAt,
	}
}
