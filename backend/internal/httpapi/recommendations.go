package httpapi

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"idletime/backend/internal/auth"
	"idletime/backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

const recommendationProgressEpsilon = 1e-9

type RecommendationHandler struct {
	cfg         config.Config
	db          *pgxpool.Pool
	authService *auth.Service
}

type Recommendation struct {
	ActivityID         int64   `json:"activity_id"`
	ActivityName       string  `json:"activity_name"`
	CategoryID         int64   `json:"category_id"`
	CategoryName       string  `json:"category_name"`
	MinimumMinutes     int     `json:"minimum_minutes"`
	TrackedMinutes     int     `json:"tracked_minutes"`
	ActivityMultiplier float64 `json:"activity_multiplier"`
	CategoryMultiplier float64 `json:"category_multiplier"`
	CombinedWeight     float64 `json:"combined_weight"`
	NormalizedProgress float64 `json:"normalized_progress"`
	Reason             string  `json:"reason"`
}

type recommendationCandidate struct {
	ActivityID         int64
	ActivityName       string
	CategoryID         int64
	CategoryName       string
	MinimumMinutes     int
	TrackedMinutes     int
	ActivityMultiplier float64
	CategoryMultiplier float64
	CreatedAt          time.Time
	CombinedWeight     float64
	NormalizedProgress float64
}

func NewRecommendationHandler(cfg config.Config, db *pgxpool.Pool, authService *auth.Service) *RecommendationHandler {
	return &RecommendationHandler{cfg: cfg, db: db, authService: authService}
}

func (h *RecommendationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/recommendations", requireAuth(h.cfg, h.authService, http.HandlerFunc(h.handleRecommendations)))
}

func (h *RecommendationHandler) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, ok := authenticatedUserFromRequest(w, r)
	if !ok {
		return
	}

	excludeActivityIDs, excludeSet, err := excludeActivityIDsFromRequest(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	categoryIDs, categoryFilterSet, err := categoryIDsFromRequest(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	recommendation, message, err := h.buildRecommendation(r, user.ID, excludeActivityIDs, excludeSet, categoryIDs, categoryFilterSet)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "failed to build recommendation")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"recommendation": recommendation,
		"message":        message,
	})
}

func excludeActivityIDsFromRequest(r *http.Request) ([]int64, bool, error) {
	rawValues := append([]string{}, r.URL.Query()["exclude_activity_id"]...)
	if csv := strings.TrimSpace(r.URL.Query().Get("exclude_activity_ids")); csv != "" {
		rawValues = append(rawValues, strings.Split(csv, ",")...)
	}
	if len(rawValues) == 0 {
		return nil, false, nil
	}

	excludeIDs := make([]int64, 0, len(rawValues))
	seen := make(map[int64]struct{}, len(rawValues))
	for _, rawValue := range rawValues {
		value := strings.TrimSpace(rawValue)
		if value == "" {
			continue
		}

		activityID, err := strconv.ParseInt(value, 10, 64)
		if err != nil || activityID <= 0 {
			return nil, false, fmt.Errorf("exclude activity filters must be positive integers")
		}
		if _, exists := seen[activityID]; exists {
			continue
		}
		seen[activityID] = struct{}{}
		excludeIDs = append(excludeIDs, activityID)
	}

	if len(excludeIDs) == 0 {
		return nil, false, nil
	}

	return excludeIDs, true, nil
}

func categoryIDsFromRequest(r *http.Request) ([]int64, bool, error) {
	rawValues := append([]string{}, r.URL.Query()["category_id"]...)
	if csv := strings.TrimSpace(r.URL.Query().Get("category_ids")); csv != "" {
		rawValues = append(rawValues, strings.Split(csv, ",")...)
	}
	if len(rawValues) == 0 {
		return nil, false, nil
	}

	categoryIDs := make([]int64, 0, len(rawValues))
	seen := make(map[int64]struct{}, len(rawValues))
	for _, rawValue := range rawValues {
		value := strings.TrimSpace(rawValue)
		if value == "" {
			continue
		}

		categoryID, err := strconv.ParseInt(value, 10, 64)
		if err != nil || categoryID <= 0 {
			return nil, false, fmt.Errorf("category filters must be positive integers")
		}
		if _, exists := seen[categoryID]; exists {
			continue
		}
		seen[categoryID] = struct{}{}
		categoryIDs = append(categoryIDs, categoryID)
	}

	if len(categoryIDs) == 0 {
		return nil, false, nil
	}

	return categoryIDs, true, nil
}

