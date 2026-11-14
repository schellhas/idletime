// Package progress decides what to do next and reports how far each activity is from
// its target.
//
// Every category and activity carries a multiplier, the weight relative to its siblings
// (subcategories and activities directly below the same category, or all top-level
// categories). The share of the total time a node should get is its weight divided by the
// sum of its siblings' weights, multiplied down from the top. So "twice as much sports
// as self-care" holds no matter how many activities each category contains.
package progress

import (
	"errors"
	"math"
	"sort"
	"strings"

	"github.com/schellhas/idletime/backend/internal/activity"
	"github.com/schellhas/idletime/backend/internal/category"
)

// Errors for scopes that point to something that does not exist.
var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrActivityNotFound = errors.New("activity not found")
)

// epsilon is the tolerance when comparing progress ratios.
const epsilon = 1e-9

// Kind tells categories and activities apart in results.
type Kind string

// Node kinds.
const (
	KindCategory Kind = "category"
	KindActivity Kind = "activity"
)

type node struct {
	kind     Kind
	id       int64
	name     string
	weight   float64
	activity activity.Activity // set for activity nodes
	// tracked is the time logged for the activity, or for all activities below a category.
	tracked  int
	children []*node
	parent   *node
}

// Plan is the weighted tree of a user's categories and activities.
type Plan struct {
	roots      []*node
	categories map[int64]*node
	activities map[int64]*node
}

// NewPlan builds the tree. Categories without any activity below them are left out: they
// can never be recommended, so they must not claim a share of the total either.
// Activities pointing to an unknown category are ignored.
func NewPlan(categories []category.Category, activities []activity.Activity) *Plan {
	p := &Plan{categories: make(map[int64]*node, len(categories)), activities: make(map[int64]*node, len(activities))}
	for _, c := range categories {
		p.categories[c.ID] = &node{kind: KindCategory, id: c.ID, name: c.Name, weight: c.Multiplier}
	}

	for _, c := range categories {
		n := p.categories[c.ID]
		if parent, ok := p.categories[derefOr(c.ParentID, 0)]; ok && parent != n {
			n.parent = parent
			parent.children = append(parent.children, n)
		} else {
			p.roots = append(p.roots, n)
		}
	}
	for _, a := range activities {
		parent, ok := p.categories[a.CategoryID]
		if !ok {
			continue
		}
		child := &node{
			kind: KindActivity, id: a.ID, name: a.Name, weight: a.Multiplier,
			activity: a, tracked: a.TrackedMinutes, parent: parent,
		}
		parent.children = append(parent.children, child)
		p.activities[a.ID] = child
	}

	p.roots = pruneAndSort(p.roots)
	for _, root := range p.roots {
		sumTracked(root)
	}
	return p
}

// pruneAndSort removes subtrees without activities and orders siblings by name.
func pruneAndSort(nodes []*node) []*node {
	kept := nodes[:0:0]
	for _, n := range nodes {
		if n.kind == KindCategory {
			n.children = pruneAndSort(n.children)
			if len(n.children) == 0 {
				continue
			}
		}
		kept = append(kept, n)
	}
	sort.SliceStable(kept, func(i, j int) bool { return less(kept[i], kept[j]) })
	return kept
}

func less(a, b *node) bool {
	if an, bn := strings.ToLower(a.name), strings.ToLower(b.name); an != bn {
		return an < bn
	}
	return a.id < b.id
}

func sumTracked(n *node) int {
	if n.kind == KindCategory {
		n.tracked = 0
		for _, child := range n.children {
			n.tracked += sumTracked(child)
		}
	}
	return n.tracked
}

// TotalMinutes is all time tracked across the plan.
func (p *Plan) TotalMinutes() int {
	total := 0
	for _, root := range p.roots {
		total += root.tracked
	}
	return total
}

// Scope narrows statistics to one category (including everything below it) or to one
// activity. The zero value means everything. Set at most one of the two fields.
type Scope struct {
	CategoryID *int64
	ActivityID *int64
}

// IsSet reports whether the scope restricts anything.
func (s Scope) IsSet() bool { return s.CategoryID != nil || s.ActivityID != nil }

