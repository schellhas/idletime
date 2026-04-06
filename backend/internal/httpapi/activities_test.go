package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestActivitiesRequireAuthentication(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	resp, body := env.request(t, http.DefaultClient, http.MethodGet, "/activities", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d with body %s", resp.StatusCode, string(body))
	}
}

func TestActivityCRUDIsUserScoped(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	alice := env.registerVerifyLogin(t, "alice")
	bob := env.registerVerifyLogin(t, "bob")

	resp, body := env.request(t, alice, http.MethodPost, "/categories", map[string]any{
		"name":       "Sport",
		"multiplier": 2.0,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 on category create, got %d with body %s", resp.StatusCode, string(body))
	}

	var categoryResponse struct {
		Category struct {
			ID int64 `json:"id"`
		} `json:"category"`
	}
	if err := json.Unmarshal(body, &categoryResponse); err != nil {
		t.Fatalf("decode category response: %v", err)
	}

	resp, body = env.request(t, alice, http.MethodPost, "/activities", map[string]any{
		"category_id":      categoryResponse.Category.ID,
		"name":             "Swimming",
		"multiplier":       1.5,
		"minimum_minutes":  30,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 on activity create, got %d with body %s", resp.StatusCode, string(body))
	}

	var created struct {
		Activity struct {
			ID             int64   `json:"id"`
			CategoryID     int64   `json:"category_id"`
			Name           string  `json:"name"`
			Multiplier     float64 `json:"multiplier"`
			MinimumMinutes int     `json:"minimum_minutes"`
		} `json:"activity"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode create activity response: %v", err)
	}
	if created.Activity.ID == 0 {
		t.Fatalf("expected created activity id, got body %s", string(body))
	}

	resp, body = env.request(t, alice, http.MethodGet, "/activities", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on alice activity list, got %d with body %s", resp.StatusCode, string(body))
	}
	if !bytes.Contains(body, []byte("Swimming")) {
		t.Fatalf("expected alice to see her activity, got body %s", string(body))
	}

	resp, body = env.request(t, bob, http.MethodGet, "/activities", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on bob activity list, got %d with body %s", resp.StatusCode, string(body))
	}
	if bytes.Contains(body, []byte("Swimming")) {
		t.Fatalf("expected bob not to see alice's activity, got body %s", string(body))
	}

	resp, body = env.request(t, bob, http.MethodPost, "/activities", map[string]any{
		"category_id":      categoryResponse.Category.ID,
		"name":             "Steal",
		"multiplier":       1.0,
		"minimum_minutes":  10,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when bob creates activity in alice category, got %d with body %s", resp.StatusCode, string(body))
	}

	resp, body = env.request(t, bob, http.MethodPatch, fmt.Sprintf("/activities/%d", created.Activity.ID), map[string]any{
		"name": "Hacked",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when bob updates alice activity, got %d with body %s", resp.StatusCode, string(body))
	}

	resp, body = env.request(t, alice, http.MethodPatch, fmt.Sprintf("/activities/%d", created.Activity.ID), map[string]any{
		"name":             "Open Water Swimming",
		"multiplier":       2.0,
		"minimum_minutes":  45,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on alice activity update, got %d with body %s", resp.StatusCode, string(body))
	}
	if !bytes.Contains(body, []byte("Open Water Swimming")) {
		t.Fatalf("expected updated activity body, got %s", string(body))
	}

	resp, body = env.request(t, alice, http.MethodDelete, fmt.Sprintf("/activities/%d", created.Activity.ID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on activity delete, got %d with body %s", resp.StatusCode, string(body))
	}

	resp, body = env.request(t, alice, http.MethodGet, "/activities", nil)
	defer resp.Body.Close()
	if bytes.Contains(body, []byte("Open Water Swimming")) {
		t.Fatalf("expected deleted activity to be gone, got body %s", string(body))
	}
}
