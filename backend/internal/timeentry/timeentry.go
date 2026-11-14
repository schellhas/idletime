// Package timeentry records how much time a user spent on an activity.
package timeentry

import (
	"errors"
	"time"
)

// Entry is one logged block of time.
type Entry struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"-"`
	ActivityID int64  `json:"activityId"`
	Minutes    int    `json:"minutes"`
	Note       string `json:"note"`
	// LoggedAt is when the activity took place.
	LoggedAt  time.Time `json:"loggedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// DayMinutes is the time logged on one calendar day.
type DayMinutes struct {
	Date    string `json:"date"` // YYYY-MM-DD in the requested time zone
	Minutes int    `json:"minutes"`
}

// Domain errors returned by the service and repository.
var (
	ErrNotFound         = errors.New("time entry not found")
	ErrActivityNotFound = errors.New("activity not found")
)
