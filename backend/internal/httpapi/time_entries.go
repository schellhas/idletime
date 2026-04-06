package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"idletime/backend/internal/auth"
	"idletime/backend/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TimeEntryHandler struct {
	cfg         config.Config
	db          *pgxpool.Pool
	authService *auth.Service
}

type TimeEntry struct {
	ID         int64     `json:"id"`
	ActivityID int64     `json:"activity_id"`
	Minutes    int       `json:"minutes"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewTimeEntryHandler(cfg config.Config, db *pgxpool.Pool, authService *auth.Service) *TimeEntryHandler {
	return &TimeEntryHandler{cfg: cfg, db: db, authService: authService}
}

func (h *TimeEntryHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/time-entries", requireAuth(h.cfg, h.authService, http.HandlerFunc(h.handleTimeEntries)))
	mux.Handle("/time-entries/", requireAuth(h.cfg, h.authService, http.HandlerFunc(h.handleTimeEntryByID)))
}

func (h *TimeEntryHandler) handleTimeEntries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListTimeEntries(w, r)
	case http.MethodPost:
		h.handleCreateTimeEntry(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *TimeEntryHandler) handleTimeEntryByID(w http.ResponseWriter, r *http.Request) {
	timeEntryID, err := timeEntryIDFromPath(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetTimeEntry(w, r, timeEntryID)
	case http.MethodPatch, http.MethodPut:
		h.handleUpdateTimeEntry(w, r, timeEntryID)
	case http.MethodDelete:
		h.handleDeleteTimeEntry(w, r, timeEntryID)
	default:
		w.Header().Set("Allow", "GET, PATCH, PUT, DELETE")
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *TimeEntryHandler) handleListTimeEntries(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	var (
		rows pgx.Rows
		err  error
	)
	activityIDValue := strings.TrimSpace(r.URL.Query().Get("activity_id"))
	if activityIDValue == "" {
		rows, err = h.db.Query(
			r.Context(),
			`SELECT te.id, te.activity_id, te.minutes, COALESCE(te.note, ''), te.created_at
			 FROM time_entries te
			 JOIN activities a ON a.id = te.activity_id
			 JOIN categories c ON c.id = a.category_id
			 WHERE c.user_id = $1
			 ORDER BY te.created_at ASC, te.id ASC`,
			user.ID,
		)
	} else {
		activityID, parseErr := strconv.ParseInt(activityIDValue, 10, 64)
		if parseErr != nil || activityID <= 0 {
			writeErrorJSON(w, http.StatusBadRequest, "activity_id must be a positive integer")
			return
		}
		rows, err = h.db.Query(
			r.Context(),
			`SELECT te.id, te.activity_id, te.minutes, COALESCE(te.note, ''), te.created_at
			 FROM time_entries te
			 JOIN activities a ON a.id = te.activity_id
			 JOIN categories c ON c.id = a.category_id
			 WHERE c.user_id = $1 AND te.activity_id = $2
			 ORDER BY te.created_at ASC, te.id ASC`,
			user.ID,
			activityID,
		)
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to list time entries")
		return
	}
	defer rows.Close()

	entries := make([]TimeEntry, 0)
	for rows.Next() {
		var entry TimeEntry
		if err := rows.Scan(&entry.ID, &entry.ActivityID, &entry.Minutes, &entry.Note, &entry.CreatedAt); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "failed to read time entries")
			return
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to read time entries")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"time_entries": entries,
	})
}

func (h *TimeEntryHandler) handleGetTimeEntry(w http.ResponseWriter, r *http.Request, timeEntryID int64) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	entry, err := h.findTimeEntry(r, user.ID, timeEntryID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "time entry not found")
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load time entry")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"time_entry": entry,
	})
}

func (h *TimeEntryHandler) handleCreateTimeEntry(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	var request struct {
		ActivityID int64   `json:"activity_id"`
		Minutes    int     `json:"minutes"`
		Note       *string `json:"note"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := validateTimeEntryInput(request.ActivityID, request.Minutes); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.findOwnedActivity(r, user.ID, request.ActivityID); errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "activity not found")
		return
	} else if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load activity")
		return
	}

	note := ""
	if request.Note != nil {
		note = strings.TrimSpace(*request.Note)
	}

	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to create time entry")
		return
	}
	defer func() {
		_ = tx.Rollback(r.Context())
	}()

	var entry TimeEntry
	err = tx.QueryRow(
		r.Context(),
		`INSERT INTO time_entries (activity_id, minutes, note)
		 VALUES ($1, $2, $3)
		 RETURNING id, activity_id, minutes, COALESCE(note, ''), created_at`,
		request.ActivityID,
		request.Minutes,
		note,
	).Scan(&entry.ID, &entry.ActivityID, &entry.Minutes, &entry.Note, &entry.CreatedAt)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to create time entry")
		return
	}

	if _, err := tx.Exec(
		r.Context(),
		`UPDATE activities SET tracked_minutes = tracked_minutes + $2 WHERE id = $1`,
		request.ActivityID,
		request.Minutes,
	); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to update tracked minutes")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to create time entry")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"time_entry": entry,
	})
}

