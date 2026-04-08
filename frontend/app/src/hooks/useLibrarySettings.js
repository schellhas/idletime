import { useState } from 'react';

export function useLibrarySettings() {
  const [libraryCreateTarget, setLibraryCreateTarget] = useState(null);
  const [libraryCreateDraft, setLibraryCreateDraft] = useState({ name: '' });
  const [librarySettingsTarget, setLibrarySettingsTarget] = useState(null);
  const [librarySettingsDraft, setLibrarySettingsDraft] = useState({});
  const [deleteConfirmTarget, setDeleteConfirmTarget] = useState(null);
  const [expandedLibraryCategories, setExpandedLibraryCategories] = useState({});
  const [expandedRecommendationCategories, setExpandedRecommendationCategories] = useState({});
  const [selectedRecommendationCategoryIds, setSelectedRecommendationCategoryIds] = useState([]);
  const [recommendationCategoryPickerOpen, setRecommendationCategoryPickerOpen] = useState(false);

  function openLibrarySettings(type, item) {
    setLibrarySettingsTarget({ type, item });

    if (type === 'category') {
      setLibrarySettingsDraft({
        name: item.name,
        multiplier: String(item.multiplier ?? 1),
        parentId: item.parent_id ? String(item.parent_id) : '',
      });
      return;
    }

    setLibrarySettingsDraft({
      name: item.name,
      multiplier: String(item.multiplier ?? 1),
      minimumMinutes: String(item.minimum_minutes ?? 0),
      trackedMinutes: String(item.tracked_minutes ?? 0),
      categoryId: String(item.category_id ?? ''),
    });
  }

  function closeLibrarySettings() {
    setLibrarySettingsTarget(null);
    setLibrarySettingsDraft({});
  }

  function openLibraryCreate(target) {
    setLibraryCreateDraft({ name: '' });
    setLibraryCreateTarget(target);
  }

  function closeLibraryCreate() {
    setLibraryCreateTarget(null);
    setLibraryCreateDraft({ name: '' });
  }

  function cancelDeleteLibraryTarget() {
    setDeleteConfirmTarget(null);
  }

  function toggleLibraryCategory(categoryId) {
    setExpandedLibraryCategories((current) => ({
      ...current,
      [categoryId]: !current[categoryId],
    }));
  }

  function expandAllLibraryCategories() {
    const expanded = {};
    setExpandedLibraryCategories(expanded);
  }

  function collapseAllLibraryCategories() {
    setExpandedLibraryCategories({});
  }

  function toggleRecommendationCategoryExpansion(categoryId) {
    setExpandedRecommendationCategories((current) => ({
      ...current,
      [categoryId]: !current[categoryId],
    }));
  }

  function expandAllRecommendationCategories(categories) {
    setExpandedRecommendationCategories(
      Object.fromEntries(categories.map((category) => [category.id, true])),
    );
  }

  function collapseAllRecommendationCategories(categories) {
    setExpandedRecommendationCategories(
      Object.fromEntries(categories.map((category) => [category.id, false])),
    );
  }

  return {
    libraryCreateTarget,
    setLibraryCreateTarget,
    libraryCreateDraft,
    setLibraryCreateDraft,
    librarySettingsTarget,
    setLibrarySettingsTarget,
    librarySettingsDraft,
    setLibrarySettingsDraft,
    deleteConfirmTarget,
    setDeleteConfirmTarget,
    expandedLibraryCategories,
    setExpandedLibraryCategories,
    expandedRecommendationCategories,
    setExpandedRecommendationCategories,
    selectedRecommendationCategoryIds,
    setSelectedRecommendationCategoryIds,
    recommendationCategoryPickerOpen,
    setRecommendationCategoryPickerOpen,
    openLibrarySettings,
    closeLibrarySettings,
    openLibraryCreate,
    closeLibraryCreate,
    cancelDeleteLibraryTarget,
    toggleLibraryCategory,
    expandAllLibraryCategories,
    collapseAllLibraryCategories,
    toggleRecommendationCategoryExpansion,
    expandAllRecommendationCategories,
    collapseAllRecommendationCategories,
  };
}
