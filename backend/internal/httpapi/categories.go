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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultRootCategoryName = "root"

type CategoryHandler struct {
	cfg         config.Config
	db          *pgxpool.Pool
	authService *auth.Service
}

type Category struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Multiplier float64   `json:"multiplier"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewCategoryHandler(cfg config.Config, db *pgxpool.Pool, authService *auth.Service) *CategoryHandler {
	return &CategoryHandler{cfg: cfg, db: db, authService: authService}
}

func (h *CategoryHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/categories", requireAuth(h.cfg, h.authService, http.HandlerFunc(h.handleCategories)))
	mux.Handle("/categories/", requireAuth(h.cfg, h.authService, http.HandlerFunc(h.handleCategoryByID)))
}

func (h *CategoryHandler) handleCategories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListCategories(w, r)
	case http.MethodPost:
		h.handleCreateCategory(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *CategoryHandler) handleCategoryByID(w http.ResponseWriter, r *http.Request) {
	categoryID, err := categoryIDFromPath(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetCategory(w, r, categoryID)
	case http.MethodPatch, http.MethodPut:
		h.handleUpdateCategory(w, r, categoryID)
	case http.MethodDelete:
		h.handleDeleteCategory(w, r, categoryID)
	default:
		w.Header().Set("Allow", "GET, PATCH, PUT, DELETE")
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *CategoryHandler) handleListCategories(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	if err := h.ensureDefaultRootCategory(r, user.ID); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to prepare default category")
		return
	}

	rows, err := h.db.Query(
		r.Context(),
		`SELECT id, name, multiplier::double precision, created_at
		 FROM categories
		 WHERE user_id = $1
		 ORDER BY created_at ASC, id ASC`,
		user.ID,
	)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to list categories")
		return
	}
	defer rows.Close()

	categories := make([]Category, 0)
	for rows.Next() {
		var category Category
		if err := rows.Scan(&category.ID, &category.Name, &category.Multiplier, &category.CreatedAt); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "failed to read categories")
			return
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to read categories")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"categories": categories,
	})
}

func (h *CategoryHandler) handleGetCategory(w http.ResponseWriter, r *http.Request, categoryID int64) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	category, err := h.findCategory(r, user.ID, categoryID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "category not found")
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load category")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"category": category,
	})
}

func (h *CategoryHandler) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	var request struct {
		Name       string   `json:"name"`
		Multiplier *float64 `json:"multiplier"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	multiplier := 1.0
	if request.Multiplier != nil {
		multiplier = *request.Multiplier
	}
	if err := validateCategoryInput(strings.TrimSpace(request.Name), multiplier, true); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	var category Category
	err := h.db.QueryRow(
		r.Context(),
		`INSERT INTO categories (user_id, name, multiplier)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, multiplier::double precision, created_at`,
		user.ID,
		strings.TrimSpace(request.Name),
		multiplier,
	).Scan(&category.ID, &category.Name, &category.Multiplier, &category.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			writeErrorJSON(w, http.StatusConflict, "category name already exists")
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, "failed to create category")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"category": category,
	})
}

func (h *CategoryHandler) handleUpdateCategory(w http.ResponseWriter, r *http.Request, categoryID int64) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	current, err := h.findCategory(r, user.ID, categoryID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "category not found")
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load category")
		return
	}

	var request struct {
		Name       *string  `json:"name"`
		Multiplier *float64 `json:"multiplier"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if request.Name == nil && request.Multiplier == nil {
		writeErrorJSON(w, http.StatusBadRequest, "at least one category field must be provided")
		return
	}

	if request.Name != nil {
		nextName := strings.TrimSpace(*request.Name)
		if current.Name == defaultRootCategoryName && nextName != defaultRootCategoryName {
			writeErrorJSON(w, http.StatusBadRequest, "root category name cannot be changed")
			return
		}
		current.Name = nextName
	}
	if request.Multiplier != nil {
		current.Multiplier = *request.Multiplier
	}

	if err := validateCategoryInput(current.Name, current.Multiplier, true); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.db.QueryRow(
		r.Context(),
		`UPDATE categories
		 SET name = $3, multiplier = $4
		 WHERE id = $1 AND user_id = $2
		 RETURNING id, name, multiplier::double precision, created_at`,
		categoryID,
		user.ID,
		current.Name,
		current.Multiplier,
	).Scan(&current.ID, &current.Name, &current.Multiplier, &current.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			writeErrorJSON(w, http.StatusConflict, "category name already exists")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeErrorJSON(w, http.StatusNotFound, "category not found")
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, "failed to update category")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"category": current,
	})
}

func (h *CategoryHandler) handleDeleteCategory(w http.ResponseWriter, r *http.Request, categoryID int64) {
	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	category, err := h.findCategory(r, user.ID, categoryID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeErrorJSON(w, http.StatusNotFound, "category not found")
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to load category")
		return
	}
	if category.Name == defaultRootCategoryName {
		writeErrorJSON(w, http.StatusBadRequest, "root category cannot be deleted")
		return
	}

	result, err := h.db.Exec(
		r.Context(),
		`DELETE FROM categories WHERE id = $1 AND user_id = $2`,
		categoryID,
		user.ID,
	)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to delete category")
		return
	}
	if result.RowsAffected() == 0 {
		writeErrorJSON(w, http.StatusNotFound, "category not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "category deleted",
	})
}

func (h *CategoryHandler) ensureDefaultRootCategory(r *http.Request, userID int64) error {
	_, err := h.db.Exec(
		r.Context(),
		`INSERT INTO categories (user_id, name, multiplier)
		 VALUES ($1, $2, 1.0)
		 ON CONFLICT (user_id, name) DO NOTHING`,
		userID,
		defaultRootCategoryName,
	)
	return err
}

func (h *CategoryHandler) findCategory(r *http.Request, userID, categoryID int64) (Category, error) {
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

func categoryIDFromPath(path string) (int64, error) {
	raw := strings.Trim(strings.TrimPrefix(path, "/categories/"), "/")
	if raw == "" || strings.Contains(raw, "/") {
		return 0, fmt.Errorf("invalid category path")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid category id")
	}
	return id, nil
}

func validateCategoryInput(name string, multiplier float64, requireName bool) error {
	switch {
	case requireName && strings.TrimSpace(name) == "":
		return fmt.Errorf("category name is required")
	case len(strings.TrimSpace(name)) > 120:
		return fmt.Errorf("category name must be at most 120 characters")
	case multiplier <= 0:
		return fmt.Errorf("multiplier must be greater than 0")
	default:
		return nil
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
