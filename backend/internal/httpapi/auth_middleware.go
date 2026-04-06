package httpapi

import (
	"context"
	"errors"
	"net/http"

	"idletime/backend/internal/auth"
	"idletime/backend/internal/config"
)

type authContextKey string

const authenticatedUserContextKey authContextKey = "authenticated-user"

func requireAuth(cfg config.Config, authService *auth.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cfg.SessionCookieName)
		if err != nil {
			writeErrorJSON(w, http.StatusUnauthorized, "authentication required")
			return
		}

		user, err := authService.CurrentUser(r.Context(), cookie.Value)
		if err != nil {
			writeAuthServiceError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), authenticatedUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func currentUserFromContext(ctx context.Context) (auth.User, bool) {
	user, ok := ctx.Value(authenticatedUserContextKey).(auth.User)
	return user, ok
}

func authenticatedUserFromRequest(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	user, ok := currentUserFromContext(r.Context())
	if !ok {
		writeErrorJSON(w, http.StatusInternalServerError, "authenticated user context missing")
		return auth.User{}, false
	}
	return user, true
}

func writeAuthServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidInput):
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, auth.ErrDuplicateUser):
		writeErrorJSON(w, http.StatusConflict, err.Error())
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeErrorJSON(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, auth.ErrEmailNotVerified):
		writeErrorJSON(w, http.StatusForbidden, "email address has not been verified yet")
	case errors.Is(err, auth.ErrInvalidToken):
		writeErrorJSON(w, http.StatusBadRequest, "invalid or expired token")
	case errors.Is(err, auth.ErrAlreadyVerified):
		writeErrorJSON(w, http.StatusConflict, "email already verified")
	case errors.Is(err, auth.ErrUnauthenticated):
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required")
	default:
		writeErrorJSON(w, http.StatusInternalServerError, "internal server error")
	}
}
