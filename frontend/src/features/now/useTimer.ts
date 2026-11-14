import { useCallback, useEffect, useState } from "react";

import { errorMessage } from "../../api/http";
import * as timerApi from "../../api/timer";
import type { Timer } from "../../api/timer";
import type { TimeEntry } from "../../api/timeEntries";

interface UseTimer {
  /** The running timer, or null. */
  timer: Timer | null;
  loading: boolean;
  error: string | null;
  start: (activityId: number) => Promise<void>;
  /** Stops the timer and returns the logged entry. */
  stop: () => Promise<TimeEntry>;
  cancel: () => Promise<void>;
}

/** The server-side stopwatch: survives reloads and is shared between devices. */
export function useTimer(): UseTimer {
  const [timer, setTimer] = useState<Timer | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    timerApi
      .getTimer()
      .then(setTimer)
      .catch((e: unknown) => setError(errorMessage(e)))
      .finally(() => setLoading(false));
  }, []);

  const start = useCallback(async (activityId: number) => {
    setTimer(await timerApi.startTimer(activityId));
  }, []);

  const stop = useCallback(async () => {
    const entry = await timerApi.stopTimer();
    setTimer(null);
    return entry;
  }, []);

  const cancel = useCallback(async () => {
    await timerApi.cancelTimer();
    setTimer(null);
  }, []);

  return { timer, loading, error, start, stop, cancel };
}

/** Seconds since startedAt, refreshed every second. */
export function useElapsedSeconds(startedAt: string): number {
  const compute = useCallback(() => Math.max(0, (Date.now() - new Date(startedAt).getTime()) / 1000), [startedAt]);
  const [seconds, setSeconds] = useState(compute);

  useEffect(() => {
    setSeconds(compute());
    const interval = setInterval(() => setSeconds(compute()), 1000);
    return () => clearInterval(interval);
  }, [compute]);

  return seconds;
}
