import type { Activity } from "../../api/activities";
import type { Category } from "../../api/categories";

export interface LibraryNode {
  category: Category;
  activities: Activity[];
  children: LibraryNode[];
}

/**
 * Turns the flat API lists into a forest of categories, each holding its activities.
 * Categories with an unknown parent are treated as top-level.
 */
export function buildLibraryTree(categories: Category[], activities: Activity[]): LibraryNode[] {
  const nodes = new Map<number, LibraryNode>(
    categories.map((category) => [category.id, { category, activities: [], children: [] }]),
  );

  for (const activity of activities) {
    nodes.get(activity.categoryId)?.activities.push(activity);
  }

  const roots: LibraryNode[] = [];
  for (const node of nodes.values()) {
    const parent = node.category.parentId === null ? undefined : nodes.get(node.category.parentId);
    (parent ? parent.children : roots).push(node);
  }
  return roots;
}

export interface CategoryOption {
  id: number;
  /** Full path such as "Sports › Outdoors", so equally named categories stay distinguishable. */
  label: string;
}

/** Flat, path-labelled list of categories for <select> elements, ordered like the tree. */
export function categoryOptions(categories: Category[], exclude: ReadonlySet<number> = new Set()): CategoryOption[] {
  const options: CategoryOption[] = [];
  const walk = (nodes: LibraryNode[], prefix: string) => {
    for (const node of nodes) {
      const label = prefix === "" ? node.category.name : `${prefix} › ${node.category.name}`;
      if (!exclude.has(node.category.id)) {
        options.push({ id: node.category.id, label });
        walk(node.children, label);
      }
    }
  };
  walk(buildLibraryTree(categories, []), "");
  return options;
}

/** The ids of a category and everything below it. A category cannot be moved into these. */
export function subtreeIds(categories: Category[], rootId: number): Set<number> {
  const ids = new Set<number>([rootId]);
  let grew = true;
  while (grew) {
    grew = false;
    for (const category of categories) {
      if (category.parentId !== null && ids.has(category.parentId) && !ids.has(category.id)) {
        ids.add(category.id);
        grew = true;
      }
    }
  }
  return ids;
}
