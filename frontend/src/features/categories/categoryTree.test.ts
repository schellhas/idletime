import { describe, expect, it } from "vitest";

import type { Category } from "../../api/categories";
import { buildCategoryTree } from "./categoryTree";

const make = (id: number, parentId: number | null): Category => ({
  id,
  parentId,
  name: `c${id}`,
  multiplier: 1,
  createdAt: "2026-01-01T00:00:00Z",
});

describe("buildCategoryTree", () => {
  it("nests children below their parent", () => {
    const tree = buildCategoryTree([make(1, null), make(2, 1), make(3, 2), make(4, null)]);

    expect(tree.map((node) => node.id)).toEqual([1, 4]);
    expect(tree[0]?.children[0]?.children[0]?.id).toBe(3);
  });

  it("treats categories with an unknown parent as top-level", () => {
    expect(buildCategoryTree([make(5, 99)]).map((node) => node.id)).toEqual([5]);
  });

  it("returns an empty forest for no input", () => {
    expect(buildCategoryTree([])).toEqual([]);
  });
});
