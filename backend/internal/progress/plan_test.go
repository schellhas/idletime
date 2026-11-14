package progress

import (
	"errors"
	"math"
	"testing"

	"github.com/schellhas/idletime/backend/internal/activity"
	"github.com/schellhas/idletime/backend/internal/category"
)

func cat(id int64, name string, weight float64, parent int64) category.Category {
	c := category.Category{ID: id, Name: name, Multiplier: weight}
	if parent != 0 {
		c.ParentID = &parent
	}
	return c
}

func act(id int64, name string, categoryID int64, weight float64, minimum, tracked int) activity.Activity {
	return activity.Activity{ID: id, Name: name, CategoryID: categoryID, Multiplier: weight, MinimumMinutes: minimum, TrackedMinutes: tracked}
}

// The example from the readme: sports (2.0), self-care (1.0), languages (0.3), with
// swimming (2.0), jogging (1.0) and gym (1.5) inside sports.
func readmeExample(tracked map[string]int) *Plan {
	return NewPlan(
		[]category.Category{
			cat(1, "Sports", 2.0, 0), cat(2, "Self-care", 1.0, 0), cat(3, "Languages", 0.3, 0),
		},
		[]activity.Activity{
			act(10, "Swimming", 1, 2.0, 20, tracked["Swimming"]),
			act(11, "Jogging", 1, 1.0, 0, tracked["Jogging"]),
			act(12, "Gym", 1, 1.5, 0, tracked["Gym"]),
			act(20, "Reading", 2, 1.0, 0, tracked["Reading"]),
			act(30, "Spanish", 3, 1.0, 0, tracked["Spanish"]),
		},
	)
}

func recommended(t *testing.T, p *Plan, o Options) string {
	t.Helper()
	rec, ok, err := p.Recommend(o)
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if !ok {
		return ""
	}
	return rec.Activity.Name
}

func TestRecommendStartsWithTheHeaviestBranch(t *testing.T) {
	// Nothing tracked yet: the heaviest category, then its heaviest activity.
	if got := recommended(t, readmeExample(nil), Options{}); got != "Swimming" {
		t.Errorf("got %q, want Swimming", got)
	}
}

func TestRecommendKeepsTheTimeInProportion(t *testing.T) {
	tracked := map[string]int{}
	// Simulate 400 recommendation rounds of 30 minutes each and check the resulting split.
	for i := 0; i < 400; i++ {
		name := recommended(t, readmeExample(tracked), Options{})
		tracked[name] += 30
	}

	sports := tracked["Swimming"] + tracked["Jogging"] + tracked["Gym"]
	total := sports + tracked["Reading"] + tracked["Spanish"]

	assertRatio(t, "sports : self-care", float64(sports), float64(tracked["Reading"]), 2.0/1.0)
	assertRatio(t, "self-care : languages", float64(tracked["Reading"]), float64(tracked["Spanish"]), 1.0/0.3)
	assertRatio(t, "swimming : jogging", float64(tracked["Swimming"]), float64(tracked["Jogging"]), 2.0/1.0)
	assertRatio(t, "gym : jogging", float64(tracked["Gym"]), float64(tracked["Jogging"]), 1.5/1.0)
	if total != 400*30 {
		t.Fatalf("total = %d, want %d", total, 400*30)
	}
}

func assertRatio(t *testing.T, name string, a, b, want float64) {
	t.Helper()
	if got := a / b; math.Abs(got-want)/want > 0.05 { // within 5 %
		t.Errorf("%s = %.2f, want about %.2f", name, got, want)
	}
}

func TestCategoryWeightDoesNotDependOnItsNumberOfActivities(t *testing.T) {
	// Equal category weights, but "Many" holds five activities and "One" a single one.
	categories := []category.Category{cat(1, "Many", 1, 0), cat(2, "One", 1, 0)}
	tracked := map[int64]int{}
	activities := func() []activity.Activity {
		list := []activity.Activity{act(20, "Solo", 2, 1, 0, tracked[20])}
		for id := int64(10); id < 15; id++ {
			list = append(list, act(id, string(rune('a'+id)), 1, 1, 0, tracked[id]))
		}
		return list
	}

	for i := 0; i < 200; i++ {
		rec, _, _ := NewPlan(categories, activities()).Recommend(Options{})
		tracked[rec.Activity.ID] += 10
	}
	many := tracked[10] + tracked[11] + tracked[12] + tracked[13] + tracked[14]
	if one := tracked[20]; math.Abs(float64(many-one)) > 10 {
		t.Errorf("Many = %d min, One = %d min; categories of equal weight must get equal time", many, one)
	}
}

func TestRecommendRespectsTheTimeSlot(t *testing.T) {
	p := readmeExample(nil)

	// Swimming needs 20 minutes; with 15 minutes the next best sports activity is chosen.
	if got := recommended(t, p, Options{AvailableMinutes: 15}); got != "Gym" {
		t.Errorf("got %q, want Gym", got)
	}
	if got := recommended(t, p, Options{AvailableMinutes: 20}); got != "Swimming" {
		t.Errorf("got %q, want Swimming", got)
	}
}