func (h *TimeEntryHandler) handleUpdateTimeEntry(w http.ResponseWriter, r *http.Request, timeEntryID int64) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	current, err := h.findTimeEntry(r, user.ID, timeEntryID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "time entry not found")
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load time entry")
		return
	}

	var request struct {
		ActivityID *int64  `json:"activity_id"`
		Minutes    *int    `json:"minutes"`
		Note       *string `json:"note"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if request.ActivityID == nil && request.Minutes == nil && request.Note == nil {
		writeErrorJSON(w, http.StatusBadRequest, "at least one time entry field must be provided")
		return
	}

	if request.ActivityID != nil {
		current.ActivityID = *request.ActivityID
	}
	if request.Minutes != nil {
		current.Minutes = *request.Minutes
	}
	if request.Note != nil {
		current.Note = strings.TrimSpace(*request.Note)
	}

	if err := validateTimeEntryInput(current.ActivityID, current.Minutes); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.findOwnedActivity(r, user.ID, current.ActivityID); errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "activity not found")
		return
	} else if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load activity")
		return
	}

	original, err := h.findTimeEntry(r, user.ID, timeEntryID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "time entry not found")
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load time entry")
		return
	}

	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to update time entry")
		return
	}
	defer func() {
		_ = tx.Rollback(r.Context())
	}()

	err = tx.QueryRow(
		r.Context(),
		`UPDATE time_entries AS te
		 SET activity_id = $3,
		     minutes = $4,
		     note = $5
		 FROM activities a, categories c
		 WHERE te.id = $1
		   AND te.activity_id = a.id
		   AND a.category_id = c.id
		   AND c.user_id = $2
		 RETURNING te.id, te.activity_id, te.minutes, COALESCE(te.note, ''), te.created_at`,
		timeEntryID,
		user.ID,
		current.ActivityID,
		current.Minutes,
		current.Note,
	).Scan(&current.ID, &current.ActivityID, &current.Minutes, &current.Note, &current.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeErrorJSON(w, http.StatusNotFound, "time entry not found")
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, "failed to update time entry")
		return
	}

	if original.ActivityID == current.ActivityID {
		delta := current.Minutes - original.Minutes
		if _, err := tx.Exec(
			r.Context(),
			`UPDATE activities SET tracked_minutes = GREATEST(tracked_minutes + $2, 0) WHERE id = $1`,
			current.ActivityID,
			delta,
		); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "failed to update tracked minutes")
			return
		}
	} else {
		if _, err := tx.Exec(
			r.Context(),
			`UPDATE activities SET tracked_minutes = GREATEST(tracked_minutes - $2, 0) WHERE id = $1`,
			original.ActivityID,
			original.Minutes,
		); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "failed to update tracked minutes")
			return
		}
		if _, err := tx.Exec(
			r.Context(),
			`UPDATE activities SET tracked_minutes = tracked_minutes + $2 WHERE id = $1`,
			current.ActivityID,
			current.Minutes,
		); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "failed to update tracked minutes")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to update time entry")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"time_entry": current,
	})
}

func (h *TimeEntryHandler) handleDeleteTimeEntry(w http.ResponseWriter, r *http.Request, timeEntryID int64) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	entry, err := h.findTimeEntry(r, user.ID, timeEntryID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "time entry not found")
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load time entry")
		return
	}

	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to delete time entry")
		return
	}
	defer func() {
		_ = tx.Rollback(r.Context())
	}()

	result, err := tx.Exec(
		r.Context(),
		`DELETE FROM time_entries AS te
		 USING activities a, categories c
		 WHERE te.id = $1
		   AND te.activity_id = a.id
		   AND a.category_id = c.id
		   AND c.user_id = $2`,
		timeEntryID,
		user.ID,
	)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to delete time entry")
		return
	}
	if result.RowsAffected() == 0 {
		writeErrorJSON(w, http.StatusNotFound, "time entry not found")
		return
	}

	if _, err := tx.Exec(
		r.Context(),
		`UPDATE activities SET tracked_minutes = GREATEST(tracked_minutes - $2, 0) WHERE id = $1`,
		entry.ActivityID,
		entry.Minutes,
	); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to update tracked minutes")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to delete time entry")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "time entry deleted",
	})
}

func (h *TimeEntryHandler) findOwnedActivity(r *http.Request, userID, activityID int64) (Activity, error) {
	var activity Activity
	err := h.db.QueryRow(
		r.Context(),
		`SELECT a.id, a.category_id, a.name, a.multiplier::double precision, a.minimum_minutes, a.tracked_minutes, a.created_at
		 FROM activities a
		 JOIN categories c ON c.id = a.category_id
		 WHERE a.id = $1 AND c.user_id = $2`,
		activityID,
		userID,
	).Scan(
		&activity.ID,
		&activity.CategoryID,
		&activity.Name,
		&activity.Multiplier,
		&activity.MinimumMinutes,
		&activity.TrackedMinutes,
		&activity.CreatedAt,
	)
	return activity, err
}

func (h *TimeEntryHandler) findTimeEntry(r *http.Request, userID, timeEntryID int64) (TimeEntry, error) {
	var entry TimeEntry
	err := h.db.QueryRow(
		r.Context(),
		`SELECT te.id, te.activity_id, te.minutes, COALESCE(te.note, ''), te.created_at
		 FROM time_entries te
		 JOIN activities a ON a.id = te.activity_id
		 JOIN categories c ON c.id = a.category_id
		 WHERE te.id = $1 AND c.user_id = $2`,
		timeEntryID,
		userID,
	).Scan(&entry.ID, &entry.ActivityID, &entry.Minutes, &entry.Note, &entry.CreatedAt)
	return entry, err
}

func timeEntryIDFromPath(path string) (int64, error) {
	raw := strings.Trim(strings.TrimPrefix(path, "/time-entries/"), "/")
	if raw == "" || strings.Contains(raw, "/") {
		return 0, fmt.Errorf("invalid time entry path")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid time entry id")
	}
	return id, nil
}

func validateTimeEntryInput(activityID int64, minutes int) error {
	switch {
	case activityID <= 0:
		return fmt.Errorf("activity_id is required")
	case minutes <= 0:
		return fmt.Errorf("minutes must be greater than 0")
	default:
		return nil
	}
}
