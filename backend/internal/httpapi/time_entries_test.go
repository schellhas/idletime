package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestTimeEntriesRequireAuthentication(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	resp, body := env.request(t, http.DefaultClient, http.MethodGet, "/time-entries", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d with body %s", resp.StatusCode, string(body))
	}
}

func TestTimeEntryCRUDIsUserScoped(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	alice := env.registerVerifyLogin(t, "alice")
	bob := env.registerVerifyLogin(t, "bob")

	resp, body := env.request(t, alice, http.MethodPost, "/categories", map[string]any{
		"name": "Sport",
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
		"category_id":     categoryResponse.Category.ID,
		"name":            "Swimming",
		"minimum_minutes": 30,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 on activity create, got %d with body %s", resp.StatusCode, string(body))
	}

	var activityResponse struct {
		Activity struct {
			ID int64 `json:"id"`
		} `json:"activity"`
	}
	if err := json.Unmarshal(body, &activityResponse); err != nil {
		t.Fatalf("decode activity response: %v", err)
	}

	resp, body = env.request(t, alice, http.MethodPost, "/time-entries", map[string]any{
		"activity_id": activityResponse.Activity.ID,
		"minutes":     45,
		"note":        "Pool session",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 on time entry create, got %d with body %s", resp.StatusCode, string(body))
	}

	var created struct {
		TimeEntry struct {
			ID         int64  `json:"id"`
			ActivityID int64  `json:"activity_id"`
			Minutes    int    `json:"minutes"`
			Note       string `json:"note"`
		} `json:"time_entry"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode time entry response: %v", err)
	}
	if created.TimeEntry.ID == 0 {
		t.Fatalf("expected created time entry id, got body %s", string(body))
	}

	resp, body = env.request(t, alice, http.MethodGet, "/time-entries", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on alice time entry list, got %d with body %s", resp.StatusCode, string(body))
	}
	if !bytes.Contains(body, []byte("Pool session")) {
		t.Fatalf("expected alice to see her time entry, got body %s", string(body))
	}

	resp, body = env.request(t, bob, http.MethodGet, "/time-entries", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on bob time entry list, got %d with body %s", resp.StatusCode, string(body))
	}
	if bytes.Contains(body, []byte("Pool session")) {
		t.Fatalf("expected bob not to see alice's time entry, got body %s", string(body))
	}

	resp, body = env.request(t, bob, http.MethodPost, "/time-entries", map[string]any{
		"activity_id": activityResponse.Activity.ID,
		"minutes":     10,
		"note":        "Steal",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when bob creates entry in alice activity, got %d with body %s", resp.StatusCode, string(body))
	}

	resp, body = env.request(t, bob, http.MethodPatch, fmt.Sprintf("/time-entries/%d", created.TimeEntry.ID), map[string]any{
		"minutes": 15,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when bob updates alice time entry, got %d with body %s", resp.StatusCode, string(body))
	}

	resp, body = env.request(t, alice, http.MethodPatch, fmt.Sprintf("/time-entries/%d", created.TimeEntry.ID), map[string]any{
		"minutes": 60,
		"note":    "Long pool session",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on alice time entry update, got %d with body %s", resp.StatusCode, string(body))
	}
	if !bytes.Contains(body, []byte("Long pool session")) {
		t.Fatalf("expected updated time entry body, got %s", string(body))
	}

	resp, body = env.request(t, alice, http.MethodDelete, fmt.Sprintf("/time-entries/%d", created.TimeEntry.ID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on time entry delete, got %d with body %s", resp.StatusCode, string(body))
	}

	resp, body = env.request(t, alice, http.MethodGet, "/time-entries", nil)
	defer resp.Body.Close()
	if bytes.Contains(body, []byte("Long pool session")) {
		t.Fatalf("expected deleted time entry to be gone, got body %s", string(body))
	}
}
