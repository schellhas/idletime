import type { Activity } from "../../api/activities";
import type { Category } from "../../api/categories";
import { categoryOptions } from "../library/libraryTree";
import { activityScopeKey, categoryScopeKey } from "./scope";
import type { ScopeKey } from "./scope";

interface ScopeSelectProps {
  value: ScopeKey;
  categories: Category[];
  activities: Activity[];
  onChange: (key: ScopeKey) => void;
}

/** Chooses what the daily chart shows: everything, one category or one activity. */
export function ScopeSelect({ value, categories, activities, onChange }: ScopeSelectProps) {
  const categoryName = (id: number) => categories.find((c) => c.id === id)?.name ?? "";

  return (
    <label className="field">
      <span>Show time for</span>
      <select value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="">Everything</option>
        <optgroup label="Categories">
          {categoryOptions(categories).map((option) => (
            <option key={option.id} value={categoryScopeKey(option.id)}>
              {option.label}
            </option>
          ))}
        </optgroup>
        <optgroup label="Activities">
          {activities.map((activity) => (
            <option key={activity.id} value={activityScopeKey(activity.id)}>
              {activity.name} ({categoryName(activity.categoryId)})
            </option>
          ))}
        </optgroup>
      </select>
    </label>
  );
}

/** Human-readable name of a scope, for chart titles. Empty for "everything". */
export function scopeLabel(key: ScopeKey, categories: Category[], activities: Activity[]): string {
  const id = Number(key.split(":")[1]);
  if (key.startsWith("c:")) {
    return categoryOptions(categories).find((option) => option.id === id)?.label ?? "";
  }
  if (key.startsWith("a:")) {
    return activities.find((activity) => activity.id === id)?.name ?? "";
  }
  return "";
}
