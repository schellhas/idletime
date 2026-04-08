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

type ActivityHandler struct {
	cfg         config.Config
	db          *pgxpool.Pool
	authService *auth.Service
}

type Activity struct {
	ID             int64     `json:"id"`
	CategoryID     int64     `json:"category_id"`
	Name           string    `json:"name"`
	Multiplier     float64   `json:"multiplier"`
	MinimumMinutes int       `json:"minimum_minutes"`
	TrackedMinutes int       `json:"tracked_minutes"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewActivityHandler(cfg config.Config, db *pgxpool.Pool, authService *auth.Service) *ActivityHandler {
	return &ActivityHandler{cfg: cfg, db: db, authService: authService}
}

func (h *ActivityHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/activities", requireAuth(h.cfg, h.authService, http.HandlerFunc(h.handleActivities)))
	mux.Handle("/activities/", requireAuth(h.cfg, h.authService, http.HandlerFunc(h.handleActivityByID)))
}

func (h *ActivityHandler) handleActivities(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListActivities(w, r)
	case http.MethodPost:
		h.handleCreateActivity(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ActivityHandler) handleActivityByID(w http.ResponseWriter, r *http.Request) {
	activityID, err := activityIDFromPath(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetActivity(w, r, activityID)
	case http.MethodPatch, http.MethodPut:
		h.handleUpdateActivity(w, r, activityID)
	case http.MethodDelete:
		h.handleDeleteActivity(w, r, activityID)
	default:
		w.Header().Set("Allow", "GET, PATCH, PUT, DELETE")
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ActivityHandler) handleListActivities(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	var (
		rows pgx.Rows
		err  error
	)

	categoryIDValue := strings.TrimSpace(r.URL.Query().Get("category_id"))
	if categoryIDValue == "" {
		rows, err = h.db.Query(
			r.Context(),
			`SELECT a.id, a.category_id, a.name, a.multiplier::double precision, a.minimum_minutes, a.tracked_minutes, a.created_at
			 FROM activities a
			 JOIN categories c ON c.id = a.category_id
			 WHERE c.user_id = $1
			 ORDER BY a.created_at ASC, a.id ASC`,
			user.ID,
		)
	} else {
		categoryID, parseErr := strconv.ParseInt(categoryIDValue, 10, 64)
		if parseErr != nil || categoryID <= 0 {
			writeErrorJSON(w, http.StatusBadRequest, "category_id must be a positive integer")
			return
		}

		rows, err = h.db.Query(
			r.Context(),
			`SELECT a.id, a.category_id, a.name, a.multiplier::double precision, a.minimum_minutes, a.tracked_minutes, a.created_at
			 FROM activities a
			 JOIN categories c ON c.id = a.category_id
			 WHERE c.user_id = $1 AND a.category_id = $2
			 ORDER BY a.created_at ASC, a.id ASC`,
			user.ID,
			categoryID,
		)
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to list activities")
		return
	}
	defer rows.Close()

	activities := make([]Activity, 0)
	for rows.Next() {
		var activity Activity
		if err := rows.Scan(
			&activity.ID,
			&activity.CategoryID,
			&activity.Name,
			&activity.Multiplier,
			&activity.MinimumMinutes,
			&activity.TrackedMinutes,
			&activity.CreatedAt,
		); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "failed to read activities")
			return
		}
		activities = append(activities, activity)
	}
	if err := rows.Err(); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to read activities")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"activities": activities,
	})
}

func (h *ActivityHandler) handleGetActivity(w http.ResponseWriter, r *http.Request, activityID int64) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	activity, err := h.findActivity(r, user.ID, activityID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "activity not found")
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load activity")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"activity": activity,
	})
}

func (h *ActivityHandler) handleCreateActivity(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	var request struct {
		CategoryID     int64    `json:"category_id"`
		Name           string   `json:"name"`
		Multiplier     *float64 `json:"multiplier"`
		MinimumMinutes *int     `json:"minimum_minutes"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	multiplier := 1.0
	if request.Multiplier != nil {
		multiplier = *request.Multiplier
	}
	minimumMinutes := 0
	if request.MinimumMinutes != nil {
		minimumMinutes = *request.MinimumMinutes
	}

	if request.CategoryID <= 0 {
		writeErrorJSON(w, http.StatusBadRequest, "category_id is required")
		return
	}
	if err := validateActivityInput(strings.TrimSpace(request.Name), multiplier, minimumMinutes, true); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.findOwnedCategory(r, user.ID, request.CategoryID); errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "category not found")
		return
	} else if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load category")
		return
	}

	var activity Activity
	err := h.db.QueryRow(
		r.Context(),
		`INSERT INTO activities (category_id, name, multiplier, minimum_minutes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, category_id, name, multiplier::double precision, minimum_minutes, tracked_minutes, created_at`,
		request.CategoryID,
		strings.TrimSpace(request.Name),
		multiplier,
		minimumMinutes,
	).Scan(
		&activity.ID,
		&activity.CategoryID,
		&activity.Name,
		&activity.Multiplier,
		&activity.MinimumMinutes,
		&activity.TrackedMinutes,
		&activity.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			writeErrorJSON(w, http.StatusConflict, "activity name already exists in this category")
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, "failed to create activity")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"activity": activity,
	})
}

