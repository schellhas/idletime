import { useCallback, useEffect, useState } from "react";

import { createCategory, listCategories } from "../../api/categories";
import type { Category, NewCategory } from "../../api/categories";

interface UseCategories {
  categories: Category[];
  loading: boolean;
  error: string | null;
  create: (input: NewCategory) => Promise<void>;
}

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : "Something went wrong.";
}

export function useCategories(): UseCategories {
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    try {
      setCategories(await listCategories());
      setError(null);
    } catch (e) {
      setError(messageOf(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const create = useCallback(
    async (input: NewCategory) => {
      await createCategory(input);
      await reload();
    },
    [reload],
  );

  return { categories, loading, error, create };
}
