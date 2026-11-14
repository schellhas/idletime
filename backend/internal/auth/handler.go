package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/schellhas/idletime/backend/internal/httpx"
	"github.com/schellhas/idletime/backend/internal/user"
	"github.com/schellhas/idletime/backend/internal/validation"
)

// Handler exposes the authentication use cases over HTTP.
type Handler struct {
	service      *Service
	limit        httpx.Middleware
	secureCookie bool
	clientIP     func(*http.Request) string
}

// NewHandler creates a Handler. limit is applied to the endpoints that can be abused
// for guessing credentials or sending emails; clientIP identifies the caller for logs.
func NewHandler(service *Service, limit httpx.Middleware, clientIP func(*http.Request) string, secureCookie bool) *Handler {
	return &Handler{service: service, limit: limit, clientIP: clientIP, secureCookie: secureCookie}
}

// Register mounts the auth routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/auth/register", h.limit(http.HandlerFunc(h.register)))
	mux.Handle("POST /api/v1/auth/login", h.limit(http.HandlerFunc(h.login)))
	mux.Handle("POST /api/v1/auth/resend-verification", h.limit(http.HandlerFunc(h.resendVerification)))
	mux.Handle("POST /api/v1/auth/forgot-password", h.limit(http.HandlerFunc(h.forgotPassword)))
	mux.HandleFunc("POST /api/v1/auth/verify-email", h.verifyEmail)
	mux.HandleFunc("POST /api/v1/auth/reset-password", h.resetPassword)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := h.service.Register(r.Context(), RegisterInput(req)); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if !decode(w, r, &req) {
		return
	}
	session, err := h.service.Login(r.Context(), LoginInput{
		Identifier: req.Identifier,
		Password:   req.Password,
		UserAgent:  r.UserAgent(),
		IPAddress:  h.clientIP(r),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	SetSessionCookie(w, session.Token, session.ExpiresAt, h.secureCookie)
	httpx.WriteJSON(w, http.StatusOK, session.User)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if rawToken := TokenFromRequest(r); rawToken != "" {
		if err := h.service.Logout(r.Context(), rawToken); err != nil {
			h.writeError(w, r, err)
			return
		}
	}
	ClearSessionCookie(w, h.secureCookie)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := h.service.VerifyEmail(r.Context(), req.Token); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) resendVerification(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := h.service.ResendVerification(r.Context(), req.Email); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := h.service.ForgotPassword(r.Context(), req.Email); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := h.service.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	case errors.Is(err, ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Wrong username/email or password.")
	case errors.Is(err, ErrEmailNotVerified):
		httpx.WriteError(w, http.StatusForbidden, "email_not_verified", "Please confirm your email address first.")
	case errors.Is(err, ErrInvalidToken):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_token", "This link is invalid or has expired.")
	default:
		slog.ErrorContext(r.Context(), "auth request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
