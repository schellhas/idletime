import { useCallback, useEffect, useMemo, useState } from "react";

import * as activitiesApi from "../../api/activities";
import type { Activity, ActivityPatch, NewActivity } from "../../api/activities";
import * as categoriesApi from "../../api/categories";
import type { Category, CategoryPatch, NewCategory } from "../../api/categories";
import { errorMessage } from "../../api/http";
import { buildLibraryTree } from "./libraryTree";
import type { LibraryNode } from "./libraryTree";

export interface Library {
  categories: Category[];
  activities: Activity[];
  tree: LibraryNode[];
  loading: boolean;
  error: string | null;
  /** Re-reads everything, e.g. after time was logged. */
  reload: () => Promise<void>;
  createCategory: (input: NewCategory) => Promise<void>;
  updateCategory: (id: number, patch: CategoryPatch) => Promise<void>;
  removeCategory: (id: number) => Promise<void>;
  createActivity: (input: NewActivity) => Promise<void>;
  updateActivity: (id: number, patch: ActivityPatch) => Promise<void>;
  removeActivity: (id: number) => Promise<void>;
}

/** The user's categories and activities, with write operations that refresh the data. */
export function useLibrary(): Library {
  const [categories, setCategories] = useState<Category[]>([]);
  const [activities, setActivities] = useState<Activity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    try {
      const [loadedCategories, loadedActivities] = await Promise.all([
        categoriesApi.listCategories(),
        activitiesApi.listActivities(),
      ]);
      setCategories(loadedCategories);
      setActivities(loadedActivities);
      setError(null);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  return useMemo(() => {
    /** Runs a write operation and then refreshes; errors reach the caller's form. */
    const write =
      <Args extends unknown[]>(operation: (...args: Args) => Promise<unknown>) =>
      async (...args: Args) => {
        await operation(...args);
        await reload();
      };

    return {
      categories,
      activities,
      tree: buildLibraryTree(categories, activities),
      loading,
      error,
      reload,
      createCategory: write(categoriesApi.createCategory),
      updateCategory: write(categoriesApi.updateCategory),
      removeCategory: write(categoriesApi.deleteCategory),
      createActivity: write(activitiesApi.createActivity),
      updateActivity: write(activitiesApi.updateActivity),
      removeActivity: write(activitiesApi.deleteActivity),
    };
  }, [categories, activities, loading, error, reload]);
}
