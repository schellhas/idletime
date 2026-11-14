import { useMemo } from "react";

import { Message } from "../../ui/Message";
import { CategoryForm } from "./CategoryForm";
import { CategoryList } from "./CategoryList";
import { buildCategoryTree } from "./categoryTree";
import { useCategories } from "./useCategories";

export function CategoriesPage() {
  const { categories, loading, error, create } = useCategories();
  const tree = useMemo(() => buildCategoryTree(categories), [categories]);

  return (
    <div className="page">
      <section className="card">
        <h2>Categories</h2>
        {loading && <Message>Loading…</Message>}
        {error && <Message kind="error">{error}</Message>}
        {!loading && !error && <CategoryList nodes={tree} />}
      </section>

      <CategoryForm categories={categories} onSubmit={create} />
    </div>
  );
}
