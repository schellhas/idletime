import { describe, expect, it } from "vitest";

import type { Activity } from "../../api/activities";
import type { Category } from "../../api/categories";
import { buildLibraryTree, categoryOptions, subtreeIds } from "./libraryTree";

const category = (id: number, parentId: number | null, name = `c${id}`): Category => ({
  id,
  parentId,
  name,
  multiplier: 1,
  createdAt: "2026-01-01T00:00:00Z",
});

const activity = (id: number, categoryId: number): Activity => ({
  id,
  categoryId,
  name: `a${id}`,
  multiplier: 1,
  minimumMinutes: 0,
  trackedMinutes: 0,
  createdAt: "2026-01-01T00:00:00Z",
});

describe("buildLibraryTree", () => {
  it("nests categories and attaches activities to their category", () => {
    const tree = buildLibraryTree(
      [category(1, null), category(2, 1), category(3, 2), category(4, null)],
      [activity(10, 2), activity(11, 2), activity(12, 4)],
    );

    expect(tree.map((node) => node.category.id)).toEqual([1, 4]);
    expect(tree[0]?.children[0]?.activities.map((a) => a.id)).toEqual([10, 11]);
    expect(tree[0]?.children[0]?.children[0]?.category.id).toBe(3);
    expect(tree[1]?.activities.map((a) => a.id)).toEqual([12]);
  });

  it("treats categories with an unknown parent as top-level and ignores orphan activities", () => {
    const tree = buildLibraryTree([category(5, 99)], [activity(1, 404)]);

    expect(tree.map((node) => node.category.id)).toEqual([5]);
    expect(tree[0]?.activities).toEqual([]);
  });

  it("returns an empty forest for no input", () => {
    expect(buildLibraryTree([], [])).toEqual([]);
  });
});

describe("categoryOptions", () => {
  const categories = [category(1, null, "Sports"), category(2, 1, "Outdoors"), category(3, null, "Care")];

  it("labels nested categories with their path", () => {
    expect(categoryOptions(categories).map((o) => o.label)).toEqual(["Sports", "Sports › Outdoors", "Care"]);
  });

  it("leaves out excluded categories together with everything below them", () => {
    expect(categoryOptions(categories, new Set([1])).map((o) => o.id)).toEqual([3]);
  });
});

describe("subtreeIds", () => {
  it("collects a category and all its descendants", () => {
    const categories = [category(1, null), category(2, 1), category(3, 2), category(4, null)];

    expect([...subtreeIds(categories, 1)].sort()).toEqual([1, 2, 3]);
    expect([...subtreeIds(categories, 4)]).toEqual([4]);
  });
});
