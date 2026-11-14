import { request } from "./http";

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

export function listCategories(): Promise<Category[]> {
  return request<Category[]>("/categories");
}

export function createCategory(input: NewCategory): Promise<Category> {
  return request<Category>("/categories", {
    method: "POST",
    body: JSON.stringify(input),
  });
}
