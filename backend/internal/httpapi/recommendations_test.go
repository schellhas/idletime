package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestRecommendationsRequireAuthentication(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	resp, body := env.request(t, http.DefaultClient, http.MethodGet, "/recommendations", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d with body %s", resp.StatusCode, string(body))
	}
}

func TestRecommendationsReturnMostBehindActivity(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	client := env.registerVerifyLogin(t, "alice")
	categoryID := createTestCategory(t, env, client, "Sport", 2.0)
	swimmingID := createTestActivity(t, env, client, categoryID, "Swimming", 2.0, 30)
	joggingID := createTestActivity(t, env, client, categoryID, "Jogging", 1.0, 20)
	gymID := createTestActivity(t, env, client, categoryID, "Gym", 1.5, 45)

	addTestTimeEntry(t, env, client, swimmingID, 90, "long swim")
	addTestTimeEntry(t, env, client, joggingID, 60, "steady run")
	addTestTimeEntry(t, env, client, gymID, 20, "short lift")

	resp, body := env.request(t, client, http.MethodGet, "/recommendations", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Recommendation *struct {
			ActivityID     int64   `json:"activity_id"`
			ActivityName   string  `json:"activity_name"`
			CategoryName   string  `json:"category_name"`
			CombinedWeight float64 `json:"combined_weight"`
			Reason         string  `json:"reason"`
		} `json:"recommendation"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode recommendation response: %v", err)
	}
	if payload.Recommendation == nil {
		t.Fatalf("expected a recommendation, got body %s", string(body))
	}
	if payload.Recommendation.ActivityID != gymID {
		t.Fatalf("expected gym (%d) to be recommended, got %+v", gymID, payload.Recommendation)
	}
	if payload.Recommendation.ActivityName != "Gym" {
		t.Fatalf("expected gym recommendation, got %+v", payload.Recommendation)
	}
	if payload.Recommendation.CategoryName != "Sport" {
		t.Fatalf("expected Sport category, got %+v", payload.Recommendation)
	}
	if payload.Recommendation.CombinedWeight != 3.0 {
		t.Fatalf("expected combined weight 3.0, got %+v", payload.Recommendation)
	}
	if !strings.Contains(strings.ToLower(payload.Recommendation.Reason), "behind") {
		t.Fatalf("expected explanation to mention behind-ness, got %+v", payload.Recommendation)
	}
}

func TestRecommendationsSupportStatelessSkip(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	client := env.registerVerifyLogin(t, "alice")
	categoryID := createTestCategory(t, env, client, "Sport", 2.0)
	swimmingID := createTestActivity(t, env, client, categoryID, "Swimming", 2.0, 30)
	joggingID := createTestActivity(t, env, client, categoryID, "Jogging", 1.0, 20)
	gymID := createTestActivity(t, env, client, categoryID, "Gym", 1.5, 45)

	addTestTimeEntry(t, env, client, swimmingID, 90, "long swim")
	addTestTimeEntry(t, env, client, joggingID, 60, "steady run")
	addTestTimeEntry(t, env, client, gymID, 20, "short lift")

	resp, body := env.request(t, client, http.MethodGet, fmt.Sprintf("/recommendations?exclude_activity_id=%d", gymID), nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Recommendation *struct {
			ActivityID   int64  `json:"activity_id"`
			ActivityName string `json:"activity_name"`
		} `json:"recommendation"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode recommendation response: %v", err)
	}
	if payload.Recommendation == nil {
		t.Fatalf("expected a recommendation after excluding the top result, got body %s", string(body))
	}
	if payload.Recommendation.ActivityID != swimmingID {
		t.Fatalf("expected swimming (%d) after skip, got %+v", swimmingID, payload.Recommendation)
	}
}

func TestRecommendationsAreUserScoped(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	alice := env.registerVerifyLogin(t, "alice")
	bob := env.registerVerifyLogin(t, "bob")

	aliceCategoryID := createTestCategory(t, env, alice, "Sport", 2.0)
	aliceActivityID := createTestActivity(t, env, alice, aliceCategoryID, "Swimming", 1.0, 30)
	addTestTimeEntry(t, env, alice, aliceActivityID, 15, "lap swim")

	bobCategoryID := createTestCategory(t, env, bob, "Languages", 1.0)
	createTestActivity(t, env, bob, bobCategoryID, "Spanish", 2.0, 20)

	resp, body := env.request(t, bob, http.MethodGet, "/recommendations", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Recommendation *struct {
			ActivityName string `json:"activity_name"`
		} `json:"recommendation"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode user-scoped recommendation response: %v", err)
	}
	if payload.Recommendation == nil {
		t.Fatalf("expected bob to receive a recommendation, got body %s", string(body))
	}
	if payload.Recommendation.ActivityName != "Spanish" {
		t.Fatalf("expected bob to only see his own recommendation, got %+v", payload.Recommendation)
	}
}

func TestRecommendationsReturnNullWhenNoActivitiesExist(t *testing.T) {
	env := newCategoryTestEnv(t)
	defer env.close()

	client := env.registerVerifyLogin(t, "empty")

	resp, body := env.request(t, client, http.MethodGet, "/recommendations", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Recommendation any    `json:"recommendation"`
		Message        string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode empty recommendation response: %v", err)
	}
	if payload.Recommendation != nil {
		t.Fatalf("expected null recommendation, got body %s", string(body))
	}
	if payload.Message == "" {
		t.Fatalf("expected helpful empty-state message, got body %s", string(body))
	}
}

func createTestCategory(t *testing.T, env *categoryTestEnv, client *http.Client, name string, multiplier float64) int64 {
	t.Helper()

	resp, body := env.request(t, client, http.MethodPost, "/categories", map[string]any{
		"name":       name,
		"multiplier": multiplier,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 on category create, got %d with body %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Category struct {
			ID int64 `json:"id"`
		} `json:"category"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode create category response: %v", err)
	}

	return payload.Category.ID
}

func createTestActivity(t *testing.T, env *categoryTestEnv, client *http.Client, categoryID int64, name string, multiplier float64, minimumMinutes int) int64 {
	t.Helper()

	resp, body := env.request(t, client, http.MethodPost, "/activities", map[string]any{
		"category_id":     categoryID,
		"name":            name,
		"multiplier":      multiplier,
		"minimum_minutes": minimumMinutes,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 on activity create, got %d with body %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Activity struct {
			ID int64 `json:"id"`
		} `json:"activity"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode create activity response: %v", err)
	}

	return payload.Activity.ID
}

func addTestTimeEntry(t *testing.T, env *categoryTestEnv, client *http.Client, activityID int64, minutes int, note string) {
	t.Helper()

	resp, body := env.request(t, client, http.MethodPost, "/time-entries", map[string]any{
		"activity_id": activityID,
		"minutes":     minutes,
		"note":        note,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 on time entry create, got %d with body %s", resp.StatusCode, string(body))
	}
}
