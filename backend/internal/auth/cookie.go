package auth

import (
	"net/http"
	"time"
)

const cookieName = "idletime_session"

// TokenFromRequest returns the raw session token sent by the client, or "".
func TokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// SetSessionCookie sends the session token to the client. The cookie is invisible to
// JavaScript (HttpOnly) and not sent on cross-site requests (SameSite=Lax).
func SetSessionCookie(w http.ResponseWriter, rawToken string, expires time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    rawToken,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie tells the client to drop its session cookie.
func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