func (h *ActivityHandler) handleUpdateActivity(w http.ResponseWriter, r *http.Request, activityID int64) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	current, err := h.findActivity(r, user.ID, activityID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "activity not found")
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load activity")
		return
	}

	var request struct {
		CategoryID     *int64   `json:"category_id"`
		Name           *string  `json:"name"`
		Multiplier     *float64 `json:"multiplier"`
		MinimumMinutes *int     `json:"minimum_minutes"`
		TrackedMinutes *int     `json:"tracked_minutes"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if request.CategoryID == nil && request.Name == nil && request.Multiplier == nil && request.MinimumMinutes == nil && request.TrackedMinutes == nil {
		writeErrorJSON(w, http.StatusBadRequest, "at least one activity field must be provided")
		return
	}

	if request.CategoryID != nil {
		if *request.CategoryID <= 0 {
			writeErrorJSON(w, http.StatusBadRequest, "category_id must be a positive integer")
			return
		}
		if _, err := h.findOwnedCategory(r, user.ID, *request.CategoryID); errors.Is(err, pgx.ErrNoRows) {
			writeErrorJSON(w, http.StatusNotFound, "category not found")
			return
		} else if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "failed to load category")
			return
		}
		current.CategoryID = *request.CategoryID
	}
	if request.Name != nil {
		current.Name = strings.TrimSpace(*request.Name)
	}
	if request.Multiplier != nil {
		current.Multiplier = *request.Multiplier
	}
	if request.MinimumMinutes != nil {
		current.MinimumMinutes = *request.MinimumMinutes
	}
	if request.TrackedMinutes != nil {
		if *request.TrackedMinutes < 0 {
			writeErrorJSON(w, http.StatusBadRequest, "tracked_minutes must be 0 or greater")
			return
		}
		current.TrackedMinutes = *request.TrackedMinutes
	}

	if err := validateActivityInput(current.Name, current.Multiplier, current.MinimumMinutes, true); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.db.QueryRow(
		r.Context(),
		`UPDATE activities AS a
		 SET category_id = $3,
		     name = $4,
		     multiplier = $5,
		     minimum_minutes = $6,
		     tracked_minutes = $7
		 FROM categories current_category, categories new_category
		 WHERE a.id = $1
		   AND a.category_id = current_category.id
		   AND current_category.user_id = $2
		   AND new_category.id = $3
		   AND new_category.user_id = $2
		 RETURNING a.id, a.category_id, a.name, a.multiplier::double precision, a.minimum_minutes, a.tracked_minutes, a.created_at`,
		activityID,
		user.ID,
		current.CategoryID,
		current.Name,
		current.Multiplier,
		current.MinimumMinutes,
		current.TrackedMinutes,
	).Scan(
		&current.ID,
		&current.CategoryID,
		&current.Name,
		&current.Multiplier,
		&current.MinimumMinutes,
		&current.TrackedMinutes,
		&current.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			writeErrorJSON(w, http.StatusConflict, "activity name already exists in this category")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeErrorJSON(w, http.StatusNotFound, "activity not found")
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, "failed to update activity")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"activity": current,
	})
}

func (h *ActivityHandler) handleDeleteActivity(w http.ResponseWriter, r *http.Request, activityID int64) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	result, err := h.db.Exec(
		r.Context(),
		`DELETE FROM activities AS a
		 USING categories AS c
		 WHERE a.id = $1
		   AND a.category_id = c.id
		   AND c.user_id = $2`,
		activityID,
		user.ID,
	)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to delete activity")
		return
	}
	if result.RowsAffected() == 0 {
		writeErrorJSON(w, http.StatusNotFound, "activity not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "activity deleted",
	})
}

func (h *ActivityHandler) findOwnedCategory(r *http.Request, userID, categoryID int64) (Category, error) {
	var category Category
	err := h.db.QueryRow(
		r.Context(),
		`SELECT id, name, multiplier::double precision, created_at
		 FROM categories
		 WHERE id = $1 AND user_id = $2`,
		categoryID,
		userID,
	).Scan(&category.ID, &category.Name, &category.Multiplier, &category.CreatedAt)
	return category, err
}

func (h *ActivityHandler) findActivity(r *http.Request, userID, activityID int64) (Activity, error) {
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

func activityIDFromPath(path string) (int64, error) {
	raw := strings.Trim(strings.TrimPrefix(path, "/activities/"), "/")
	if raw == "" || strings.Contains(raw, "/") {
		return 0, fmt.Errorf("invalid activity path")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid activity id")
	}
	return id, nil
}

func validateActivityInput(name string, multiplier float64, minimumMinutes int, requireName bool) error {
	switch {
	case requireName && strings.TrimSpace(name) == "":
		return fmt.Errorf("activity name is required")
	case len(strings.TrimSpace(name)) > 120:
		return fmt.Errorf("activity name must be at most 120 characters")
	case multiplier <= 0:
		return fmt.Errorf("multiplier must be greater than 0")
	case minimumMinutes < 0:
		return fmt.Errorf("minimum_minutes must be greater than or equal to 0")
	default:
		return nil
	}
}
