package user

import (
	"net/mail"
	"regexp"
	"strings"

	"github.com/schellhas/idletime/backend/internal/validation"
)

const maxEmailLength = 254

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{3,32}$`)

// NormalizeUsername trims surrounding whitespace.
func NormalizeUsername(username string) string {
	return strings.TrimSpace(username)
}

// ValidateUsername checks a normalized username.
func ValidateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return validation.Errorf("username must be 3-32 characters: letters, digits, '.', '_' or '-'")
	}
	return nil
}

// NormalizeEmail trims whitespace and lower-cases the address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail checks a normalized email address.
func ValidateEmail(email string) error {
	if len(email) > maxEmailLength {
		return validation.Errorf("email address is too long")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return validation.Errorf("email address is invalid")
	}
	return nil
}
