// Package password hashes and validates user passwords.
package password

import (
	"sync"

	"golang.org/x/crypto/bcrypt"

	"github.com/schellhas/idletime/backend/internal/validation"
)

const (
	// MinLength is the minimum accepted password length in bytes.
	MinLength = 10
	// maxLength is bcrypt's hard input limit; longer passwords would be silently truncated.
	maxLength = 72
	cost      = 12
)

// Validate checks the password against the password policy.
func Validate(password string) error {
	switch {
	case len(password) < MinLength:
		return validation.Errorf("password must be at least %d characters", MinLength)
	case len(password) > maxLength:
		return validation.Errorf("password must be at most %d bytes", maxLength)
	}
	return nil
}

// Hash returns a bcrypt hash of password.
func Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Matches reports whether password corresponds to hash.
func Matches(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

var dummyHash = sync.OnceValue(func() string {
	hash, err := Hash("dummy-password-for-timing")
	if err != nil {
		panic(err) // only fails for invalid cost or oversized input, neither is possible here
	}
	return hash
})

// SpendHashingTime burns the same time as a real comparison. Call it when no user was
// found so that response timing does not reveal which accounts exist.
func SpendHashingTime(password string) {
	Matches(dummyHash(), password)
}
