// Package timer implements the stopwatch: one running timer per user, stored on the
// server so it survives reloads and can be stopped from another device.
package timer

import (
	"errors"
	"time"
)

// Timer is a running stopwatch.
type Timer struct {
	ActivityID int64     `json:"activityId"`
	StartedAt  time.Time `json:"startedAt"`
}

// Domain errors returned by the service and repository.
var (
	ErrAlreadyRunning   = errors.New("a timer is already running")
	ErrNotRunning       = errors.New("no timer is running")
	ErrActivityNotFound = errors.New("activity not found")
)
