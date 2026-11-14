package category

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/schellhas/idletime/backend/internal/httpx"
)

// Handler exposes the category service over HTTP.
type Handler struct {
	service *Service
}

// NewHandler creates a Handler for service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the category routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/categories", h.list)
	mux.HandleFunc("POST /api/v1/categories", h.create)
}

type createRequest struct {
	ParentID   *int64   `json:"parentId"`
	Name       string   `json:"name"`
	Multiplier *float64 `json:"multiplier"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.List(r.Context())
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	if categories == nil {
		categories = []Category{} // serialise as [] rather than null
	}
	httpx.WriteJSON(w, http.StatusOK, categories)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	created, err := h.service.Create(r.Context(), CreateInput(req))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, ErrParentNotFound):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "parent_not_found", err.Error())
	case errors.Is(err, ErrDuplicateName):
		httpx.WriteError(w, http.StatusConflict, "duplicate_name", err.Error())
	default:
		slog.ErrorContext(r.Context(), "category request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
