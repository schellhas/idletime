import type { CategoryNode } from "./categoryTree";

interface CategoryListProps {
  nodes: CategoryNode[];
}

export function CategoryList({ nodes }: CategoryListProps) {
  if (nodes.length === 0) {
    return <p className="message">No categories yet. Add your first one.</p>;
  }

  return (
    <ul className="tree">
      {nodes.map((node) => (
        <li key={node.id}>
          <div className="tree__row">
            <span>{node.name}</span>
            <span className="badge">×{node.multiplier}</span>
          </div>
          {node.children.length > 0 && <CategoryList nodes={node.children} />}
        </li>
      ))}
    </ul>
  );
}
