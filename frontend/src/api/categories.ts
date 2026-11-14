import { http } from "./http";

export interface Category {
  id: number;
  parentId: number | null;
  name: string;
  multiplier: number;
  createdAt: string;
}

export interface NewCategory {
  parentId: number | null;
  name: string;
  multiplier: number;
}

/** Fields to change; omitted fields stay as they are. `parentId: null` moves to the top level. */
export type CategoryPatch = Partial<NewCategory>;

export const listCategories = () => http.get<Category[]>("/categories");

export const createCategory = (input: NewCategory) => http.post<Category>("/categories", input);

export const updateCategory = (id: number, patch: CategoryPatch) =>
  http.patch<Category>(`/categories/${id}`, patch);

export const deleteCategory = (id: number) => http.delete(`/categories/${id}`);
