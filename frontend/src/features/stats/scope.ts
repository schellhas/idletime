import type { StatsScope } from "../../api/progress";

/**
 * A scope as one string for <select> values: "" for everything, "c:3" for a category,
 * "a:7" for an activity.
 */
export type ScopeKey = string;

export const categoryScopeKey = (id: number): ScopeKey => `c:${id}`;
export const activityScopeKey = (id: number): ScopeKey => `a:${id}`;

export function parseScopeKey(key: ScopeKey): StatsScope {
  const [kind, rawId] = key.split(":");
  const id = Number(rawId);
  if (!Number.isInteger(id)) {
    return {};
  }
  if (kind === "c") {
    return { categoryId: id };
  }
  if (kind === "a") {
    return { activityId: id };
  }
  return {};
}
