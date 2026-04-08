import { useEffect, useState } from 'react';

export function useTimerState() {
  const [timerState, setTimerState] = useState({ activityId: null, startedAt: null });
  const [timerNow, setTimerNow] = useState(Date.now());

  useEffect(() => {
    if (!timerState.startedAt) {
      return undefined;
    }

    const intervalId = window.setInterval(() => {
      setTimerNow(Date.now());
    }, 1000);

    return () => window.clearInterval(intervalId);
  }, [timerState.startedAt]);

  return {
    timerState,
    setTimerState,
    timerNow,
    setTimerNow,
  };
}
