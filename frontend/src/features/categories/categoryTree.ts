import type { Category } from "../../api/categories";

export interface CategoryNode extends Category {
  children: CategoryNode[];
}

/** Turns the flat API list into a forest. Orphans are treated as top-level. */
export function buildCategoryTree(categories: Category[]): CategoryNode[] {
  const nodes = new Map<number, CategoryNode>(
    categories.map((category) => [category.id, { ...category, children: [] }]),
  );

  const roots: CategoryNode[] = [];
  for (const node of nodes.values()) {
    const parent = node.parentId === null ? undefined : nodes.get(node.parentId);
    (parent ? parent.children : roots).push(node);
  }
  return roots;
}
