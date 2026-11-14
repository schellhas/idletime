import { http } from "./http";

export interface Activity {
  id: number;
  categoryId: number;
  name: string;
  multiplier: number;
  /** Shortest time slot in which the activity makes sense. */
  minimumMinutes: number;
  /** Total time logged for the activity. */
  trackedMinutes: number;
  createdAt: string;
}

export interface NewActivity {
  categoryId: number;
  name: string;
  multiplier: number;
  minimumMinutes: number;
}

export type ActivityPatch = Partial<NewActivity>;

export const listActivities = () => http.get<Activity[]>("/activities");

export const createActivity = (input: NewActivity) => http.post<Activity>("/activities", input);

export const updateActivity = (id: number, patch: ActivityPatch) =>
  http.patch<Activity>(`/activities/${id}`, patch);

export const deleteActivity = (id: number) => http.delete(`/activities/${id}`);
