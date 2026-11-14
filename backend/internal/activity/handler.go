package activity

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/schellhas/idletime/backend/internal/auth"
	"github.com/schellhas/idletime/backend/internal/httpx"
	"github.com/schellhas/idletime/backend/internal/validation"
)

// Handler exposes the activity service over HTTP. All routes require a session.
type Handler struct {
	service *Service
	guard   httpx.Middleware
}

// NewHandler creates a Handler for service. guard authenticates requests.
func NewHandler(service *Service, guard httpx.Middleware) *Handler {
	return &Handler{service: service, guard: guard}
}

// Register mounts the activity routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/activities", h.guard(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/activities", h.guard(http.HandlerFunc(h.create)))
	mux.Handle("PATCH /api/v1/activities/{id}", h.guard(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/activities/{id}", h.guard(http.HandlerFunc(h.delete)))
}

type createRequest struct {
	CategoryID     int64    `json:"categoryId"`
	Name           string   `json:"name"`
	Multiplier     *float64 `json:"multiplier"`
	MinimumMinutes *int     `json:"minimumMinutes"`
}

type updateRequest struct {
	CategoryID     *int64   `json:"categoryId"`
	Name           *string  `json:"name"`
	Multiplier     *float64 `json:"multiplier"`
	MinimumMinutes *int     `json:"minimumMinutes"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	activities, err := h.service.List(r.Context(), u.ID)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	if activities == nil {
		activities = []Activity{} // serialise as [] rather than null
	}
	httpx.WriteJSON(w, http.StatusOK, activities)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	var req createRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	created, err := h.service.Create(r.Context(), u.ID, CreateInput(req))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathID(w, r)
	if !ok {
		return
	}
	var req updateRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	updated, err := h.service.Update(r.Context(), u.ID, id, UpdateInput(req))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathID(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), u.ID, id); err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, validation.ErrInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrCategoryNotFound):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "category_not_found", err.Error())
	case errors.Is(err, ErrDuplicateName):
		httpx.WriteError(w, http.StatusConflict, "duplicate_name", err.Error())
	default:
		slog.ErrorContext(r.Context(), "activity request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
