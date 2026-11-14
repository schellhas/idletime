package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimiterAllow(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := New(2, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if ok, retry := l.Allow("a"); ok || retry <= 0 {
		t.Fatalf("third request should be blocked with a retry delay, got ok=%v retry=%v", ok, retry)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Error("other keys must not be affected")
	}

	now = now.Add(time.Minute)
	if ok, _ := l.Allow("a"); !ok {
		t.Error("the window should have reset")
	}
}

func TestMiddlewareReturns429(t *testing.T) {
	l := New(1, time.Minute)
	handler := Middleware(l, func(*http.Request) string { return "k" })(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/", nil))
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/", nil))

	if first.Code != http.StatusOK || second.Code != http.StatusTooManyRequests {
		t.Fatalf("statuses = %d, %d; want 200, 429", first.Code, second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After header")
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:4321"
	req.Header.Set("X-Real-IP", "203.0.113.9")

	if got := ClientIP(req, false); got != "10.0.0.5" {
		t.Errorf("untrusted proxy: got %q, want the socket address", got)
	}
	if got := ClientIP(req, true); got != "203.0.113.9" {
		t.Errorf("trusted proxy: got %q, want the X-Real-IP value", got)
	}
}
