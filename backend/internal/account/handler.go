package account

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/schellhas/idletime/backend/internal/auth"
	"github.com/schellhas/idletime/backend/internal/httpx"
	"github.com/schellhas/idletime/backend/internal/token"
	"github.com/schellhas/idletime/backend/internal/user"
	"github.com/schellhas/idletime/backend/internal/validation"
)

// Handler exposes the account use cases over HTTP. All routes require a session.
type Handler struct {
	service      *Service
	guard        httpx.Middleware
	secureCookie bool
}

// NewHandler creates a Handler.
func NewHandler(service *Service, guard httpx.Middleware, secureCookie bool) *Handler {
	return &Handler{service: service, guard: guard, secureCookie: secureCookie}
}

// Register mounts the account routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/me", h.guard(http.HandlerFunc(h.get)))
	mux.Handle("PATCH /api/v1/me", h.guard(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/me", h.guard(http.HandlerFunc(h.delete)))
	mux.Handle("POST /api/v1/me/password", h.guard(http.HandlerFunc(h.changePassword)))
	mux.Handle("GET /api/v1/me/export", h.guard(http.HandlerFunc(h.export)))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if u, ok := auth.CurrentUser(w, r); ok {
		httpx.WriteJSON(w, http.StatusOK, u)
	}
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if !decode(w, r, &req) {
		return
	}
	updated, err := h.service.UpdateUsername(r.Context(), u.ID, req.Username)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if !decode(w, r, &req) {
		return
	}
	keep := token.Hash(auth.TokenFromRequest(r))
	if err := h.service.ChangePassword(r.Context(), u.ID, req.CurrentPassword, req.NewPassword, keep); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := h.service.Delete(r.Context(), u.ID, req.Password); err != nil {
		h.writeError(w, r, err)
		return
	}
	auth.ClearSessionCookie(w, h.secureCookie)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.CurrentUser(w, r)
	if !ok {
		return
	}
	data, err := h.service.Export(r.Context(), u)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="idletime-export.json"`)
	httpx.WriteJSON(w, http.StatusOK, data)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := httpx.DecodeJSON(w, r, dst); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return false
	}
	return true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, validation.ErrInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, user.ErrDuplicateUsername):
		httpx.WriteError(w, http.StatusConflict, "username_taken", "This username is already taken.")
	case errors.Is(err, ErrWrongPassword):
		httpx.WriteError(w, http.StatusForbidden, "wrong_password", "The password is incorrect.")
	default:
		slog.ErrorContext(r.Context(), "account request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
