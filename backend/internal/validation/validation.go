// Package validation provides the shared error for rejected user input.
package validation

import (
	"errors"
	"fmt"
)

// ErrInvalid marks an error caused by invalid user input. Handlers map it to HTTP 400.
var ErrInvalid = errors.New("invalid input")

// Errorf returns an error that wraps ErrInvalid with a message safe to show to the user.
func Errorf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}
