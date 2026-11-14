package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/schellhas/idletime/backend/internal/httpx"
	"github.com/schellhas/idletime/backend/internal/user"
)

// Guard returns middleware that only lets requests with a valid session through and
// stores the authenticated user in the request context.
func Guard(service *Service, secureCookie bool) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken := TokenFromRequest(r)
			if rawToken == "" {
				writeUnauthenticated(w)
				return
			}

			u, err := service.Authenticate(r.Context(), rawToken)
			switch {
			case errors.Is(err, ErrUnauthenticated):
				ClearSessionCookie(w, secureCookie)
				writeUnauthenticated(w)
				return
			case err != nil:
				slog.ErrorContext(r.Context(), "authenticate request failed", "error", err)
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
				return
			}

			w.Header().Set("Cache-Control", "no-store") // private data must not be cached
			next.ServeHTTP(w, r.WithContext(user.WithContext(r.Context(), u)))
		})
	}
}

// CurrentUser returns the user authenticated by Guard. If there is none, it writes a
// 401 response and returns false.
func CurrentUser(w http.ResponseWriter, r *http.Request) (user.User, bool) {
	u, ok := user.FromContext(r.Context())
	if !ok {
		writeUnauthenticated(w)
	}
	return u, ok
}

func writeUnauthenticated(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "You are not logged in.")
}
