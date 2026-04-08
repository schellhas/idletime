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
  const total = Math.round(Number(value ?? 0));
  const days = Math.floor(total / (60 * 24));
  const hours = Math.floor((total % (60 * 24)) / 60);
  const mins = total % 60;
  const parts = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  parts.push(`${mins}m`);
  return parts.join(' ');
}

function formatHoursFromMinutes(value) {
  const roundedHours = Math.round((Number(value ?? 0) / 60) * 10) / 10;
  return Number.isInteger(roundedHours) ? `${roundedHours.toFixed(0)}h` : `${roundedHours.toFixed(1)}h`;
}

function currentValue(value) {
  return value ?? '';
}

function displayCategoryName(name) {
  const normalized = String(name ?? '').trim().toLowerCase();
  if (normalized === 'root' || normalized === 'none') {
    return 'Activities';
  }
  return String(name ?? 'Activities');
}

function displayCategoryPath(category, categoryById) {
  if (!category) {
    return 'Activities';
  }

  const parts = [];
  const seen = new Set();
  let current = category;

  while (current && !seen.has(current.id)) {
    const label = displayCategoryName(current.name);
    if (label !== 'Activities') {
      parts.unshift(label);
    }
    seen.add(current.id);
    current = current.parent_id ? categoryById[current.parent_id] : null;
  }

  return parts.length > 0 ? parts.join(' / ') : 'Activities';
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

const PIE_COLORS = [
  '#2563eb',
  '#0ea5e9',
  '#14b8a6',
  '#22c55e',
  '#84cc16',
  '#eab308',
  '#f97316',
  '#ef4444',
  '#ec4899',
  '#8b5cf6',
];

function buildActivityPieData(activityList) {
  const relevantActivities = activityList
    .map((activity) => ({
      id: activity.id,
      name: activity.name,
      minutes: Math.max(0, Number(activity.tracked_minutes ?? activity.trackedDisplayMinutes ?? 0)),
    }))
    .filter((activity) => activity.minutes > 0)
    .sort((left, right) => right.minutes - left.minutes || left.name.localeCompare(right.name));

  const totalMinutes = relevantActivities.reduce((sum, activity) => sum + activity.minutes, 0);
  if (totalMinutes <= 0) {
    return {
      totalMinutes: 0,
      gradient: 'conic-gradient(#dbe6f3 0deg 360deg)',
      slices: [],
    };
  }

  let currentAngle = 0;
  const slices = relevantActivities.map((activity, index) => {
    const angle = (activity.minutes / totalMinutes) * 360;
    const start = currentAngle;
    const end = currentAngle + angle;
    currentAngle = end;
    const color = PIE_COLORS[index % PIE_COLORS.length];
    const percent = Math.round((activity.minutes / totalMinutes) * 100);

    return {
      ...activity,
      color,
      percent,
      gradientStop: `${color} ${start}deg ${end}deg`,
    };
  });

  return {
    totalMinutes,
    gradient: `conic-gradient(${slices.map((slice) => slice.gradientStop).join(', ')})`,
    slices,
  };
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
  const [libraryCreateTarget, setLibraryCreateTarget] = useState(null);
  const [libraryCreateDraft, setLibraryCreateDraft] = useState({ name: '' });
  const [librarySettingsTarget, setLibrarySettingsTarget] = useState(null);
  const [librarySettingsDraft, setLibrarySettingsDraft] = useState({});
  const [dragMultipliers, setDragMultipliers] = useState({});

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
    // sort each group alphabetically
    for (const key of Object.keys(grouped)) {
      grouped[key].sort((a, b) => String(a.name ?? '').localeCompare(String(b.name ?? '')));
    }
    return grouped;
  }, [categories, categoryById, rootCategory]);
  // flat list of categories in DFS tree order (matches the category tree)
  const categoriesInTreeOrder = useMemo(() => {
    const result = [];
    function walk(parentId) {
      const children = categoriesByParent[String(parentId)] ?? [];
      for (const child of children) {
        result.push(child);
        walk(child.id);
      }
    }
    if (rootCategory) walk(rootCategory.id);
    return result;
  }, [categoriesByParent, rootCategory]);
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
  const activityPieChart = useMemo(() => {
    return buildActivityPieData(activities);
  }, [activities]);

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

  async function saveMultiplier(activityId, value) {
    try {
      await apiFetch(`/activities/${activityId}`, {
        method: 'PATCH',
        body: JSON.stringify({ multiplier: value }),
      });
      await loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
    } finally {
      setDragMultipliers((m) => {
        const next = { ...m };
        delete next[activityId];
        return next;
      });
    }
  }

  function openLibraryCreate(target) {
    setErrorMessage('');
    setLibraryCreateDraft({ name: '' });
    setLibraryCreateTarget(target);
  }

  function closeLibraryCreate() {
    setLibraryCreateTarget(null);
    setLibraryCreateDraft({ name: '' });
  }

  async function handleSubmitLibraryCreate(event) {
    event.preventDefault();
    if (!libraryCreateTarget) {
      return;
    }

    const trimmedName = String(libraryCreateDraft.name ?? '').trim();
    if (!trimmedName) {
      setErrorMessage(`${libraryCreateTarget.type === 'category' ? 'Category' : 'Activity'} name is required.`);
      return;
    }

    setErrorMessage('');

    try {
      if (libraryCreateTarget.type === 'category') {
        const response = await apiFetch('/categories', {
          method: 'POST',
          body: JSON.stringify({
            name: trimmedName,
            multiplier: 1,
            parent_id: libraryCreateTarget.parentCategoryID ?? 0,
          }),
        });

        const createdCategoryID = response.category?.id;
        setExpandedLibraryCategories((current) => ({
          ...current,
          ...(libraryCreateTarget.parentCategoryID ? { [libraryCreateTarget.parentCategoryID]: true } : {}),
          ...(createdCategoryID ? { [createdCategoryID]: true } : {}),
        }));
        await loadOwnedData();
        closeLibraryCreate();
        setStatusMessage('Category added.');
        return;
      }

      await apiFetch('/activities', {
        method: 'POST',
        body: JSON.stringify({
          category_id: libraryCreateTarget.categoryID,
          name: trimmedName,
          multiplier: 1,
          minimum_minutes: 0,
        }),
      });
      setExpandedLibraryCategories((current) => ({
        ...current,
        [libraryCreateTarget.categoryID]: true,
      }));
      await loadOwnedData();
      closeLibraryCreate();
      setStatusMessage('Activity added.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function createCategoryInLibrary(parentCategory = null) {
    const targetParent = parentCategory ?? rootCategory;
    openLibraryCreate({
      type: 'category',
      parentCategoryID: targetParent?.id ?? 0,
      parentLabel: targetParent ? displayCategoryPath(targetParent, categoryById) : 'Library',
    });
  }

  async function createActivityInLibrary(category) {
    openLibraryCreate({
      type: 'activity',
      categoryID: category.id,
      parentLabel: displayCategoryPath(category, categoryById),
    });
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
              Stats
            </button>
            <button
              className={activeView === 'edit' ? 'nav-tab active' : 'nav-tab'}
              onClick={() => setActiveView('edit')}
              type="button"
            >
              Edit
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
                  <h2>All activities</h2>
                  <span className="pill">{formatMinutes(activityPieChart.totalMinutes)} tracked</span>
                </div>

                {activityPieChart.totalMinutes > 0 ? (
                  <div className="activity-pie-layout">
                    <div
                      className="activity-pie"
                      aria-label="Activity time distribution pie chart"
                      role="img"
                      style={{ background: activityPieChart.gradient }}
                    />

                    <div className="activity-pie-legend">
                      {activityPieChart.slices.map((slice) => (
                        <div className="activity-pie-legend-item" key={slice.id}>
                          <span
                            className="activity-pie-color"
                            aria-hidden="true"
                            style={{ background: slice.color }}
                          />
                          <span className="activity-pie-label">{slice.name}</span>
                          <span className="activity-pie-value">{formatHoursFromMinutes(slice.minutes)}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : (
                  <p className="empty">Track some activity time to populate the pie chart.</p>
                )}
              </article>

              {(() => {
                const progressById = Object.fromEntries(categoryProgress.map((g) => [g.id, g]));
                function renderStatsGroup(group) {
                  const childCategories = categoriesByParent[String(group.id)] ?? [];
                  const childGroups = childCategories.map((c) => progressById[c.id]).filter(Boolean);
                  const groupPieChart = buildActivityPieData(group.activities);
                  return (
                    <article className="card stack" key={group.id}>
                      <div className="section-heading">
                        <h2>{displayCategoryName(group.name)}</h2>
                        <div className="row gap small-gap wrap-row">
                          <span className="pill success">{formatMinutes(group.totalTrackedMinutes)} tracked</span>
                          {isDefaultCategory(group) ? <span className="pill">Default</span> : null}
                        </div>
                      </div>

                      {groupPieChart.totalMinutes > 0 ? (
                        <div className="category-pie-layout">
                          <div
                            className="category-pie"
                            aria-label={`Activity distribution for ${displayCategoryName(group.name)}`}
                            role="img"
                            style={{ background: groupPieChart.gradient }}
                          />

                          <div className="category-pie-legend">
                            {groupPieChart.slices.map((slice) => (
                              <div className="category-pie-legend-item" key={slice.id}>
                                <span
                                  className="category-pie-color"
                                  aria-hidden="true"
                                  style={{ background: slice.color }}
                                />
                                <span className="category-pie-label">{slice.name}</span>
                                <span className="category-pie-value">{formatHoursFromMinutes(slice.minutes)}</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      ) : (
                        <p className="empty">Track time in this category to populate its pie chart.</p>
                      )}

                      {childGroups.length > 0 && (
                        <div className="nested-category-cards">
                          {childGroups.map(renderStatsGroup)}
                        </div>
                      )}
                    </article>
                  );
                }
                const topLevel = (categoriesByParent[rootCategory ? String(rootCategory.id) : 'root'] ?? [])
                  .map((c) => progressById[c.id]).filter(Boolean);
                if (topLevel.length === 0) {
                  return (
                    <article className="card">
                      <p className="empty">Add a category and an activity to start seeing progress here.</p>
                    </article>
                  );
                }
                return <div className="progress-grid">{topLevel.map(renderStatsGroup)}</div>;
              })()}
            </section>
          ) : null}

          {activeView === 'edit' ? (
            <section className="stack">
              <article className="card stack">
                <div className="section-heading">
                  <h2>Category tree</h2>
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

              {(() => {
                const progressById = Object.fromEntries(categoryProgress.map((g) => [g.id, g]));
                function renderEditGroup(group) {
                  const childCategories = categoriesByParent[String(group.id)] ?? [];
                  const childGroups = childCategories.map((c) => progressById[c.id]).filter(Boolean);
                  const rawMax = Math.max(1, ...group.activities.map((a) => Number(a.multiplier ?? 1)));
                  const maxScale = Math.max(rawMax, 5);
                  const sorted = [...group.activities].sort(
                    (a, b) => Number(b.multiplier ?? 1) - Number(a.multiplier ?? 1),
                  );
                  return (
                    <article className="card stack" key={group.id}>
                      <div className="section-heading">
                        <h2>{displayCategoryName(group.name)}</h2>
                        {isDefaultCategory(group) ? <span className="pill">Default</span> : null}
                      </div>
                      {sorted.length === 0 ? (
                        <p className="empty">No activities in this category yet.</p>
                      ) : (
                        <div className="item-list">
                          {sorted.map((activity) => {
                            const activeMultiplier = dragMultipliers[activity.id] ?? Number(activity.multiplier ?? 1);
                            const pct = Math.min(100, Math.round((activeMultiplier / maxScale) * 100));
                            return (
                              <div className="edit-multiplier-row" key={activity.id}>
                                <span className="edit-multiplier-name">{activity.name}</span>
                                <div
                                  className="edit-multiplier-track"
                                  onPointerDown={(e) => {
                                    e.currentTarget.setPointerCapture(e.pointerId);
                                    const rect = e.currentTarget.getBoundingClientRect();
                                    const val = Math.max(0, Math.round(((e.clientX - rect.left) / rect.width) * maxScale * 10) / 10);
                                    setDragMultipliers((m) => ({ ...m, [activity.id]: val }));
                                  }}
                                  onPointerMove={(e) => {
                                    if (!e.currentTarget.hasPointerCapture(e.pointerId)) return;
                                    const rect = e.currentTarget.getBoundingClientRect();
                                    const val = Math.max(0, Math.round(((e.clientX - rect.left) / rect.width) * maxScale * 10) / 10);
                                    setDragMultipliers((m) => ({ ...m, [activity.id]: val }));
                                  }}
                                  onPointerUp={(e) => {
                                    if (!e.currentTarget.hasPointerCapture(e.pointerId)) return;
                                    e.currentTarget.releasePointerCapture(e.pointerId);
                                    const rect = e.currentTarget.getBoundingClientRect();
                                    const val = Math.max(0, Math.round(((e.clientX - rect.left) / rect.width) * maxScale * 10) / 10);
                                    void saveMultiplier(activity.id, val);
                                  }}
                                  onPointerCancel={(e) => {
                                    setDragMultipliers((m) => { const next = { ...m }; delete next[activity.id]; return next; });
                                  }}
                                >
                                  <div className="edit-multiplier-fill" style={{ width: `${pct}%` }} />
                                </div>
                                <input
                                  className="edit-multiplier-value no-spin"
                                  type="number"
                                  min="0"
                                  step="0.1"
                                  value={dragMultipliers[activity.id] !== undefined ? activeMultiplier.toFixed(1) : activeMultiplier}
                                  onChange={(e) => {
                                    setDragMultipliers((m) => ({ ...m, [activity.id]: Number(e.target.value) }));
                                  }}
                                  onBlur={(e) => {
                                    const val = Math.max(0, Math.round(Number(e.target.value) * 10) / 10);
                                    void saveMultiplier(activity.id, val);
                                  }}
                                  onKeyDown={(e) => {
                                    if (e.key === 'Enter') e.currentTarget.blur();
                                    if (e.key === 'Escape') {
                                      setDragMultipliers((m) => { const next = { ...m }; delete next[activity.id]; return next; });
                                      e.currentTarget.blur();
                                    }
                                  }}
                                />
                              </div>
                            );
                          })}
                        </div>
                      )}

                      {childGroups.length > 0 && (
                        <div className="nested-category-cards">
                          {childGroups.map(renderEditGroup)}
                        </div>
                      )}
                    </article>
                  );
                }
                const topLevel = (categoriesByParent[rootCategory ? String(rootCategory.id) : 'root'] ?? [])
                  .map((c) => progressById[c.id]).filter(Boolean);
                if (topLevel.length === 0) {
                  return (
                    <article className="card">
                      <p className="empty">Add a category and activities to see multiplier charts here.</p>
                    </article>
                  );
                }
                return <div className="progress-grid">{topLevel.map(renderEditGroup)}</div>;
              })()}
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
                className="no-spin"
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
                    className="no-spin"
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
                    className="no-spin"
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

      {libraryCreateTarget ? (
        <div
          className="library-settings-backdrop"
          role="dialog"
          aria-modal="true"
          onClick={closeLibraryCreate}
        >
          <form className="library-settings-modal card stack" onClick={(event) => event.stopPropagation()} onSubmit={handleSubmitLibraryCreate}>
            <div className="section-heading">
              <h2>{libraryCreateTarget.type === 'category' ? 'New category' : 'New activity'}</h2>
              <button type="button" onClick={closeLibraryCreate} aria-label="Close create dialog">
                ✕
              </button>
            </div>

            <p className="muted-text">
              Inside {libraryCreateTarget.parentLabel}
            </p>

            <label>
              Name
              <input
                autoFocus
                value={String(libraryCreateDraft.name ?? '')}
                onChange={(event) => setLibraryCreateDraft((current) => ({
                  ...current,
                  name: event.target.value,
                }))}
              />
            </label>

            <div className="row gap wrap-row">
              <button className="primary-button" type="submit">Create</button>
              <button type="button" onClick={closeLibraryCreate}>Cancel</button>
            </div>
          </form>
        </div>
      ) : null}
    </div>
  );
}