// ActivityIDs lists the activities inside the scope. It returns nil for the zero scope and
// ErrCategoryNotFound / ErrActivityNotFound for unknown ids. A known category without any
// activity yields an empty, non-nil list.
func (p *Plan) ActivityIDs(scope Scope) ([]int64, error) {
	switch {
	case scope.ActivityID != nil:
		n, ok := p.activities[*scope.ActivityID]
		if !ok {
			return nil, ErrActivityNotFound
		}
		return []int64{n.id}, nil
	case scope.CategoryID != nil:
		n, ok := p.categories[*scope.CategoryID]
		if !ok {
			return nil, ErrCategoryNotFound
		}
		ids := []int64{}
		var collect func(*node)
		collect = func(n *node) {
			if n.kind == KindActivity {
				ids = append(ids, n.id)
			}
			for _, child := range n.children {
				collect(child)
			}
		}
		collect(n)
		return ids, nil
	}
	return nil, nil
}

// Options narrows a recommendation.
type Options struct {
	// AvailableMinutes is the length of the free time slot. Zero means unlimited.
	AvailableMinutes int
	// CategoryID restricts the choice to a category and everything below it.
	CategoryID *int64
	// Skip lists activities the user already declined.
	Skip map[int64]bool
}

// Recommendation is the activity to do next.
type Recommendation struct {
	Activity activity.Activity `json:"activity"`
	// Path lists the category names from the top level down to the activity's category.
	Path []string `json:"path"`
}

// Recommend picks the activity that is furthest behind its target. Starting at the top it
// repeatedly chooses, among the eligible siblings, the one with the least time logged
// relative to its weight, and descends into it. Ties go to the heavier weight, then to the
// name. ok is false when nothing fits.
func (p *Plan) Recommend(o Options) (rec Recommendation, ok bool, err error) {
	candidates := p.roots
	if o.CategoryID != nil {
		scope, found := p.categories[*o.CategoryID]
		if !found {
			return Recommendation{}, false, ErrCategoryNotFound
		}
		candidates = scope.children
	}

	chosen := pick(candidates, o)
	if chosen == nil {
		return Recommendation{}, false, nil
	}
	return Recommendation{Activity: chosen.activity, Path: pathOf(chosen)}, true, nil
}

func pick(nodes []*node, o Options) *node {
	var best *node
	for _, n := range nodes {
		if eligible(n, o) && (best == nil || behind(n, best)) {
			best = n
		}
	}
	if best == nil || best.kind == KindActivity {
		return best
	}
	return pick(best.children, o)
}

// eligible reports whether the node is, or contains, an activity that can be done now.
func eligible(n *node, o Options) bool {
	if n.kind == KindActivity {
		fits := o.AvailableMinutes <= 0 || n.activity.MinimumMinutes <= o.AvailableMinutes
		return fits && !o.Skip[n.id]
	}
	for _, child := range n.children {
		if eligible(child, o) {
			return true
		}
	}
	return false
}

// behind reports whether a is further behind its target than b.
func behind(a, b *node) bool {
	ratioA, ratioB := float64(a.tracked)/a.weight, float64(b.tracked)/b.weight
	if math.Abs(ratioA-ratioB) > epsilon {
		return ratioA < ratioB
	}
	if a.weight != b.weight {
		return a.weight > b.weight
	}
	return less(a, b)
}

func pathOf(n *node) []string {
	path := []string{}
	for p := n.parent; p != nil; p = p.parent {
		path = append([]string{p.name}, path...)
	}
	return path
}

// Share compares how much of the total time a node should get with what it got.
type Share struct {
	Kind        Kind    `json:"kind"`
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Depth       int     `json:"depth"`
	Minutes     int     `json:"minutes"`
	TargetShare float64 `json:"targetShare"` // 0..1
	ActualShare float64 `json:"actualShare"` // 0..1
}

// Shares lists every node depth-first (siblings sorted by name) with its target and actual share.
func (p *Plan) Shares() []Share {
	total := p.TotalMinutes()
	var shares []Share

	var walk func(nodes []*node, depth int, parentTarget float64)
	walk = func(nodes []*node, depth int, parentTarget float64) {
		weightSum := 0.0
		for _, n := range nodes {
			weightSum += n.weight
		}
		for _, n := range nodes {
			target := parentTarget * n.weight / weightSum
			actual := 0.0
			if total > 0 {
				actual = float64(n.tracked) / float64(total)
			}
			shares = append(shares, Share{
				Kind: n.kind, ID: n.id, Name: n.name, Depth: depth,
				Minutes: n.tracked, TargetShare: target, ActualShare: actual,
			})
			walk(n.children, depth+1, target)
		}
	}
	walk(p.roots, 0, 1)
	return shares
}

func derefOr(v *int64, fallback int64) int64 {
	if v == nil {
		return fallback
	}
	return *v
}
