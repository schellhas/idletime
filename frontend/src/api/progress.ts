import type { Activity } from "./activities";
import { ApiError, http } from "./http";

export interface Recommendation {
  activity: Activity;
  /** Category names from the top level down to the activity's category. */
  path: string[];
}

export interface RecommendationOptions {
  /** Length of the free time slot; 0 or undefined means unlimited. */
  availableMinutes?: number;
  categoryId?: number;
  /** Activities the user already declined. */
  skip?: number[];
}

export function recommendationQuery({ availableMinutes, categoryId, skip }: RecommendationOptions): string {
  const params = new URLSearchParams();
  if (availableMinutes) {
    params.set("availableMinutes", String(availableMinutes));
  }
  if (categoryId !== undefined) {
    params.set("categoryId", String(categoryId));
  }
  if (skip && skip.length > 0) {
    params.set("skip", skip.join(","));
  }
  const query = params.toString();
  return query === "" ? "" : `?${query}`;
}

/** The activity to do next, or null when nothing fits. */
export async function getRecommendation(options: RecommendationOptions): Promise<Recommendation | null> {
  try {
    return await http.get<Recommendation>(`/recommendation${recommendationQuery(options)}`);
  } catch (error) {
    if (error instanceof ApiError && error.code === "no_recommendation") {
      return null;
    }
    throw error;
  }
}

export interface Share {
  kind: "category" | "activity";
  id: number;
  name: string;
  depth: number;
  minutes: number;
  /** Share of all time this node should get, 0..1. */
  targetShare: number;
  /** Share of all time this node actually got, 0..1. */
  actualShare: number;
}

export interface DayMinutes {
  /** YYYY-MM-DD in the requested time zone. */
  date: string;
  minutes: number;
}

export interface Stats {
  totalMinutes: number;
  periodMinutes: number;
  days: number;
  shares: Share[];
  daily: DayMinutes[];
}

/** Restricts the daily series to one category (with everything below it) or one activity. */
export interface StatsScope {
  categoryId?: number;
  activityId?: number;
}

export function getStats(days: number, scope: StatsScope = {}): Promise<Stats> {
  const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const params = new URLSearchParams({ days: String(days), tz: timeZone });
  if (scope.categoryId !== undefined) {
    params.set("categoryId", String(scope.categoryId));
  }
  if (scope.activityId !== undefined) {
    params.set("activityId", String(scope.activityId));
  }
  return http.get<Stats>(`/stats?${params.toString()}`);
}
