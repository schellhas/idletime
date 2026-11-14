package category

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestMux(repo Repository) *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(NewService(repo)).Register(mux)
	return mux
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
