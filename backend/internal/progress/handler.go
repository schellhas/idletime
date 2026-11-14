package progress

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/schellhas/idletime/backend/internal/auth"
	"github.com/schellhas/idletime/backend/internal/httpx"
	"github.com/schellhas/idletime/backend/internal/validation"
)

// Handler exposes recommendations and statistics over HTTP. All routes require a session.
type Handler struct {
	service *Service
	guard   httpx.Middleware
}

// NewHandler creates a Handler for service. guard authenticates requests.
func NewHandler(service *Service, guard httpx.Middleware) *Handler {
	return &Handler{service: service, guard: guard}
}

// Register mounts the routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/recommendation", h.guard(http.HandlerFunc(h.recommend)))
	mux.Handle("GET /api/v1/stats", h.guard(http.HandlerFunc(h.stats)))
}

// recommend answers GET /recommendation?availableMinutes=30&categoryId=2&skip=4,5
func (h *Handler) recommend(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	options, err := parseOptions(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	rec, found, err := h.service.Recommend(r.Context(), u.ID, options)
	switch {
	case errors.Is(err, ErrCategoryNotFound):
		httpx.WriteError(w, http.StatusNotFound, "category_not_found", err.Error())
	case err != nil:
		h.writeError(w, r, err)
	case !found:
		httpx.WriteError(w, http.StatusNotFound, "no_recommendation", "Nothing fits right now.")
	default:
		httpx.WriteJSON(w, http.StatusOK, rec)
	}
}

// stats answers GET /stats?days=30&tz=Europe/Berlin
func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	days := 0
	if raw := r.URL.Query().Get("days"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "days must be a number")
			return
		}
		days = parsed
	}

	stats, err := h.service.Stats(r.Context(), u.ID, days, r.URL.Query().Get("tz"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, stats)
}

func parseOptions(r *http.Request) (Options, error) {
	query := r.URL.Query()
	var o Options

	if raw := query.Get("availableMinutes"); raw != "" {
		minutes, err := strconv.Atoi(raw)
		if err != nil || minutes < 0 {
			return o, errors.New("availableMinutes must be a non-negative number")
		}
		o.AvailableMinutes = minutes
	}
	if raw := query.Get("categoryId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return o, errors.New("categoryId must be a number")
		}
		o.CategoryID = &id
	}
	if raw := query.Get("skip"); raw != "" {
		o.Skip = make(map[int64]bool)
		for _, part := range strings.Split(raw, ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err != nil {
				return o, fmt.Errorf("skip must be a comma-separated list of ids")
			}
			o.Skip[id] = true
		}
	}
	return o, nil
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, validation.ErrInvalid) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	slog.ErrorContext(r.Context(), "progress request failed", "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
}
