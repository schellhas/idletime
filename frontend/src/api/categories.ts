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

export const listCategories = () => http.get<Category[]>("/categories");

export const createCategory = (input: NewCategory) => http.post<Category>("/categories", input);
