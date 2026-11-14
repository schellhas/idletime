import { http } from "./http";

export interface TimeEntry {
  id: number;
  activityId: number;
  minutes: number;
  note: string;
  /** When the activity took place (ISO timestamp). */
  loggedAt: string;
  createdAt: string;
}

export interface NewTimeEntry {
  activityId: number;
  minutes: number;
  note?: string;
}

export const listTimeEntries = (limit: number) =>
  http.get<TimeEntry[]>(`/time-entries?limit=${limit}`);

export const createTimeEntry = (input: NewTimeEntry) => http.post<TimeEntry>("/time-entries", input);

export const deleteTimeEntry = (id: number) => http.delete(`/time-entries/${id}`);
