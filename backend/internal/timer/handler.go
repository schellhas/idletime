package timer

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/schellhas/idletime/backend/internal/auth"
	"github.com/schellhas/idletime/backend/internal/httpx"
)

// Handler exposes the stopwatch over HTTP. All routes require a session.
type Handler struct {
	service *Service
	guard   httpx.Middleware
}

// NewHandler creates a Handler for service. guard authenticates requests.
func NewHandler(service *Service, guard httpx.Middleware) *Handler {
	return &Handler{service: service, guard: guard}
}

// Register mounts the timer routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/timer", h.guard(http.HandlerFunc(h.get)))
	mux.Handle("POST /api/v1/timer", h.guard(http.HandlerFunc(h.start)))
	mux.Handle("POST /api/v1/timer/stop", h.guard(http.HandlerFunc(h.stop)))
	mux.Handle("DELETE /api/v1/timer", h.guard(http.HandlerFunc(h.cancel)))
}

// get returns the running timer, or JSON null when none is running.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	t, running, err := h.service.Current(r.Context(), u.ID)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	if !running {
		httpx.WriteJSON(w, http.StatusOK, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		ActivityID int64 `json:"activityId"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	t, err := h.service.Start(r.Context(), u.ID, req.ActivityID)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, t)
}

func (h *Handler) stop(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	entry, err := h.service.Stop(r.Context(), u.ID)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, entry)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.Cancel(r.Context(), u.ID); err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrAlreadyRunning):
		httpx.WriteError(w, http.StatusConflict, "timer_running", err.Error())
	case errors.Is(err, ErrNotRunning):
		httpx.WriteError(w, http.StatusNotFound, "timer_not_running", err.Error())
	case errors.Is(err, ErrActivityNotFound):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "activity_not_found", err.Error())
	default:
		slog.ErrorContext(r.Context(), "timer request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
