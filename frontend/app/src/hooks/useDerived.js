import { useMemo } from 'react';
import { clampPercent } from '../utils/appUtils';

export function useDerived(
  categories,
  activities,
  dragMultipliers,
  selectedRecommendationCategoryIds,
) {
  const categoryById = useMemo(
    () => Object.fromEntries(categories.map((category) => [category.id, category])),
    [categories],
  );

  const activityById = useMemo(
    () => Object.fromEntries(activities.map((activity) => [activity.id, activity])),
    [activities],
  );

  const activitiesByCategoryId = useMemo(() => {
    const grouped = {};
    activities.forEach((activity) => {
      const key = String(activity.category_id);
      if (!grouped[key]) {
        grouped[key] = [];
      }
      grouped[key].push(activity);
    });
    return grouped;
  }, [activities]);

  const categoriesByParent = useMemo(() => {
    const grouped = { root: [] };
    categories.forEach((category) => {
      const parentKey = category.parent_id
        ? String(category.parent_id)
        : 'root';
      if (!grouped[parentKey]) {
        grouped[parentKey] = [];
      }
      grouped[parentKey].push(category);
    });
    for (const key of Object.keys(grouped)) {
      grouped[key].sort((a, b) => String(a.name ?? '').localeCompare(String(b.name ?? '')));
    }
    return grouped;
  }, [categories]);

  const categoriesInTreeOrder = useMemo(() => {
    const result = [];
    function walk(parentId) {
      const children = categoriesByParent[String(parentId)] ?? [];
      for (const child of children) {
        result.push(child);
        walk(child.id);
      }
    }
    const rootCat = categories.find((cat) => String(cat.name ?? '').trim().toLowerCase() === 'root')
      ?? categories.find((cat) => !cat.parent_id);
    if (rootCat) walk(rootCat.id);
    return result;
  }, [categories, categoriesByParent]);

  const selectedRecommendationCategoryIdSet = useMemo(() => {
    const selected = new Set();
    const stack = [...selectedRecommendationCategoryIds];
    while (stack.length > 0) {
      const currentID = String(stack.pop());
      if (selected.has(currentID)) {
        continue;
      }
      selected.add(currentID);
      const children = categoriesByParent[currentID] ?? [];
      children.forEach((child) => stack.push(String(child.id)));
    }
    return selected;
  }, [selectedRecommendationCategoryIds, categoriesByParent]);

  const overallTrackedMinutes = useMemo(
    () => activities.reduce((sum, activity) => sum + Number(activity.tracked_minutes ?? 0), 0),
    [activities],
  );

  const subtreeTrackedMinutesByCategoryId = useMemo(() => {
    const cache = {};
    function sumForCategory(categoryId) {
      if (cache[categoryId] !== undefined) {
        return cache[categoryId];
      }
      const ownMinutes = (activitiesByCategoryId[String(categoryId)] ?? [])
        .reduce((sum, activity) => sum + Number(activity.tracked_minutes ?? 0), 0);
      const childMinutes = (categoriesByParent[String(categoryId)] ?? [])
        .reduce((sum, child) => sum + sumForCategory(child.id), 0);
      cache[categoryId] = ownMinutes + childMinutes;
      return cache[categoryId];
    }
    categories.forEach((category) => {
      sumForCategory(category.id);
    });
    return cache;
  }, [activitiesByCategoryId, categories, categoriesByParent]);

  const categoryProgress = useMemo(() => {
    const maxCategoryMinutes = Math.max(
      1,
      ...categories.map((category) => (
        activities
          .filter((activity) => activity.category_id === category.id)
          .reduce((sum, activity) => sum + Number(activity.tracked_minutes ?? 0), 0)
      )),
    );
    return categories
      .map((category) => {
        const categoryActivities = activities
          .filter((activity) => activity.category_id === category.id)
          .sort((left, right) => Number(right.tracked_minutes ?? 0) - Number(left.tracked_minutes ?? 0));
        const directTrackedMinutes = categoryActivities.reduce(
          (sum, activity) => sum + Number(activity.tracked_minutes ?? 0),
          0,
        );
        const totalTrackedMinutes = Number(subtreeTrackedMinutesByCategoryId[category.id] ?? directTrackedMinutes);
        const maxActivityMinutes = Math.max(
          1,
          ...categoryActivities.map((activity) => Number(activity.tracked_minutes ?? 0)),
        );
        const totalWeight = Number(category.multiplier ?? 1)
          * (categoryActivities.reduce((sum, activity) => sum + Number(activity.multiplier ?? 1), 0) || 1);
        return {
          ...category,
          directTrackedMinutes,
          totalTrackedMinutes,
          totalWeight,
          normalizedProgress: totalWeight > 0 ? totalTrackedMinutes / totalWeight : totalTrackedMinutes,
          shareOfAll: overallTrackedMinutes > 0
            ? Math.round((totalTrackedMinutes / overallTrackedMinutes) * 100)
            : 0,
          totalPercent: clampPercent(totalTrackedMinutes, maxCategoryMinutes),
          activities: categoryActivities.map((activity) => ({
            ...activity,
            trackedDisplayMinutes: Number(activity.tracked_minutes ?? 0),
            percent: clampPercent(activity.tracked_minutes, maxActivityMinutes),
          })),
        };
      })
      .sort(
        (left, right) => right.totalTrackedMinutes - left.totalTrackedMinutes || left.name.localeCompare(right.name),
      );
  }, [activities, categories, overallTrackedMinutes, subtreeTrackedMinutesByCategoryId]);

  return {
    categoryById,
    activityById,
    activitiesByCategoryId,
    categoriesByParent,
    categoriesInTreeOrder,
    selectedRecommendationCategoryIdSet,
    overallTrackedMinutes,
    subtreeTrackedMinutesByCategoryId,
    categoryProgress,
  };
}
