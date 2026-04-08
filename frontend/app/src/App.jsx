import { useEffect, useMemo, useState } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080';
const DEFAULT_CATEGORY_NAMES = new Set(['none', 'root']);

async function apiFetch(path, options = {}) {
  const { body, headers, ...rest } = options;
  const finalHeaders = {
    Accept: 'application/json',
    ...headers,
  };

  if (body !== undefined) {
    finalHeaders['Content-Type'] = 'application/json';
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: 'include',
    headers: finalHeaders,
    body,
    ...rest,
  });

  const raw = await response.text();
  let data = {};

  if (raw) {
    try {
      data = JSON.parse(raw);
    } catch {
      data = { raw };
    }
  }

  if (!response.ok) {
    throw new Error(data.error || data.message || `Request failed (${response.status})`);
  }

  return data;
}

function formatTimestamp(value) {
  if (!value) {
    return '—';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

function formatMinutes(value) {
  return `${Number(value ?? 0)} min`;
}

function currentValue(value) {
  return value ?? '';
}

function displayCategoryName(name) {
  const normalized = String(name ?? '').trim().toLowerCase();
  if (normalized === 'root' || normalized === 'none') {
    return 'Library';
  }
  return String(name ?? 'Library');
}

function displayCategoryPath(category, categoryById) {
  if (!category) {
    return 'Library';
  }

  const parts = [];
  const seen = new Set();
  let current = category;

  while (current && !seen.has(current.id)) {
    parts.unshift(displayCategoryName(current.name));
    seen.add(current.id);
    current = current.parent_id ? categoryById[current.parent_id] : null;
  }

  return parts.join(' / ');
}

function isDefaultCategory(category) {
  return DEFAULT_CATEGORY_NAMES.has(String(category?.name ?? '').trim().toLowerCase());
}

function clampPercent(value, maxValue) {
  const safeValue = Math.max(0, Number(value ?? 0));
  if (safeValue === 0) {
    return 0;
  }

  const safeMax = Math.max(1, Number(maxValue ?? 0));
  return Math.min(100, Math.max(8, Math.round((safeValue / safeMax) * 100)));
}

function sameIdList(left, right) {
  if (left.length !== right.length) {
    return false;
  }

  return left.every((value, index) => value === right[index]);
}

function formatTimerDuration(startedAt, now) {
  if (!startedAt) {
    return '00:00';
  }

  const totalSeconds = Math.max(0, Math.floor((now - startedAt) / 1000));
  const minutes = String(Math.floor(totalSeconds / 60)).padStart(2, '0');
  const seconds = String(totalSeconds % 60).padStart(2, '0');
  return `${minutes}:${seconds}`;
}

export default function App() {
  const [theme, setTheme] = useState(() => {
    const saved = window.localStorage.getItem('idletime_theme');
    if (saved === 'dark' || saved === 'light') {
      return saved;
    }
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  });
  const [bootstrapping, setBootstrapping] = useState(true);
  const [user, setUser] = useState(null);
  const [categories, setCategories] = useState([]);
  const [activities, setActivities] = useState([]);
  const [timeEntries, setTimeEntries] = useState([]);
  const [recommendation, setRecommendation] = useState(null);
  const [recommendationMessage, setRecommendationMessage] = useState('');
  const [authMode, setAuthMode] = useState('login');
  const [statusMessage, setStatusMessage] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const [verificationLink, setVerificationLink] = useState('');
  const [verificationHandled, setVerificationHandled] = useState(false);
  const [activeView, setActiveView] = useState('use');
  const [selectedRecommendationCategoryIds, setSelectedRecommendationCategoryIds] = useState([]);
  const [accountPanelOpen, setAccountPanelOpen] = useState(false);
  const [categoryEditDrafts, setCategoryEditDrafts] = useState({});
  const [activityEditDrafts, setActivityEditDrafts] = useState({});
  const [timerState, setTimerState] = useState({ activityId: null, startedAt: null });
  const [timerNow, setTimerNow] = useState(Date.now());
  const [hasRequestedRecommendation, setHasRequestedRecommendation] = useState(false);
  const [skippedRecommendationActivityIds, setSkippedRecommendationActivityIds] = useState([]);
  const [expandedLibraryCategories, setExpandedLibraryCategories] = useState({});
  const [recommendationCategoryPickerOpen, setRecommendationCategoryPickerOpen] = useState(false);
  const [expandedRecommendationCategories, setExpandedRecommendationCategories] = useState({});
  const [librarySettingsTarget, setLibrarySettingsTarget] = useState(null);
  const [librarySettingsDraft, setLibrarySettingsDraft] = useState({});

  const [loginForm, setLoginForm] = useState({
    identifier: '',
    password: '',
  });
  const [registerForm, setRegisterForm] = useState({
    username: '',
    email: '',
    password: '',
  });
  const [categoryForm, setCategoryForm] = useState({
    name: '',
    multiplier: '1',
  });
  const [activityForm, setActivityForm] = useState({
    categoryId: '',
    name: '',
    multiplier: '1',
    minimumMinutes: '0',
  });
  const [timeEntryForm, setTimeEntryForm] = useState({
    activityId: '',
    minutes: '30',
    note: '',
  });

  const categoryById = useMemo(
    () => Object.fromEntries(categories.map((category) => [category.id, category])),
    [categories],
  );
  const activityById = useMemo(
    () => Object.fromEntries(activities.map((activity) => [activity.id, activity])),
    [activities],
  );
  const rootCategory = useMemo(
    () => categories.find((category) => String(category.name ?? '').trim().toLowerCase() === 'root')
      ?? categories.find((category) => isDefaultCategory(category))
      ?? null,
    [categories],
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
      if (rootCategory && category.id === rootCategory.id) {
        return;
      }

      const parentKey = category.parent_id && categoryById[category.parent_id]
        ? String(category.parent_id)
        : (rootCategory ? String(rootCategory.id) : 'root');
      if (!grouped[parentKey]) {
        grouped[parentKey] = [];
      }
      grouped[parentKey].push(category);
    });
    return grouped;
  }, [categories, categoryById, rootCategory]);
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

        const totalTrackedMinutes = categoryActivities.reduce(
          (sum, activity) => sum + Number(activity.tracked_minutes ?? 0),
          0,
        );
        const maxActivityMinutes = Math.max(
          1,
          ...categoryActivities.map((activity) => Number(activity.tracked_minutes ?? 0)),
        );
        const totalWeight = Number(category.multiplier ?? 1)
          * (categoryActivities.reduce((sum, activity) => sum + Number(activity.multiplier ?? 1), 0) || 1);

        return {
          ...category,
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
  }, [activities, categories, overallTrackedMinutes]);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    window.localStorage.setItem('idletime_theme', theme);
  }, [theme]);

  useEffect(() => {
    if (!timerState.startedAt) {
      return undefined;
    }

    const intervalId = window.setInterval(() => {
      setTimerNow(Date.now());
    }, 1000);

    return () => window.clearInterval(intervalId);
  }, [timerState.startedAt]);

  useEffect(() => {
    void bootstrapSession();
  }, []);

  useEffect(() => {
    if (categories.length === 0) {
      setActivityForm((current) => ({ ...current, categoryId: '' }));
      return;
    }

    const hasSelectedCategory = categories.some((category) => String(category.id) === currentValue(activityForm.categoryId));
    if (!hasSelectedCategory) {
      setActivityForm((current) => ({ ...current, categoryId: String(categories[0].id) }));
    }
  }, [categories, activityForm.categoryId]);

  useEffect(() => {
    if (activities.length === 0) {
      setTimeEntryForm((current) => ({ ...current, activityId: '' }));
      return;
    }

    const hasSelectedActivity = activities.some((activity) => String(activity.id) === currentValue(timeEntryForm.activityId));
    if (!hasSelectedActivity) {
      setTimeEntryForm((current) => ({ ...current, activityId: String(activities[0].id) }));
    }
  }, [activities, timeEntryForm.activityId]);

  useEffect(() => {
    const token = new URLSearchParams(window.location.search).get('token');
    const onVerifyRoute = window.location.pathname === '/verify-email';

    if (!onVerifyRoute || !token || verificationHandled) {
      return;
    }

    void (async () => {
      setVerificationHandled(true);
      setErrorMessage('');
      setStatusMessage('Verifying your email...');

      try {
        const response = await apiFetch('/auth/verify-email', {
          method: 'POST',
          body: JSON.stringify({ token }),
        });
        setStatusMessage(`Email verified for ${response.user.username}. You can now log in.`);
        setAuthMode('login');
        window.history.replaceState({}, '', '/');
      } catch (error) {
        setErrorMessage(error.message);
      }
    })();
  }, [verificationHandled]);

  async function bootstrapSession() {
    setBootstrapping(true);
    setErrorMessage('');

    try {
      const response = await apiFetch('/auth/me');
      setUser(response.user);
      await loadOwnedData();
      setStatusMessage('');
    } catch {
      clearOwnedData();
      setUser(null);
    } finally {
      setBootstrapping(false);
    }
  }

  async function loadOwnedData(options = {}) {
    const requestedCategoryIds = (options.categoryIds ?? selectedRecommendationCategoryIds).map((value) => String(value));
    const shouldLoadRecommendation = options.includeRecommendation ?? hasRequestedRecommendation;

    const [categoryResponse, activityResponse, timeEntryResponse] = await Promise.all([
      apiFetch('/categories'),
      apiFetch('/activities'),
      apiFetch('/time-entries'),
    ]);

    const nextCategories = categoryResponse.categories ?? [];
    const validCategoryIdSet = new Set(nextCategories.map((category) => String(category.id)));
    const filteredCategoryIds = requestedCategoryIds.filter((value) => validCategoryIdSet.has(value));

    if (!sameIdList(filteredCategoryIds, selectedRecommendationCategoryIds)) {
      setSelectedRecommendationCategoryIds(filteredCategoryIds);
    }

    setCategories(nextCategories);
    setActivities(activityResponse.activities ?? []);
    setTimeEntries(timeEntryResponse.time_entries ?? []);

    if (!shouldLoadRecommendation) {
      setRecommendation(null);
      setRecommendationMessage('');
      return null;
    }

    const excludedActivityIds = (options.excludeActivityIds ?? skippedRecommendationActivityIds).map(String);
    const recommendationParams = new URLSearchParams();
    excludedActivityIds.forEach((activityId) => {
      recommendationParams.append('exclude_activity_id', activityId);
    });
    filteredCategoryIds.forEach((categoryId) => {
      recommendationParams.append('category_id', categoryId);
    });

    const recommendationQuery = recommendationParams.toString();
    const recommendationPath = recommendationQuery
      ? `/recommendations?${recommendationQuery}`
      : '/recommendations';
    const recommendationResponse = await apiFetch(recommendationPath);

    setRecommendation(recommendationResponse.recommendation ?? null);
    setRecommendationMessage(recommendationResponse.message ?? '');

    return recommendationResponse.recommendation ?? null;
  }

  function clearOwnedData() {
    setCategories([]);
    setActivities([]);
    setTimeEntries([]);
    setRecommendation(null);
    setRecommendationMessage('');
    setHasRequestedRecommendation(false);
    setSkippedRecommendationActivityIds([]);
  }

  async function handleRegister(event) {
    event.preventDefault();
    setErrorMessage('');
    setStatusMessage('Creating account...');

    try {
      const response = await apiFetch('/auth/register', {
        method: 'POST',
        body: JSON.stringify(registerForm),
      });

      setVerificationLink(response.verification_url ?? '');
      setRegisterForm({ username: '', email: '', password: '' });
      setAuthMode('login');
      setStatusMessage(response.message);
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function handleLogin(event) {
    event.preventDefault();
    setErrorMessage('');
    setStatusMessage('Signing in...');

    try {
      const response = await apiFetch('/auth/login', {
        method: 'POST',
        body: JSON.stringify(loginForm),
      });

      setUser(response.user);
      await loadOwnedData();
      setStatusMessage('');
      setLoginForm({ identifier: '', password: '' });
      setVerificationLink('');
      setAccountPanelOpen(false);
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function handleLogout() {
    setErrorMessage('');

    try {
      await apiFetch('/auth/logout', { method: 'POST' });
      setUser(null);
      clearOwnedData();
      setStatusMessage('Signed out.');
      setAccountPanelOpen(false);
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function handleCreateCategory(event) {
    event.preventDefault();
    setErrorMessage('');

    try {
      await apiFetch('/categories', {
        method: 'POST',
        body: JSON.stringify({
          name: categoryForm.name,
          multiplier: Number(categoryForm.multiplier),
        }),
      });
      setCategoryForm({ name: '', multiplier: '1' });
      await loadOwnedData();
      setStatusMessage('Category saved.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function handleCreateActivity(event) {
    event.preventDefault();
    setErrorMessage('');

    try {
      await apiFetch('/activities', {
        method: 'POST',
        body: JSON.stringify({
          category_id: Number(activityForm.categoryId),
          name: activityForm.name,
          multiplier: Number(activityForm.multiplier),
          minimum_minutes: Number(activityForm.minimumMinutes),
        }),
      });
      setActivityForm((current) => ({
        ...current,
        name: '',
        multiplier: '1',
        minimumMinutes: '0',
      }));
      await loadOwnedData();
      setStatusMessage('Activity saved.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function handleCreateTimeEntry(event) {
    event.preventDefault();
    setErrorMessage('');

    try {
      await apiFetch('/time-entries', {
        method: 'POST',
        body: JSON.stringify({
          activity_id: Number(timeEntryForm.activityId),
          minutes: Number(timeEntryForm.minutes),
          note: timeEntryForm.note,
        }),
      });
      setTimeEntryForm((current) => ({
        ...current,
        minutes: '30',
        note: '',
      }));
      await loadOwnedData();
      setStatusMessage('Time entry saved.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function handleRefreshData() {
    setErrorMessage('');

    try {
      await loadOwnedData();
      setStatusMessage('Data refreshed.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  function handleUseRecommendation() {
    if (!recommendation) {
      return;
    }

    setTimeEntryForm((current) => ({
      ...current,
      activityId: String(recommendation.activity_id),
    }));
    setStatusMessage(`Selected ${recommendation.activity_name} for your next time entry.`);
  }

  async function handleSkipRecommendation() {
    if (!recommendation) {
      return;
    }

    setErrorMessage('');
    const nextExcludedIds = [...skippedRecommendationActivityIds, String(recommendation.activity_id)];
    setSkippedRecommendationActivityIds(nextExcludedIds);

    try {
      await loadOwnedData({
        includeRecommendation: true,
        excludeActivityIds: nextExcludedIds,
        categoryIds: selectedRecommendationCategoryIds,
      });
      setStatusMessage('');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  function handleToggleRecommendationCategory(categoryId) {
    const id = String(categoryId);
    const descendantIDs = [];
    const stack = [id];
    const seen = new Set();

    while (stack.length > 0) {
      const currentID = String(stack.pop());
      if (seen.has(currentID)) {
        continue;
      }

      seen.add(currentID);
      descendantIDs.push(currentID);
      const children = categoriesByParent[currentID] ?? [];
      children.forEach((child) => stack.push(String(child.id)));
    }

    const nextCategoryIds = selectedRecommendationCategoryIds.includes(id)
      ? selectedRecommendationCategoryIds.filter((value) => !descendantIDs.includes(value))
      : [
        ...selectedRecommendationCategoryIds.filter((value) => !descendantIDs.includes(value)),
        id,
      ];

    setSelectedRecommendationCategoryIds(nextCategoryIds);
    setHasRequestedRecommendation(false);
    setSkippedRecommendationActivityIds([]);
    setRecommendation(null);
    setRecommendationMessage('');
    setErrorMessage('');
    setStatusMessage('');
  }

  function toggleRecommendationCategoryExpansion(categoryId) {
    setExpandedRecommendationCategories((current) => ({
      ...current,
      [categoryId]: !current[categoryId],
    }));
  }

  function expandAllRecommendationCategories() {
    setExpandedRecommendationCategories(
      Object.fromEntries(categories.map((category) => [category.id, true])),
    );
  }

  function collapseAllRecommendationCategories() {
    setExpandedRecommendationCategories(
      Object.fromEntries(categories.map((category) => [category.id, false])),
    );
  }

  function renderRecommendationCategoryTreeNode(category, depth = 0) {
    const childCategories = categoriesByParent[String(category.id)] ?? [];
    const isExpanded = expandedRecommendationCategories[category.id] ?? depth === 0;
    const isSelected = selectedRecommendationCategoryIdSet.has(String(category.id));
    const hasChildren = childCategories.length > 0;

    return (
      <div className="recommendation-tree-node" key={category.id}>
        <div
          className={isSelected ? 'recommendation-tree-item active' : 'recommendation-tree-item'}
          onClick={() => handleToggleRecommendationCategory(category.id)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault();
              handleToggleRecommendationCategory(category.id);
            }
          }}
          role="button"
          tabIndex={0}
        >
          <span
            className={hasChildren ? 'library-arrow recommendation-inline-toggle' : 'library-arrow recommendation-inline-toggle muted'}
            aria-hidden="true"
            onClick={(event) => {
              event.stopPropagation();
              if (hasChildren) {
                toggleRecommendationCategoryExpansion(category.id);
              }
            }}
          >
            {hasChildren ? (isExpanded ? '▾' : '▸') : '•'}
          </span>
          <span className="library-icon" aria-hidden="true">📁</span>
          <span className="library-label">{displayCategoryName(category.name)}</span>
        </div>

        {isExpanded && hasChildren ? (
          <div className="recommendation-tree-children" role="group">
            {childCategories.map((childCategory) => renderRecommendationCategoryTreeNode(childCategory, depth + 1))}
          </div>
        ) : null}
      </div>
    );
  }

  async function handleRecommendActivity() {
    setErrorMessage('');
    setHasRequestedRecommendation(true);
    setSkippedRecommendationActivityIds([]);

    try {
      await loadOwnedData({
        includeRecommendation: true,
        excludeActivityIds: [],
        categoryIds: selectedRecommendationCategoryIds,
      });
      setStatusMessage('');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function handleRecommendationTimer() {
    if (!recommendation) {
      return;
    }

    const isRunningCurrentRecommendation = timerState.startedAt && timerState.activityId === recommendation.activity_id;
    if (!isRunningCurrentRecommendation) {
      setTimerState({
        activityId: recommendation.activity_id,
        startedAt: Date.now(),
      });
      setTimerNow(Date.now());
      setStatusMessage('');
      return;
    }

    setErrorMessage('');

    try {
      const minutes = Math.max(1, Math.round((Date.now() - timerState.startedAt) / 60000));
      await apiFetch('/time-entries', {
        method: 'POST',
        body: JSON.stringify({
          activity_id: recommendation.activity_id,
          minutes,
          note: '',
        }),
      });
      setTimerState({ activityId: null, startedAt: null });
      setTimerNow(Date.now());
      setSkippedRecommendationActivityIds([]);
      await loadOwnedData({ includeRecommendation: true, categoryIds: selectedRecommendationCategoryIds });
      setStatusMessage(`${minutes} min saved for ${recommendation.activity_name}.`);
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  function toggleLibraryCategory(categoryId) {
    setExpandedLibraryCategories((current) => ({
      ...current,
      [categoryId]: !current[categoryId],
    }));
  }

  function expandAllLibraryCategories() {
    setExpandedLibraryCategories(
      Object.fromEntries(categories.map((category) => [category.id, true])),
    );
  }

  function collapseAllLibraryCategories() {
    setExpandedLibraryCategories(
      Object.fromEntries(categories.map((category) => [category.id, false])),
    );
  }

  function openLibrarySettings(type, item) {
    setErrorMessage('');
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

  async function handleSaveLibrarySettings() {
    if (!librarySettingsTarget) {
      return;
    }

    setErrorMessage('');

    try {
      if (librarySettingsTarget.type === 'category') {
        const parentID = librarySettingsDraft.parentId ? Number(librarySettingsDraft.parentId) : null;
        await apiFetch(`/categories/${librarySettingsTarget.item.id}`, {
          method: 'PATCH',
          body: JSON.stringify({
            name: String(librarySettingsDraft.name ?? ''),
            multiplier: Number(librarySettingsDraft.multiplier ?? 1),
            parent_id: parentID,
          }),
        });
      } else {
        await apiFetch(`/activities/${librarySettingsTarget.item.id}`, {
          method: 'PATCH',
          body: JSON.stringify({
            name: String(librarySettingsDraft.name ?? ''),
            multiplier: Number(librarySettingsDraft.multiplier ?? 1),
            minimum_minutes: Number(librarySettingsDraft.minimumMinutes ?? 0),
            tracked_minutes: Number(librarySettingsDraft.trackedMinutes ?? 0),
            category_id: Number(librarySettingsDraft.categoryId),
          }),
        });
      }

      await loadOwnedData();
      closeLibrarySettings();
      setStatusMessage('Saved changes.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function handleDeleteLibraryTarget() {
    if (!librarySettingsTarget) {
      return;
    }

    const target = librarySettingsTarget;
    closeLibrarySettings();

    if (target.type === 'category') {
      await deleteCategory(target.item);
      return;
    }

    await deleteActivity(target.item);
  }

  async function createCategoryInLibrary(parentCategory = null) {
    const targetParent = parentCategory ?? rootCategory;
    const name = window.prompt(
      targetParent
        ? `New category inside “${displayCategoryPath(targetParent, categoryById)}”`
        : 'New category name',
    );
    if (name === null) {
      return;
    }

    const trimmedName = name.trim();
    if (!trimmedName) {
      setErrorMessage('Category name is required.');
      return;
    }

    setErrorMessage('');

    try {
      const response = await apiFetch('/categories', {
        method: 'POST',
        body: JSON.stringify({
          name: trimmedName,
          multiplier: 1,
          parent_id: targetParent?.id ?? 0,
        }),
      });

      const createdCategoryID = response.category?.id;
      setExpandedLibraryCategories((current) => ({
        ...current,
        ...(targetParent ? { [targetParent.id]: true } : {}),
        ...(createdCategoryID ? { [createdCategoryID]: true } : {}),
      }));
      await loadOwnedData();
      setStatusMessage(targetParent ? 'Category added.' : 'Category added.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function createActivityInLibrary(category) {
    const name = window.prompt(`New activity inside “${displayCategoryPath(category, categoryById)}”`);
    if (name === null) {
      return;
    }

    const trimmedName = name.trim();
    if (!trimmedName) {
      setErrorMessage('Activity name is required.');
      return;
    }

    setErrorMessage('');

    try {
      await apiFetch('/activities', {
        method: 'POST',
        body: JSON.stringify({
          category_id: category.id,
          name: trimmedName,
          multiplier: 1,
          minimum_minutes: 0,
        }),
      });
      setExpandedLibraryCategories((current) => ({
        ...current,
        [category.id]: true,
      }));
      await loadOwnedData();
      setStatusMessage('Activity added.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function deleteCategory(category) {
    if (isDefaultCategory(category)) {
      setStatusMessage('The root “Library” category cannot be deleted.');
      return;
    }

    if (!window.confirm(`Delete category “${category.name}”?`)) {
      return;
    }

    setErrorMessage('');

    try {
      await apiFetch(`/categories/${category.id}`, { method: 'DELETE' });
      await loadOwnedData();
      setStatusMessage('Category deleted.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }


  async function deleteActivity(activity) {
    if (!window.confirm(`Delete activity “${activity.name}”?`)) {
      return;
    }

    setErrorMessage('');

    try {
      await apiFetch(`/activities/${activity.id}`, { method: 'DELETE' });
      await loadOwnedData();
      setStatusMessage('Activity deleted.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function editTimeEntry(entry) {
    const minutes = window.prompt('Minutes', String(entry.minutes));
    if (minutes === null) {
      return;
    }

    const note = window.prompt('Note', entry.note ?? '');
    if (note === null) {
      return;
    }

    setErrorMessage('');

    try {
      await apiFetch(`/time-entries/${entry.id}`, {
        method: 'PATCH',
        body: JSON.stringify({
          minutes: Number(minutes),
          note,
        }),
      });
      await loadOwnedData();
      setStatusMessage('Time entry updated.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function deleteTimeEntry(entry) {
    if (!window.confirm(`Delete the ${entry.minutes}-minute entry?`)) {
      return;
    }

    setErrorMessage('');

    try {
      await apiFetch(`/time-entries/${entry.id}`, { method: 'DELETE' });
      await loadOwnedData();
      setStatusMessage('Time entry deleted.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  function renderLibraryCategory(category, depth = 0) {
    const childCategories = categoriesByParent[String(category.id)] ?? [];
    const categoryActivities = activitiesByCategoryId[String(category.id)] ?? [];
    const isExpanded = expandedLibraryCategories[category.id] ?? depth === 0;

    return (
      <div className="library-node" key={category.id}>
        <div className="library-row">
          <div
            className="library-row-main"
            onClick={() => toggleLibraryCategory(category.id)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                toggleLibraryCategory(category.id);
              }
            }}
            role="button"
            tabIndex={0}
            aria-expanded={isExpanded}
          >
            <span className="library-arrow" aria-hidden="true">{isExpanded ? '▾' : '▸'}</span>
            <span className="library-icon" aria-hidden="true">{isExpanded ? '📂' : '📁'}</span>
            <span className="library-label">{displayCategoryName(category.name)}</span>
            <button
              className="library-settings-btn"
              type="button"
              aria-label={`Settings for ${displayCategoryName(category.name)}`}
              onClick={(event) => {
                event.stopPropagation();
                openLibrarySettings('category', category);
              }}
            >
              ⚙︎
            </button>
          </div>
        </div>

        {isExpanded ? (
          <div className="library-children" role="group">
            {childCategories.map((childCategory) => renderLibraryCategory(childCategory, depth + 1))}
            {categoryActivities.map((activity) => (
              <div className="library-child" key={activity.id}>
                <span className="library-arrow placeholder" aria-hidden="true">•</span>
                <span className="library-icon" aria-hidden="true">📄</span>
                <span className="library-label">{activity.name}</span>
                <button
                  className="library-settings-btn"
                  type="button"
                  aria-label={`Settings for ${activity.name}`}
                  onClick={() => openLibrarySettings('activity', activity)}
                >
                  ⚙︎
                </button>
              </div>
            ))}

            <button
              className="library-ghost-row"
              onClick={() => void createCategoryInLibrary(category)}
              type="button"
            >
              <span className="library-arrow placeholder" aria-hidden="true">+</span>
              <span className="library-icon" aria-hidden="true">📁</span>
              <span className="library-label">New Category</span>
            </button>

            <button
              className="library-ghost-row"
              onClick={() => void createActivityInLibrary(category)}
              type="button"
            >
              <span className="library-arrow placeholder" aria-hidden="true">+</span>
              <span className="library-icon" aria-hidden="true">📄</span>
              <span className="library-label">New Activity</span>
            </button>
          </div>
        ) : null}
      </div>
    );
  }

  function renderLibraryNodes(parentId = null, depth = 0) {
    const parentKey = parentId === null ? 'root' : String(parentId);
    const childCategories = categoriesByParent[parentKey] ?? [];
    return childCategories.map((category) => renderLibraryCategory(category, depth));
  }

  const isTimingRecommendation = Boolean(
    recommendation && timerState.startedAt && timerState.activityId === recommendation.activity_id,
  );

  if (bootstrapping) {
    return (
      <div className="app-shell">
        <header className="card top-bar">
          <button
            className="theme-button"
            onClick={() => setTheme((current) => (current === 'dark' ? 'light' : 'dark'))}
            type="button"
            aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
          >
            <span aria-hidden="true">{theme === 'dark' ? '☼' : '☾'}</span>
          </button>
          <h1 className="top-bar-title">idletime</h1>
          <button className="account-button" type="button" aria-label="Account">
            <span aria-hidden="true">👤</span>
          </button>
        </header>
      </div>
    );
  }

  return (
    <div className="app-shell">
      <header className="card top-bar">
        <button
          className="theme-button"
          onClick={() => setTheme((current) => (current === 'dark' ? 'light' : 'dark'))}
          type="button"
          aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        >
          <span aria-hidden="true">{theme === 'dark' ? '☼' : '☾'}</span>
        </button>
        <h1 className="top-bar-title">idletime</h1>
        <button
          className={accountPanelOpen || !user ? 'account-button active' : 'account-button'}
          onClick={() => setAccountPanelOpen((open) => !open)}
          type="button"
          aria-label="Account"
        >
          <span aria-hidden="true">👤</span>
        </button>
      </header>

      {statusMessage ? <div className="alert success">{statusMessage}</div> : null}
      {errorMessage ? <div className="alert error">{errorMessage}</div> : null}

      {!user || accountPanelOpen ? (
        <section className="card stack account-panel">
          {user ? (
            <>
              <div className="section-heading">
                <h2>Account</h2>
              </div>
              <p className="muted-text">
                <strong>{user.username}</strong> · {user.email}
              </p>
              <div className="row gap wrap-row">
                <button type="button" onClick={handleRefreshData}>
                  Refresh
                </button>
                <button type="button" onClick={handleLogout}>
                  Log out
                </button>
              </div>
            </>
          ) : (
            <>
              <div className="tabs">
                <button
                  className={authMode === 'login' ? 'tab active' : 'tab'}
                  onClick={() => setAuthMode('login')}
                  type="button"
                >
                  Log in
                </button>
                <button
                  className={authMode === 'register' ? 'tab active' : 'tab'}
                  onClick={() => setAuthMode('register')}
                  type="button"
                >
                  Register
                </button>
              </div>

              {authMode === 'login' ? (
                <form className="stack" onSubmit={handleLogin}>
                  <label>
                    Email or username
                    <input
                      value={loginForm.identifier}
                      onChange={(event) => setLoginForm((current) => ({ ...current, identifier: event.target.value }))}
                      placeholder="you@example.com"
                      required
                    />
                  </label>
                  <label>
                    Password
                    <input
                      type="password"
                      value={loginForm.password}
                      onChange={(event) => setLoginForm((current) => ({ ...current, password: event.target.value }))}
                      placeholder="••••••••"
                      required
                    />
                  </label>
                  <button className="primary-button" type="submit">Log in</button>
                </form>
              ) : (
                <form className="stack" onSubmit={handleRegister}>
                  <label>
                    Username
                    <input
                      value={registerForm.username}
                      onChange={(event) => setRegisterForm((current) => ({ ...current, username: event.target.value }))}
                      placeholder="alice"
                      required
                    />
                  </label>
                  <label>
                    Email
                    <input
                      type="email"
                      value={registerForm.email}
                      onChange={(event) => setRegisterForm((current) => ({ ...current, email: event.target.value }))}
                      placeholder="you@example.com"
                      required
                    />
                  </label>
                  <label>
                    Password
                    <input
                      type="password"
                      value={registerForm.password}
                      onChange={(event) => setRegisterForm((current) => ({ ...current, password: event.target.value }))}
                      placeholder="At least 8 characters"
                      required
                    />
                  </label>
                  <button className="primary-button" type="submit">Create account</button>
                </form>
              )}

              {verificationLink ? (
                <div className="dev-note">
                  <strong>Verification link</strong>
                  <a href={verificationLink}>{verificationLink}</a>
                </div>
              ) : null}
            </>
          )}
        </section>
      ) : null}

      {user ? (
        <>
          <nav className="card view-switcher" aria-label="Main sections">
            <button
              className={activeView === 'use' ? 'nav-tab active' : 'nav-tab'}
              onClick={() => setActiveView('use')}
              type="button"
            >
              Activity
            </button>
            <button
              className={activeView === 'progress' ? 'nav-tab active' : 'nav-tab'}
              onClick={() => setActiveView('progress')}
              type="button"
            >
              Progress
            </button>
            <button
              className={activeView === 'edit' ? 'nav-tab active' : 'nav-tab'}
              onClick={() => setActiveView('edit')}
              type="button"
            >
              Library
            </button>
          </nav>

          {activeView === 'use' ? (
            <section className="stack">
              <article className="card stack">
                <div className="section-heading">
                  <h2>Choose from categories</h2>
                  <span className="pill">
                    {selectedRecommendationCategoryIdSet.size} selected
                  </span>
                </div>

                {categories.length === 0 ? (
                  <p className="empty">Add a category first.</p>
                ) : (
                  <div className="recommendation-picker stack">
                    <button
                      className="recommendation-picker-trigger"
                      type="button"
                      onClick={() => setRecommendationCategoryPickerOpen((open) => !open)}
                      aria-expanded={recommendationCategoryPickerOpen}
                    >
                      Choose from categories
                      <span aria-hidden="true">{recommendationCategoryPickerOpen ? '▴' : '▾'}</span>
                    </button>

                    {recommendationCategoryPickerOpen ? (
                      <div className="recommendation-picker-panel stack" role="region" aria-label="Category picker">
                        <div className="row gap small-gap wrap-row">
                          <button type="button" onClick={expandAllRecommendationCategories}>Expand all</button>
                          <button type="button" onClick={collapseAllRecommendationCategories}>Collapse all</button>
                        </div>

                        <div className="recommendation-tree" role="tree" aria-label="Categories">
                          {rootCategory
                            ? renderRecommendationCategoryTreeNode(rootCategory, 0)
                            : (categoriesByParent.root ?? []).map((category) => (
                              renderRecommendationCategoryTreeNode(category, 0)
                            ))}
                        </div>
                      </div>
                    ) : null}
                  </div>
                )}

                <div className="row gap wrap-row center-row">
                  <button className="primary-button" type="button" onClick={handleRecommendActivity}>
                    Recommend activity
                  </button>
                </div>
              </article>

              {hasRequestedRecommendation ? (
                <article className="card stack recommendation-card centered-card">
                  <h2 className="center-text">{recommendation?.activity_name ?? 'No activity yet'}</h2>

                  {recommendation ? (
                    <div className="row gap wrap-row center-row">
                      <button className="primary-button" type="button" onClick={handleRecommendationTimer}>
                        {isTimingRecommendation
                          ? `Stop timer (${formatTimerDuration(timerState.startedAt, timerNow)})`
                          : 'Start timer'}
                      </button>
                      <button type="button" onClick={handleSkipRecommendation}>
                        Skip
                      </button>
                    </div>
                  ) : (
                    <p className="empty center-text">
                      {recommendationMessage || 'No activity yet.'}
                    </p>
                  )}
                </article>
              ) : null}
            </section>
          ) : null}

          {activeView === 'progress' ? (
            <section className="stack">
              <article className="card stack">
                <div className="section-heading">
                  <div>
                    <p className="eyebrow">Progress</p>
                    <h2>Where your time is going</h2>
                  </div>
                  <span className="pill">{formatMinutes(overallTrackedMinutes)} total</span>
                </div>

                <div className="stat-grid">
                  <div className="stat-tile">
                    <strong>{categoryProgress.length}</strong>
                    <span>Categories in play</span>
                  </div>
                  <div className="stat-tile">
                    <strong>{activities.length}</strong>
                    <span>Tracked activities</span>
                  </div>
                  <div className="stat-tile">
                    <strong>{timeEntries.length}</strong>
                    <span>Saved sessions</span>
                  </div>
                  <div className="stat-tile">
                    <strong>{overallTrackedMinutes}</strong>
                    <span>Total minutes</span>
                  </div>
                </div>
              </article>

              <div className="progress-grid">
                {categoryProgress.length === 0 ? (
                  <article className="card">
                    <p className="empty">Add a category and an activity to start seeing progress here.</p>
                  </article>
                ) : categoryProgress.map((group) => (
                  <article className="card stack" key={group.id}>
                    <div className="section-heading">
                      <div>
                        <h2>{displayCategoryPath(group, categoryById)}</h2>
                        <p className="muted-text">
                          {group.activities.length} activities · multiplier {group.multiplier}
                        </p>
                      </div>
                      {isDefaultCategory(group) ? <span className="pill">Default</span> : null}
                    </div>

                    <div className="row gap wrap-row">
                      <span className="pill success">{formatMinutes(group.totalTrackedMinutes)} tracked</span>
                      <span className="pill">{group.shareOfAll}% of all tracked time</span>
                      <span className="pill">Balance score {group.normalizedProgress.toFixed(1)}</span>
                    </div>

                    <div className="mini-chart" aria-hidden="true">
                      <span style={{ width: `${group.totalPercent}%` }} />
                    </div>

                    <div className="item-list">
                      {group.activities.length === 0 ? (
                        <p className="empty">No activities in this category yet.</p>
                      ) : group.activities.map((activity) => (
                        <div className="progress-item" key={activity.id}>
                          <div className="progress-item-header">
                            <strong>{activity.name}</strong>
                            <span>{formatMinutes(activity.trackedDisplayMinutes)}</span>
                          </div>
                          <div className="mini-chart small" aria-hidden="true">
                            <span style={{ width: `${activity.percent}%` }} />
                          </div>
                          <p className="muted-text">
                            Minimum {formatMinutes(activity.minimum_minutes)} · multiplier {activity.multiplier}
                          </p>
                        </div>
                      ))}
                    </div>
                  </article>
                ))}
              </div>
            </section>
          ) : null}

          {activeView === 'edit' ? (
            <section className="stack">
              <article className="card stack">
                <div className="section-heading">
                  <h2>Library</h2>
                  <div className="row gap small-gap wrap-row">
                    <button type="button" onClick={expandAllLibraryCategories}>Expand all</button>
                    <button type="button" onClick={collapseAllLibraryCategories}>Collapse all</button>
                  </div>
                </div>

                <div className="library-tree" role="tree" aria-label="Category library">
                  {rootCategory ? renderLibraryCategory(rootCategory, 0) : (
                    <button className="library-ghost-row" onClick={() => void createCategoryInLibrary()} type="button">
                      <span className="library-arrow placeholder" aria-hidden="true">+</span>
                      <span className="library-icon" aria-hidden="true">📁</span>
                      <span className="library-label">New Category</span>
                    </button>
                  )}
                </div>
              </article>
            </section>
          ) : null}
        </>
      ) : null}

      {librarySettingsTarget ? (
        <div
          className="library-settings-backdrop"
          role="dialog"
          aria-modal="true"
          onClick={closeLibrarySettings}
        >
          <div className="library-settings-modal card stack" onClick={(event) => event.stopPropagation()}>
            <div className="section-heading">
              <h2>
                {librarySettingsTarget.type === 'category' ? 'Category settings' : 'Activity settings'}
              </h2>
              <button type="button" onClick={closeLibrarySettings} aria-label="Close settings">
                ✕
              </button>
            </div>

            <label>
              Name
              <input
                value={String(librarySettingsDraft.name ?? '')}
                onChange={(event) => setLibrarySettingsDraft((current) => ({
                  ...current,
                  name: event.target.value,
                }))}
              />
            </label>

            <label>
              Multiplier
              <input
                type="number"
                min="0"
                step="0.1"
                value={String(librarySettingsDraft.multiplier ?? '1')}
                onChange={(event) => setLibrarySettingsDraft((current) => ({
                  ...current,
                  multiplier: event.target.value,
                }))}
              />
            </label>

            {librarySettingsTarget.type === 'category' ? (
              <label>
                Parent category
                <select
                  value={String(librarySettingsDraft.parentId ?? '')}
                  onChange={(event) => setLibrarySettingsDraft((current) => ({
                    ...current,
                    parentId: event.target.value,
                  }))}
                >
                  <option value="">Library root</option>
                  {categories
                    .filter((category) => category.id !== librarySettingsTarget.item.id)
                    .map((category) => (
                      <option key={category.id} value={String(category.id)}>
                        {displayCategoryPath(category, categoryById)}
                      </option>
                    ))}
                </select>
              </label>
            ) : (
              <>
                <label>
                  Minimum minutes
                  <input
                    type="number"
                    min="0"
                    value={String(librarySettingsDraft.minimumMinutes ?? '0')}
                    onChange={(event) => setLibrarySettingsDraft((current) => ({
                      ...current,
                      minimumMinutes: event.target.value,
                    }))}
                  />
                </label>

                <label>
                  Tracked minutes
                  <input
                    type="number"
                    min="0"
                    value={String(librarySettingsDraft.trackedMinutes ?? '0')}
                    onChange={(event) => setLibrarySettingsDraft((current) => ({
                      ...current,
                      trackedMinutes: event.target.value,
                    }))}
                  />
                </label>

                <label>
                  Category
                  <select
                    value={String(librarySettingsDraft.categoryId ?? '')}
                    onChange={(event) => setLibrarySettingsDraft((current) => ({
                      ...current,
                      categoryId: event.target.value,
                    }))}
                  >
                    {categories.map((category) => (
                      <option key={category.id} value={String(category.id)}>
                        {displayCategoryPath(category, categoryById)}
                      </option>
                    ))}
                  </select>
                </label>
              </>
            )}

            <div className="row gap wrap-row">
              <button className="primary-button" type="button" onClick={() => void handleSaveLibrarySettings()}>
                Save
              </button>
              {!(librarySettingsTarget.type === 'category' && isDefaultCategory(librarySettingsTarget.item)) ? (
                <button className="destructive-button" type="button" onClick={() => void handleDeleteLibraryTarget()}>
                  Delete
                </button>
              ) : null}
              <button type="button" onClick={closeLibrarySettings}>Cancel</button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
