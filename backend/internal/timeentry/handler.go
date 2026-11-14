package timeentry

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/schellhas/idletime/backend/internal/auth"
	"github.com/schellhas/idletime/backend/internal/httpx"
	"github.com/schellhas/idletime/backend/internal/validation"
)

// Handler exposes the time entry service over HTTP. All routes require a session.
type Handler struct {
	service *Service
	guard   httpx.Middleware
}

// NewHandler creates a Handler for service. guard authenticates requests.
func NewHandler(service *Service, guard httpx.Middleware) *Handler {
	return &Handler{service: service, guard: guard}
}

// Register mounts the time entry routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/time-entries", h.guard(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/time-entries", h.guard(http.HandlerFunc(h.create)))
	mux.Handle("PATCH /api/v1/time-entries/{id}", h.guard(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/time-entries/{id}", h.guard(http.HandlerFunc(h.delete)))
}

type createRequest struct {
	ActivityID int64      `json:"activityId"`
	Minutes    int        `json:"minutes"`
	Note       string     `json:"note"`
	LoggedAt   *time.Time `json:"loggedAt"`
}

type updateRequest struct {
	Minutes  *int       `json:"minutes"`
	Note     *string    `json:"note"`
	LoggedAt *time.Time `json:"loggedAt"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	filter, err := parseFilter(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	entries, err := h.service.List(r.Context(), u.ID, filter)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	if entries == nil {
		entries = []Entry{} // serialise as [] rather than null
	}
	httpx.WriteJSON(w, http.StatusOK, entries)
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

func parseFilter(r *http.Request) (ListFilter, error) {
	var f ListFilter
	query := r.URL.Query()
	if raw := query.Get("activityId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return f, errors.New("activityId must be a number")
		}
		f.ActivityID = &id
	}
	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return f, errors.New("limit must be a number")
		}
		f.Limit = limit
	}
	return f, nil
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, validation.ErrInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrActivityNotFound):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "activity_not_found", err.Error())
	default:
		slog.ErrorContext(r.Context(), "time entry request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
