// Package ratelimit provides a small in-memory fixed-window rate limiter.
package ratelimit

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/schellhas/idletime/backend/internal/httpx"
)

type bucket struct {
	count   int
	resetAt time.Time
}

// Limiter allows at most limit events per key within each window.
// State is kept in memory, so limits apply per server instance.
type Limiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	buckets   map[string]*bucket
	lastPurge time.Time
	now       func() time.Time
}

// New creates a Limiter.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, buckets: make(map[string]*bucket), now: time.Now}
}

// Allow records an event for key and reports whether it is within the limit.
// When it is not, retryAfter tells how long until the window resets.
func (l *Limiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.purgeExpired(now)

	b, ok := l.buckets[key]
	if !ok || !now.Before(b.resetAt) {
		b = &bucket{resetAt: now.Add(l.window)}
		l.buckets[key] = b
	}
	b.count++
	if b.count > l.limit {
		return false, b.resetAt.Sub(now)
	}
	return true, 0
}

// purgeExpired drops stale buckets at most once per window so memory stays bounded.
func (l *Limiter) purgeExpired(now time.Time) {
	if now.Sub(l.lastPurge) < l.window {
		return
	}
	for key, b := range l.buckets {
		if !now.Before(b.resetAt) {
			delete(l.buckets, key)
		}
	}
	l.lastPurge = now
}

// Middleware rejects requests with HTTP 429 once the limit for keyFn(r) is exceeded.
func Middleware(l *Limiter, keyFn func(*http.Request) string) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if allowed, retryAfter := l.Allow(keyFn(r)); !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
				httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "Too many requests. Please try again later.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP returns the caller's IP address. Behind a trusted reverse proxy it uses the
// X-Real-IP header the proxy sets; otherwise the header is ignored because clients can forge it.
func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
