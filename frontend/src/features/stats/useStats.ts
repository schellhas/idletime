import { useCallback, useEffect, useState } from "react";

import { errorMessage } from "../../api/http";
import { getStats } from "../../api/progress";
import type { Stats } from "../../api/progress";

interface UseStats {
  stats: Stats | null;
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
}

/** Statistics for the last `days` days; reloads when `days` changes. */
export function useStats(days: number): UseStats {
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    try {
      setStats(await getStats(days));
      setError(null);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  }, [days]);

  useEffect(() => {
    setLoading(true);
    void reload();
  }, [reload]);

  return { stats, loading, error, reload };
}
