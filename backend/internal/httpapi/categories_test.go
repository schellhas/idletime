package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"idletime/backend/internal/auth"
	"idletime/backend/internal/config"
	"idletime/backend/internal/database"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

type categoryTestEnv struct {
	server      *httptest.Server
	cfg         config.Config
	authService *auth.Service
}

func TestCategoriesRequireAuthentication(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	resp, body := env.request(t, http.DefaultClient, http.MethodGet, "/categories", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d with body %s", resp.StatusCode, string(body))
	}
}

func TestNewUsersReceiveDefaultRootCategory(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	client := env.registerVerifyLogin(t, "rooted")

	resp, body := env.request(t, client, http.MethodGet, "/categories", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", resp.StatusCode, string(body))
	}
	if !bytes.Contains(body, []byte(`"name":"root"`)) {
		t.Fatalf("expected default root category, got body %s", string(body))
	}
}

func TestRootCategoryCannotBeDeleted(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	client := env.registerVerifyLogin(t, "keeper")

	resp, body := env.request(t, client, http.MethodGet, "/categories", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", resp.StatusCode, string(body))
	}

	var listed struct {
		Categories []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(body, &listed); err != nil {
		t.Fatalf("decode categories response: %v", err)
	}

	var rootID int64
	for _, category := range listed.Categories {
		if category.Name == "root" {
			rootID = category.ID
			break
		}
	}
	if rootID == 0 {
		t.Fatalf("expected root category in list, got body %s", string(body))
	}

	resp, body = env.request(t, client, http.MethodDelete, fmt.Sprintf("/categories/%d", rootID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when deleting root category, got %d with body %s", resp.StatusCode, string(body))
	}
}

func TestCategoryCRUDIsUserScoped(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	alice := env.registerVerifyLogin(t, "alice")
	bob := env.registerVerifyLogin(t, "bob")

	resp, body := env.request(t, alice, http.MethodPost, "/categories", map[string]any{
		"name":       "Sport",
		"multiplier": 2.5,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 on create, got %d with body %s", resp.StatusCode, string(body))
	}

	var created struct {
		Category struct {
			ID         int64   `json:"id"`
			Name       string  `json:"name"`
			Multiplier float64 `json:"multiplier"`
		} `json:"category"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Category.ID == 0 {
		t.Fatalf("expected created category id, got body %s", string(body))
	}

	resp, body = env.request(t, alice, http.MethodGet, "/categories", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on alice list, got %d with body %s", resp.StatusCode, string(body))
	}
	if !bytes.Contains(body, []byte("Sport")) {
		t.Fatalf("expected alice to see her category, got body %s", string(body))
	}

	resp, body = env.request(t, bob, http.MethodGet, "/categories", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on bob list, got %d with body %s", resp.StatusCode, string(body))
	}
	if bytes.Contains(body, []byte("Sport")) {
		t.Fatalf("expected bob not to see alice's category, got body %s", string(body))
	}

	resp, body = env.request(t, bob, http.MethodPatch, fmt.Sprintf("/categories/%d", created.Category.ID), map[string]any{
		"name": "Hacked",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when bob updates alice category, got %d with body %s", resp.StatusCode, string(body))
	}

	resp, body = env.request(t, alice, http.MethodPatch, fmt.Sprintf("/categories/%d", created.Category.ID), map[string]any{
		"name":       "Fitness",
		"multiplier": 3.0,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on alice update, got %d with body %s", resp.StatusCode, string(body))
	}
	if !bytes.Contains(body, []byte("Fitness")) {
		t.Fatalf("expected updated category body, got %s", string(body))
	}

	resp, body = env.request(t, alice, http.MethodDelete, fmt.Sprintf("/categories/%d", created.Category.ID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d with body %s", resp.StatusCode, string(body))
	}

	resp, body = env.request(t, alice, http.MethodGet, "/categories", nil)
	defer resp.Body.Close()
	if bytes.Contains(body, []byte("Fitness")) {
		t.Fatalf("expected deleted category to be gone, got body %s", string(body))
	}
}

func newCategoryTestEnv(t *testing.T) *categoryTestEnv {
	t.Helper()

	port := getFreePort(t)
	tmpDir := t.TempDir()
	binaryDir := filepath.Join(os.TempDir(), "idletime-embedded-postgres")

	db := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(uint32(port)).
			Username("idletime").
			Password("idletime").
			Database("idletime").
			RuntimePath(filepath.Join(tmpDir, "runtime")).
			DataPath(filepath.Join(tmpDir, "data")).
			BinariesPath(binaryDir),
	)

	if err := db.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}

	cfg := config.Config{
		AppEnv:            "development",
		Port:              "0",
		FrontendURL:       "http://localhost:3000",
		SessionCookieName: "idletime_session",
		SessionTTLHours:   24,
		CookieSecure:      false,
		DBHost:            "localhost",
		DBPort:            fmt.Sprintf("%d", port),
		DBUser:            "idletime",
		DBPassword:        "idletime",
		DBName:            "idletime",
		DBSSLMode:         "disable",
	}

	pool, err := database.NewPostgresPool(context.Background(), cfg.DatabaseURL())
	if err != nil {
		_ = db.Stop()
		t.Fatalf("connect test postgres: %v", err)
	}

	migrationPath := migrationFilePath(t)
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		pool.Close()
		_ = db.Stop()
		t.Fatalf("read migration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
		pool.Close()
		_ = db.Stop()
		t.Fatalf("apply migration: %v", err)
	}

	authService := auth.NewService(pool, nil, cfg.FrontendURL, cfg.SessionTTL())
	server := httptest.NewServer(NewRouter(cfg, pool, authService))

	t.Cleanup(func() {
		server.Close()
		pool.Close()
		_ = db.Stop()
	})

	return &categoryTestEnv{
		server:      server,
		cfg:         cfg,
		authService: authService,
	}
}

func (e *categoryTestEnv) close() {
	if e.server != nil {
		e.server.Close()
	}
}

func (e *categoryTestEnv) registerVerifyLogin(t *testing.T, prefix string) *http.Client {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("new cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}

	stamp := time.Now().UnixNano()
	username := fmt.Sprintf("%s_%d", prefix, stamp)
	email := fmt.Sprintf("%s@example.com", username)
	password := "correct-horse-battery-staple"

	resp, body := e.request(t, client, http.MethodPost, "/auth/register", map[string]any{
		"username": username,
		"email":    email,
		"password": password,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var register struct {
		VerificationURL string `json:"verification_url"`
	}
	if err := json.Unmarshal(body, &register); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	token := extractToken(t, register.VerificationURL)

	resp, body = e.request(t, client, http.MethodPost, "/auth/verify-email", map[string]any{"token": token})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("verify failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	resp, body = e.request(t, client, http.MethodPost, "/auth/login", map[string]any{
		"identifier": email,
		"password":   password,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	return client
}

func (e *categoryTestEnv) request(t *testing.T, client *http.Client, method, path string, payload any) (*http.Response, []byte) {
	t.Helper()

	var bodyReader io.Reader
	if payload != nil {
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, e.server.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		t.Fatalf("read response body: %v", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(responseBody))

	return resp, responseBody
}

func migrationFilePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("determine caller path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations", "001_init.sql")
}

func extractToken(t *testing.T, verificationURL string) string {
	t.Helper()
	parsed, err := url.Parse(verificationURL)
	if err != nil {
		t.Fatalf("parse verification url: %v", err)
	}
	token := parsed.Query().Get("token")
	if token == "" {
		t.Fatalf("verification url missing token: %s", verificationURL)
	}
	return token
}

func getFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

