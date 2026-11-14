package integration_test

import (
	"errors"
	"testing"

	"github.com/schellhas/idletime/backend/internal/activity"
	"github.com/schellhas/idletime/backend/internal/category"
	"github.com/schellhas/idletime/backend/internal/dbtest"
)

func TestCategoryRepository(t *testing.T) {
	db := dbtest.New(t)
	repo := category.NewPostgresRepository(db.Pool)
	alice := dbtest.CreateUser(t, db.Pool, "alice")
	bob := dbtest.CreateUser(t, db.Pool, "bob")

	sports := mustCategory(t, repo, alice.ID, "Sports", nil)
	outdoors := mustCategory(t, repo, alice.ID, "Outdoors", &sports.ID)

	t.Run("lists only the owner's categories", func(t *testing.T) {
		mustCategory(t, repo, bob.ID, "Bobs", nil)
		got, err := repo.List(ctx, alice.ID)
		if err != nil || len(got) != 2 {
			t.Fatalf("List() = %+v, %v; want Alice's two categories", got, err)
		}
		// Ordered by name, so "Outdoors" comes before "Sports".
		if got[0].Name != "Outdoors" || got[0].ParentID == nil || *got[0].ParentID != sports.ID ||
			got[1].Name != "Sports" || got[1].ParentID != nil {
			t.Errorf("order or parent links are wrong: %+v", got)
		}
	})

	t.Run("sibling names are unique, ignoring case, also at the top level", func(t *testing.T) {
		_, err := repo.Create(ctx, category.Category{UserID: alice.ID, Name: "SPORTS", Multiplier: 1})
		if !errors.Is(err, category.ErrDuplicateName) {
			t.Errorf("top level: error = %v, want ErrDuplicateName", err)
		}
		_, err = repo.Create(ctx, category.Category{UserID: alice.ID, ParentID: &sports.ID, Name: "outdoors", Multiplier: 1})
		if !errors.Is(err, category.ErrDuplicateName) {
			t.Errorf("nested: error = %v, want ErrDuplicateName", err)
		}
		// The same name below another parent, or for another user, is fine.
		mustCategory(t, repo, alice.ID, "Outdoors", nil)
		mustCategory(t, repo, bob.ID, "Sports", nil)
	})

	t.Run("a category cannot be nested below another user's category", func(t *testing.T) {
		_, err := repo.Create(ctx, category.Category{UserID: bob.ID, ParentID: &sports.ID, Name: "Intruder", Multiplier: 1})
		if !errors.Is(err, category.ErrParentNotFound) {
			t.Errorf("error = %v, want ErrParentNotFound", err)
		}
	})

	t.Run("update changes fields and respects ownership", func(t *testing.T) {
		moved := outdoors
		moved.Name, moved.Multiplier, moved.ParentID = "Open air", 2.5, nil
		got, err := repo.Update(ctx, moved)
		if err != nil || got.Name != "Open air" || got.Multiplier != 2.5 || got.ParentID != nil {
			t.Fatalf("Update() = %+v, %v", got, err)
		}

		stolen := got
		stolen.UserID = bob.ID
		if _, err := repo.Update(ctx, stolen); !errors.Is(err, category.ErrNotFound) {
			t.Errorf("foreign update: error = %v, want ErrNotFound", err)
		}
		bobParent := got
		bobParent.ParentID = ptr(mustCategory(t, repo, bob.ID, "Bobs parent", nil).ID)
		if _, err := repo.Update(ctx, bobParent); !errors.Is(err, category.ErrParentNotFound) {
			t.Errorf("moving below a foreign parent: error = %v, want ErrParentNotFound", err)
		}
	})

	t.Run("delete cascades to subcategories and activities, and respects ownership", func(t *testing.T) {
		activities := activity.NewPostgresRepository(db.Pool)
		parent := mustCategory(t, repo, alice.ID, "Parent", nil)
		child := mustCategory(t, repo, alice.ID, "Child", &parent.ID)
		mustActivity(t, activities, alice.ID, child.ID, "Deep activity")

		if err := repo.Delete(ctx, bob.ID, parent.ID); !errors.Is(err, category.ErrNotFound) {
			t.Errorf("foreign delete: error = %v, want ErrNotFound", err)
		}
		if err := repo.Delete(ctx, alice.ID, parent.ID); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if n := count(t, db.Pool, "SELECT count(*) FROM categories WHERE id = $1", child.ID); n != 0 {
			t.Error("the subcategory survived")
		}
		if n := count(t, db.Pool, "SELECT count(*) FROM activities WHERE name = 'Deep activity'"); n != 0 {
			t.Error("the activity survived")
		}
	})
}

