import { useCallback, useEffect, useState } from "react";

import { errorMessage } from "../../api/http";
import { getRecommendation } from "../../api/progress";
import type { Recommendation } from "../../api/progress";

const DEBOUNCE_MS = 250;

export interface Slot {
  /** Free time in minutes; 0 means unlimited. */
  availableMinutes: number;
  /** Restricts the choice to this category and everything below it. */
  categoryId: number | undefined;
}

type State =
  | { status: "loading" }
  | { status: "found"; recommendation: Recommendation }
  | { status: "none" }
  | { status: "error"; message: string };

interface UseRecommendation {
  state: State;
  /** Hides the current suggestion and shows the next best one. */
  skip: (activityId: number) => void;
  /** Forgets all skips. */
  resetSkips: () => void;
  skippedCount: number;
  /** Fetches a fresh suggestion, e.g. after time was logged. */
  refresh: () => void;
}

/** Suggests what to do in the given time slot and refreshes whenever the slot changes. */
export function useRecommendation({ availableMinutes, categoryId }: Slot): UseRecommendation {
  const [state, setState] = useState<State>({ status: "loading" });
  const [skipped, setSkipped] = useState<number[]>([]);
  const [refreshCount, setRefreshCount] = useState(0);

  // A different slot is a different question, so earlier skips no longer apply.
  useEffect(() => setSkipped([]), [availableMinutes, categoryId]);

  useEffect(() => {
    let current = true; // ignore answers to outdated questions
    setState({ status: "loading" });

    const timer = setTimeout(() => {
      getRecommendation({ availableMinutes, categoryId, skip: skipped })
        .then((recommendation) => {
          if (current) {
            setState(recommendation ? { status: "found", recommendation } : { status: "none" });
          }
        })
        .catch((e: unknown) => {
          if (current) {
            setState({ status: "error", message: errorMessage(e) });
          }
        });
    }, DEBOUNCE_MS);

    return () => {
      current = false;
      clearTimeout(timer);
    };
  }, [availableMinutes, categoryId, skipped, refreshCount]);

  const skip = useCallback((activityId: number) => setSkipped((ids) => [...ids, activityId]), []);
  const resetSkips = useCallback(() => setSkipped([]), []);
  const refresh = useCallback(() => setRefreshCount((n) => n + 1), []);

  return { state, skip, resetSkips, skippedCount: skipped.length, refresh };
}