func (h *RecommendationHandler) expandCategoryIDs(ctx context.Context, userID int64, categoryIDs []int64) ([]int64, error) {
	rows, err := h.db.Query(
		ctx,
		`WITH RECURSIVE selected_categories AS (
		     SELECT id, parent_id
		     FROM categories
		     WHERE user_id = $1
		       AND id = ANY($2)
		   UNION ALL
		     SELECT c.id, c.parent_id
		     FROM categories c
		     JOIN selected_categories sc ON c.parent_id = sc.id
		     WHERE c.user_id = $1
		 )
		 SELECT DISTINCT id
		 FROM selected_categories
		 ORDER BY id ASC`,
		userID,
		categoryIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expandedIDs := make([]int64, 0, len(categoryIDs))
	for rows.Next() {
		var categoryID int64
		if err := rows.Scan(&categoryID); err != nil {
			return nil, err
		}
		expandedIDs = append(expandedIDs, categoryID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return expandedIDs, nil
}

func (h *RecommendationHandler) buildRecommendation(r *http.Request, userID int64, excludeActivityIDs []int64, excludeSet bool, categoryIDs []int64, categoryFilterSet bool) (*Recommendation, string, error) {
	query := `SELECT a.id,
	                a.name,
	                a.category_id,
	                c.name,
	                a.minimum_minutes,
	                a.tracked_minutes,
	                a.multiplier::double precision,
	                c.multiplier::double precision,
	                a.created_at
	         FROM activities a
	         JOIN categories c ON c.id = a.category_id
	         WHERE c.user_id = $1`
	args := []any{userID}

	if excludeSet {
		placeholders := make([]string, 0, len(excludeActivityIDs))
		for _, activityID := range excludeActivityIDs {
			args = append(args, activityID)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		query += ` AND a.id NOT IN (` + strings.Join(placeholders, ", ") + `)`
	}
	if categoryFilterSet {
		expandedCategoryIDs, err := h.expandCategoryIDs(r.Context(), userID, categoryIDs)
		if err != nil {
			return nil, "", err
		}
		if len(expandedCategoryIDs) == 0 {
			return nil, "No activities match the selected categories yet.", nil
		}

		placeholders := make([]string, 0, len(expandedCategoryIDs))
		for _, categoryID := range expandedCategoryIDs {
			args = append(args, categoryID)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		query += ` AND a.category_id IN (` + strings.Join(placeholders, ", ") + `)`
	}

	query += ` ORDER BY a.created_at ASC, a.id ASC`

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	candidates := make([]recommendationCandidate, 0)
	for rows.Next() {
		var candidate recommendationCandidate
		if err := rows.Scan(
			&candidate.ActivityID,
			&candidate.ActivityName,
			&candidate.CategoryID,
			&candidate.CategoryName,
			&candidate.MinimumMinutes,
			&candidate.TrackedMinutes,
			&candidate.ActivityMultiplier,
			&candidate.CategoryMultiplier,
			&candidate.CreatedAt,
		); err != nil {
			return nil, "", err
		}

		candidate.CombinedWeight = candidate.ActivityMultiplier * candidate.CategoryMultiplier
		if candidate.CombinedWeight <= 0 {
			candidate.NormalizedProgress = math.MaxFloat64
		} else {
			candidate.NormalizedProgress = float64(candidate.TrackedMinutes) / candidate.CombinedWeight
		}

		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	if len(candidates) == 0 {
		if excludeSet && categoryFilterSet {
			return nil, "No alternative recommendation is available for the selected categories yet.", nil
		}
		if excludeSet {
			return nil, "No alternative recommendation is available yet. Add more activities to get another option.", nil
		}
		if categoryFilterSet {
			return nil, "No activities match the selected categories yet.", nil
		}
		return nil, "Create a category and at least one activity to get a recommendation.", nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]

		if math.Abs(left.NormalizedProgress-right.NormalizedProgress) > recommendationProgressEpsilon {
			return left.NormalizedProgress < right.NormalizedProgress
		}
		if math.Abs(left.CombinedWeight-right.CombinedWeight) > recommendationProgressEpsilon {
			return left.CombinedWeight > right.CombinedWeight
		}
		if left.TrackedMinutes != right.TrackedMinutes {
			return left.TrackedMinutes < right.TrackedMinutes
		}
		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.Before(right.CreatedAt)
		}
		return left.ActivityID < right.ActivityID
	})

	best := candidates[0]
	reason := fmt.Sprintf(
		"%s is currently the most behind based on your ratios (%.0f tracked min at combined weight %.2f).",
		best.ActivityName,
		float64(best.TrackedMinutes),
		best.CombinedWeight,
	)

	return &Recommendation{
		ActivityID:         best.ActivityID,
		ActivityName:       best.ActivityName,
		CategoryID:         best.CategoryID,
		CategoryName:       best.CategoryName,
		MinimumMinutes:     best.MinimumMinutes,
		TrackedMinutes:     best.TrackedMinutes,
		ActivityMultiplier: best.ActivityMultiplier,
		CategoryMultiplier: best.CategoryMultiplier,
		CombinedWeight:     best.CombinedWeight,
		NormalizedProgress: best.NormalizedProgress,
		Reason:             reason,
	}, reason, nil
}
