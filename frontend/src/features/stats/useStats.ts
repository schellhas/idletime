import { useCallback, useEffect, useMemo, useState } from "react";

import { errorMessage } from "../../api/http";
import { getStats } from "../../api/progress";
import type { Stats } from "../../api/progress";
import { parseScopeKey } from "./scope";
import type { ScopeKey } from "./scope";

interface UseStats {
  stats: Stats | null;
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
}

/** Statistics for the last `days` days, optionally for one category or activity. */
export function useStats(days: number, scopeKey: ScopeKey): UseStats {
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const scope = useMemo(() => parseScopeKey(scopeKey), [scopeKey]);

  const reload = useCallback(async () => {
    try {
      setStats(await getStats(days, scope));
      setError(null);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  }, [days, scope]);

  useEffect(() => {
    setLoading(true);
    void reload();
  }, [reload]);

  return { stats, loading, error, reload };
}