func TestActivityRepository(t *testing.T) {
	db := dbtest.New(t)
	categories := category.NewPostgresRepository(db.Pool)
	repo := activity.NewPostgresRepository(db.Pool)
	alice := dbtest.CreateUser(t, db.Pool, "alice")
	bob := dbtest.CreateUser(t, db.Pool, "bob")

	sports := mustCategory(t, categories, alice.ID, "Sports", nil)
	care := mustCategory(t, categories, alice.ID, "Care", nil)
	bobs := mustCategory(t, categories, bob.ID, "Bobs", nil)

	swimming, err := repo.Create(ctx, activity.Activity{UserID: alice.ID, CategoryID: sports.ID, Name: "Swimming", Multiplier: 2, MinimumMinutes: 20})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("stores all fields and starts without tracked time", func(t *testing.T) {
		if swimming.Multiplier != 2 || swimming.MinimumMinutes != 20 || swimming.TrackedMinutes != 0 || swimming.CategoryID != sports.ID {
			t.Errorf("got %+v", swimming)
		}
	})

	t.Run("lists and fetches only the owner's activities", func(t *testing.T) {
		mustActivity(t, repo, bob.ID, bobs.ID, "Bobs activity")
		own, err := repo.List(ctx, alice.ID)
		if err != nil || len(own) != 1 || own[0].ID != swimming.ID {
			t.Errorf("List() = %+v, %v", own, err)
		}
		if _, err := repo.Get(ctx, bob.ID, swimming.ID); !errors.Is(err, activity.ErrNotFound) {
			t.Errorf("foreign Get: error = %v, want ErrNotFound", err)
		}
	})

	t.Run("names are unique per category, ignoring case", func(t *testing.T) {
		_, err := repo.Create(ctx, activity.Activity{UserID: alice.ID, CategoryID: sports.ID, Name: "SWIMMING", Multiplier: 1})
		if !errors.Is(err, activity.ErrDuplicateName) {
			t.Errorf("error = %v, want ErrDuplicateName", err)
		}
		mustActivity(t, repo, alice.ID, care.ID, "Swimming") // another category is fine
	})

	t.Run("an activity cannot be placed in another user's category", func(t *testing.T) {
		_, err := repo.Create(ctx, activity.Activity{UserID: alice.ID, CategoryID: bobs.ID, Name: "Sneaky", Multiplier: 1})
		if !errors.Is(err, activity.ErrCategoryNotFound) {
			t.Errorf("create: error = %v, want ErrCategoryNotFound", err)
		}
		moved := swimming
		moved.CategoryID = bobs.ID
		if _, err := repo.Update(ctx, moved); !errors.Is(err, activity.ErrCategoryNotFound) {
			t.Errorf("update: error = %v, want ErrCategoryNotFound", err)
		}
	})

	t.Run("update and delete respect ownership", func(t *testing.T) {
		changed := swimming
		changed.Name, changed.CategoryID, changed.MinimumMinutes = "Laps", care.ID, 0
		got, err := repo.Update(ctx, changed)
		if err != nil || got.Name != "Laps" || got.CategoryID != care.ID || got.MinimumMinutes != 0 {
			t.Fatalf("Update() = %+v, %v", got, err)
		}

		foreign := got
		foreign.UserID = bob.ID
		if _, err := repo.Update(ctx, foreign); !errors.Is(err, activity.ErrNotFound) {
			t.Errorf("foreign update: error = %v, want ErrNotFound", err)
		}
		if err := repo.Delete(ctx, bob.ID, got.ID); !errors.Is(err, activity.ErrNotFound) {
			t.Errorf("foreign delete: error = %v, want ErrNotFound", err)
		}
		if err := repo.Delete(ctx, alice.ID, got.ID); err != nil {
			t.Errorf("Delete() error = %v", err)
		}
	})
}
