package category

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/schellhas/idletime/backend/internal/user"
)

// asUser stands in for the auth guard: it authenticates every request as user 1.
func asUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(user.WithContext(r.Context(), user.User{ID: 1})))
	})
}

func newTestMux(repo Repository) *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(NewService(repo), asUser).Register(mux)
	return mux
}

func TestHandlerRejectsRequestsWithoutAuthenticatedUser(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(NewService(&fakeRepository{}), func(next http.Handler) http.Handler { return next }).Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerListReturnsEmptyArray(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestMux(&fakeRepository{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("body = %s, want []", body)
	}
}

func TestHandlerCreate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		repoErr    error
		wantStatus int
	}{
		{name: "created", body: `{"name":"Sports","multiplier":2}`, wantStatus: http.StatusCreated},
		{name: "invalid json", body: `{"name":`, wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: `{"name":"Sports","foo":1}`, wantStatus: http.StatusBadRequest},
		{name: "invalid input", body: `{"name":""}`, wantStatus: http.StatusBadRequest},
		{name: "unknown parent", body: `{"name":"Sports","parentId":99}`, repoErr: ErrParentNotFound, wantStatus: http.StatusUnprocessableEntity},
		{name: "duplicate", body: `{"name":"Sports"}`, repoErr: ErrDuplicateName, wantStatus: http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(tt.body))

			newTestMux(&fakeRepository{err: tt.repoErr}).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
		})
	}
}

func TestHandlerHidesOwnerID(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":"Sports"}`))

	newTestMux(&fakeRepository{}).ServeHTTP(rec, req)

	if strings.Contains(strings.ToLower(rec.Body.String()), "userid") {
		t.Errorf("response leaks the owner id: %s", rec.Body)
	}
}