func TestRecommendSkipsDeclinedActivities(t *testing.T) {
	p := readmeExample(nil)

	got := recommended(t, p, Options{Skip: map[int64]bool{10: true}})
	if got != "Gym" {
		t.Errorf("after skipping Swimming got %q, want Gym", got)
	}

	// Skipping everything in a category moves on to the next category.
	got = recommended(t, p, Options{Skip: map[int64]bool{10: true, 11: true, 12: true}})
	if got != "Reading" {
		t.Errorf("after skipping all sports got %q, want Reading", got)
	}

	all := map[int64]bool{10: true, 11: true, 12: true, 20: true, 30: true}
	if got := recommended(t, p, Options{Skip: all}); got != "" {
		t.Errorf("got %q, want no recommendation", got)
	}
}

func TestRecommendCanBeScopedToACategory(t *testing.T) {
	p := readmeExample(nil)
	selfCare := int64(2)

	if got := recommended(t, p, Options{CategoryID: &selfCare}); got != "Reading" {
		t.Errorf("got %q, want Reading", got)
	}

	unknown := int64(999)
	if _, _, err := p.Recommend(Options{CategoryID: &unknown}); !errors.Is(err, ErrCategoryNotFound) {
		t.Errorf("error = %v, want ErrCategoryNotFound", err)
	}
}

func TestRecommendReturnsThePathToTheActivity(t *testing.T) {
	p := NewPlan(
		[]category.Category{cat(1, "Life", 1, 0), cat(2, "Sports", 1, 1)},
		[]activity.Activity{act(10, "Swimming", 2, 1, 0, 0)},
	)
	rec, ok, _ := p.Recommend(Options{})
	if !ok || len(rec.Path) != 2 || rec.Path[0] != "Life" || rec.Path[1] != "Sports" {
		t.Errorf("path = %v, want [Life Sports]", rec.Path)
	}
}

func TestEmptyCategoriesAreIgnored(t *testing.T) {
	p := NewPlan(
		[]category.Category{cat(1, "Empty", 100, 0), cat(2, "Used", 1, 0), cat(3, "Nested empty", 1, 2)},
		[]activity.Activity{act(10, "Reading", 2, 1, 0, 0)},
	)

	shares := p.Shares()
	if len(shares) != 2 {
		t.Fatalf("shares = %+v, want only Used and Reading", shares)
	}
	if math.Abs(shares[0].TargetShare-1) > epsilon {
		t.Errorf("target share of the only non-empty category = %v, want 1", shares[0].TargetShare)
	}
}

func TestNoActivitiesMeansNoRecommendation(t *testing.T) {
	if _, ok, err := NewPlan(nil, nil).Recommend(Options{}); ok || err != nil {
		t.Errorf("ok = %v, err = %v; want no recommendation", ok, err)
	}
}

func TestShares(t *testing.T) {
	p := readmeExample(map[string]int{"Swimming": 60, "Reading": 30, "Spanish": 10})

	byName := map[string]Share{}
	for _, s := range p.Shares() {
		byName[s.Name] = s
	}

	const total = 100.0
	checks := []struct {
		name   string
		target float64
		actual float64
	}{
		{"Sports", 2.0 / 3.3, 60 / total},
		{"Self-care", 1.0 / 3.3, 30 / total},
		{"Languages", 0.3 / 3.3, 10 / total},
		// Nested targets multiply: sports' share times swimming's share among the sports activities.
		{"Swimming", 2.0 / 3.3 * 2.0 / 4.5, 60 / total},
		{"Jogging", 2.0 / 3.3 * 1.0 / 4.5, 0},
	}
	for _, c := range checks {
		got := byName[c.name]
		if math.Abs(got.TargetShare-c.target) > 1e-6 || math.Abs(got.ActualShare-c.actual) > 1e-6 {
			t.Errorf("%s: target %.4f / actual %.4f, want %.4f / %.4f", c.name, got.TargetShare, got.ActualShare, c.target, c.actual)
		}
	}
	if byName["Sports"].Depth != 0 || byName["Swimming"].Depth != 1 {
		t.Errorf("depths = %d, %d; want 0, 1", byName["Sports"].Depth, byName["Swimming"].Depth)
	}
	if p.TotalMinutes() != 100 {
		t.Errorf("TotalMinutes() = %d, want 100", p.TotalMinutes())
	}
}

func TestSharesWithoutTrackedTimeHaveZeroActualShare(t *testing.T) {
	for _, s := range readmeExample(nil).Shares() {
		if s.ActualShare != 0 {
			t.Errorf("%s: actual share = %v, want 0", s.Name, s.ActualShare)
		}
	}
}
