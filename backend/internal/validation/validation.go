// Package validation provides the shared error for rejected user input and the
// field rules used by several features.
package validation

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// MaxNameLength is the longest allowed name of a category or activity.
	MaxNameLength = 100
	// MaxMultiplier bounds multipliers to a sane range.
	MaxMultiplier = 1000.0
)

// ErrInvalid marks an error caused by invalid user input. Handlers map it to HTTP 400.
var ErrInvalid = errors.New("invalid input")

// Errorf returns an error that wraps ErrInvalid with a message safe to show to the user.
func Errorf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// Name trims a category or activity name and checks its length.
func Name(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > MaxNameLength {
		return "", Errorf("name must be 1-%d characters", MaxNameLength)
	}
	return name, nil
}

// Multiplier checks that a weighting factor is positive and not absurdly large.
func Multiplier(m float64) error {
	if !(m > 0 && m <= MaxMultiplier) { // written this way so NaN is rejected too
		return Errorf("multiplier must be greater than 0 and at most %g", MaxMultiplier)
	}
	return nil
}
