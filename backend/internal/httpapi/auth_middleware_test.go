package httpapi

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAuthAddsUserToRequestContext(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	client := env.registerVerifyLogin(t, "alice")

	protected := httptest.NewServer(requireAuth(env.cfg, env.authService, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUserFromContext(r.Context())
		if !ok {
			t.Fatal("expected authenticated user in request context")
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"user": user,
		})
	})))
	defer protected.Close()

	resp, err := client.Get(protected.URL)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", resp.StatusCode, string(body))
	}
	if !bytes.Contains(body, []byte("email_verified")) {
		t.Fatalf("expected user payload in body, got %s", string(body))
	}
}
