import { http } from "./http";
import type { TimeEntry } from "./timeEntries";

export interface Timer {
  activityId: number;
  startedAt: string;
}

/** The running timer, or null if none is running. */
export const getTimer = () => http.get<Timer | null>("/timer");

export const startTimer = (activityId: number) => http.post<Timer>("/timer", { activityId });

/** Stops the timer and logs the elapsed time. */
export const stopTimer = () => http.post<TimeEntry>("/timer/stop");

/** Discards the timer without logging time. */
export const cancelTimer = () => http.delete("/timer");
