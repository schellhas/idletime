package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"idletime/backend/internal/auth"
	"idletime/backend/internal/config"
)

type AuthHandler struct {
	cfg         config.Config
	authService *auth.Service
}

func NewAuthHandler(cfg config.Config, authService *auth.Service) *AuthHandler {
	return &AuthHandler{cfg: cfg, authService: authService}
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/register", h.handleRegister)
	mux.HandleFunc("/auth/verify-email", h.handleVerifyEmail)
	mux.HandleFunc("/auth/login", h.handleLogin)
	mux.HandleFunc("/auth/logout", h.handleLogout)
	mux.HandleFunc("/auth/me", h.handleCurrentUser)
	mux.HandleFunc("/auth/resend-verification", h.handleResendVerification)
}

func (h *AuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var request struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	result, err := h.authService.Register(r.Context(), auth.RegisterInput{
		Username: request.Username,
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	response := map[string]any{
		"message": "registration successful; verify your email to continue",
		"user":    result.User,
	}
	if h.cfg.AppEnv != "production" {
		response["verification_url"] = result.VerificationURL
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *AuthHandler) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var request struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := h.authService.VerifyEmail(r.Context(), request.Token)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "email verified",
		"user":    user,
	})
}

func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var request struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	result, err := h.authService.Login(r.Context(), auth.LoginInput{
		Identifier: request.Identifier,
		Password:   request.Password,
		UserAgent:  r.UserAgent(),
		RemoteAddr: r.RemoteAddr,
	})
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	h.setSessionCookie(w, result.SessionToken, result.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "login successful",
		"user":    result.User,
	})
}

func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	sessionToken, _ := h.sessionTokenFromRequest(r)
	if sessionToken != "" {
		if err := h.authService.Logout(r.Context(), sessionToken); err != nil && !errors.Is(err, auth.ErrUnauthenticated) {
			h.writeAuthError(w, err)
			return
		}
	}

	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "logout successful",
	})
}

func (h *AuthHandler) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	sessionToken, err := h.sessionTokenFromRequest(r)
	if err != nil {
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.authService.CurrentUser(r.Context(), sessionToken)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user": user,
	})
}

func (h *AuthHandler) handleResendVerification(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var request struct {
		Identifier string `json:"identifier"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	verificationURL, err := h.authService.ResendVerification(r.Context(), request.Identifier)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	response := map[string]any{
		"message": "verification email sent",
	}
	if h.cfg.AppEnv != "production" {
		response["verification_url"] = verificationURL
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) writeAuthError(w http.ResponseWriter, err error) {
	writeAuthServiceError(w, err)
}

func (h *AuthHandler) sessionTokenFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(h.cfg.SessionCookieName)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		w.Header().Set("Allow", method)
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	return true
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeErrorJSON(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
