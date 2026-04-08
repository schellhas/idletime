import { useEffect, useState, useCallback } from 'react';
import CreateModal from './components/modals/CreateModal';
import DeleteConfirmModal from './components/modals/DeleteConfirmModal';
import SettingsModal from './components/modals/SettingsModal';
import { UseView } from './components/views/UseView';
import { StatsView } from './components/views/StatsView';
import { EditView } from './components/views/EditView';
import { useAuthState } from './hooks/useAuthState';
import { useOwnedData } from './hooks/useOwnedData';
import { useLibrarySettings } from './hooks/useLibrarySettings';
import { useDerived } from './hooks/useDerived';
import { useTimerState } from './hooks/useTimerState';
import { useSessionAPI, apiFetch } from './hooks/useSessionAPI';
import {
  buildActivityPieData,
  displayCategoryPath,
  isDefaultCategory,
} from './utils/appUtils';
export default function App() {
  const [theme, setTheme] = useState(() => {
    const saved = window.localStorage.getItem('idletime_theme');
    if (saved === 'dark' || saved === 'light') return saved;
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  });
  const [activeView, setActiveView] = useState('use');
  const [dragMultipliers, setDragMultipliers] = useState({});
  const [hasRequestedRecommendation, setHasRequestedRecommendation] = useState(false);
  const [skippedActivityIds, setSkippedActivityIds] = useState([]);
  const [recommendation, setRecommendation] = useState(null);
  const [recommendationMessage, setRecommendationMessage] = useState('');
  const [statusMessage, setStatusMessage] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const auth = useAuthState();
  const data = useOwnedData();
  const lib = useLibrarySettings();
  const derived = useDerived(data.categories, data.activities, dragMultipliers, lib.selectedRecommendationCategoryIds);
  const timer = useTimerState();
  const recState = { hasRequested: hasRequestedRecommendation, skipped: skippedActivityIds, setRecommendation, setMessage: setRecommendationMessage };
  const sessionAPI = useSessionAPI(data, lib, recState, setStatusMessage, setErrorMessage);
  const activityPieChart = buildActivityPieData(data.activities);
  const rootCategory = data.categories.find((cat) => String(cat.name ?? '').trim().toLowerCase() === 'root')
    ?? data.categories.find((cat) => !cat.parent_id) ?? null;
  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    window.localStorage.setItem('idletime_theme', theme);
  }, [theme]);
  useEffect(() => {
    void sessionAPI.bootstrapSession();
  }, []);
  useEffect(() => {
    const token = new URLSearchParams(window.location.search).get('token');
    if (window.location.pathname !== '/verify-email' || !token || sessionAPI.verificationHandled) return;
    void (async () => {
      sessionAPI.setVerificationHandled(true);
      setErrorMessage('');
      setStatusMessage('Verifying email...');
      try {
        await apiFetch('/auth/verify-email', {
          method: 'POST',
          body: JSON.stringify({ token }),
        });
        setStatusMessage('Email verified. Log in now.');
        auth.setAuthMode('login');
        window.history.replaceState({}, '', '/');
      } catch (error) {
        setErrorMessage(error.message);
      }
    })();
  }, [sessionAPI.verificationHandled]);
  const handleRecommendActivity = useCallback(async () => {
    setErrorMessage('');
    setHasRequestedRecommendation(true);
    setSkippedActivityIds([]);
    try {
      await sessionAPI.loadOwnedData({
        includeRecommendation: true,
        excludeActivityIds: [],
        categoryIds: lib.selectedRecommendationCategoryIds,
      });
      setStatusMessage('');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }, [lib.selectedRecommendationCategoryIds, sessionAPI]);
  const handleRecommendationTimer = useCallback(async () => {
    if (!recommendation) return;
    const isRunning = timer.timerState.startedAt && timer.timerState.activityId === recommendation.activity_id;
    if (!isRunning) {
      timer.setTimerState({ activityId: recommendation.activity_id, startedAt: Date.now() });
      timer.setTimerNow(Date.now());
      setStatusMessage('');
      return;
    }
    setErrorMessage('');
    try {
      const mins = Math.max(1, Math.round((Date.now() - timer.timerState.startedAt) / 60000));
      await apiFetch('/time-entries', {
        method: 'POST',
        body: JSON.stringify({ activity_id: recommendation.activity_id, minutes: mins, note: '' }),
      });
      timer.setTimerState({ activityId: null, startedAt: null });
      timer.setTimerNow(Date.now());
      setSkippedActivityIds([]);
      await sessionAPI.loadOwnedData({ includeRecommendation: true, categoryIds: lib.selectedRecommendationCategoryIds });
      setStatusMessage(`${mins} min saved.`);
    } catch (error) {
      setErrorMessage(error.message);
    }
  }, [recommendation, timer, sessionAPI, lib.selectedRecommendationCategoryIds]);
  const handleSkipRecommendation = useCallback(async () => {
    if (!recommendation) return;
    setErrorMessage('');
    const nextIds = [...skippedActivityIds, String(recommendation.activity_id)];
    setSkippedActivityIds(nextIds);
    try {
      await sessionAPI.loadOwnedData({
        includeRecommendation: true,
        excludeActivityIds: nextIds,
        categoryIds: lib.selectedRecommendationCategoryIds,
      });
      setStatusMessage('');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }, [recommendation, skippedActivityIds, sessionAPI, lib.selectedRecommendationCategoryIds]);
  const handleToggleRecommendationCategory = useCallback((categoryId) => {
    const id = String(categoryId);
    const descendantIds = [];
    const stack = [id];
    const seen = new Set();
    while (stack.length > 0) {
      const current = String(stack.pop());
      if (seen.has(current)) continue;
      seen.add(current);
      descendantIds.push(current);
      (derived.categoriesByParent[current] ?? []).forEach((child) => stack.push(String(child.id)));
    }
    const next = lib.selectedRecommendationCategoryIds.includes(id)
      ? lib.selectedRecommendationCategoryIds.filter((v) => !descendantIds.includes(v))
      : [...lib.selectedRecommendationCategoryIds.filter((v) => !descendantIds.includes(v)), id];
    lib.setSelectedRecommendationCategoryIds(next);
    setHasRequestedRecommendation(false);
    setSkippedActivityIds([]);
    setRecommendation(null);
    setRecommendationMessage('');
    setErrorMessage('');
    setStatusMessage('');
  }, [lib, derived.categoriesByParent]);
  const handleSaveLibrarySettings = useCallback(async () => {
    if (!lib.librarySettingsTarget) return;
    setErrorMessage('');
    try {
      const method = lib.librarySettingsTarget.type === 'category' ? '/categories' : '/activities';
      const body = lib.librarySettingsTarget.type === 'category'
        ? {
          name: String(lib.librarySettingsDraft.name ?? ''),
          multiplier: Number(lib.librarySettingsDraft.multiplier ?? 1),
          parent_id: lib.librarySettingsDraft.parentId ? Number(lib.librarySettingsDraft.parentId) : null,
        }
        : {
          name: String(lib.librarySettingsDraft.name ?? ''),
          multiplier: Number(lib.librarySettingsDraft.multiplier ?? 1),
          minimum_minutes: Number(lib.librarySettingsDraft.minimumMinutes ?? 0),
          tracked_minutes: Number(lib.librarySettingsDraft.trackedMinutes ?? 0),
          category_id: Number(lib.librarySettingsDraft.categoryId),
        };
      await apiFetch(`${method}/${lib.librarySettingsTarget.item.id}`, { method: 'PATCH', body: JSON.stringify(body) });
      await sessionAPI.loadOwnedData();
      lib.closeLibrarySettings();
      setStatusMessage('Saved.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }, [lib, sessionAPI]);
  const handleDeleteLibraryTarget = useCallback(async () => {
    if (!lib.librarySettingsTarget) return;
    if (lib.librarySettingsTarget.type === 'category' && isDefaultCategory(lib.librarySettingsTarget.item)) {
      setStatusMessage('Root category cannot be deleted.');
      return;
    }
    lib.closeLibrarySettings();
    lib.setDeleteConfirmTarget(lib.librarySettingsTarget);
  }, [lib]);
  const confirmDeleteLibraryTarget = useCallback(async () => {
    if (!lib.deleteConfirmTarget) return;
    const target = lib.deleteConfirmTarget;
    lib.setDeleteConfirmTarget(null);
    setErrorMessage('');
    try {
      const type = target.type === 'category' ? 'categories' : 'activities';
      await apiFetch(`/${type}/${target.item.id}`, { method: 'DELETE' });
      await sessionAPI.loadOwnedData();
      setStatusMessage(`${target.type === 'category' ? 'Category' : 'Activity'} deleted.`);
    } catch (error) {
      setErrorMessage(error.message);
    }
  }, [lib, sessionAPI]);
  const saveMultiplier = useCallback(async (activityId, value) => {
    try {
      await apiFetch(`/activities/${activityId}`, {
        method: 'PATCH',
        body: JSON.stringify({ multiplier: value }),
      });
      await sessionAPI.loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
    } finally {
      setDragMultipliers((m) => {
        const next = { ...m };
        delete next[`activity-${activityId}`];
        return next;
      });
    }
  }, [sessionAPI]);
  const saveCategoryMultiplier = useCallback(async (categoryId, value) => {
    try {
      await apiFetch(`/categories/${categoryId}`, {
        method: 'PATCH',
        body: JSON.stringify({ multiplier: value }),
      });
      await sessionAPI.loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
    } finally {
      setDragMultipliers((m) => {
        const next = { ...m };
        delete next[`category-${categoryId}`];
        return next;
      });
    }
  }, [sessionAPI]);
  const handleSubmitLibraryCreate = useCallback(async (event) => {
    event.preventDefault();
    if (!lib.libraryCreateTarget) return;
    const name = String(lib.libraryCreateDraft.name ?? '').trim();
    if (!name) {
      setErrorMessage('Name required.');
      return;
    }
    setErrorMessage('');
    try {
      if (lib.libraryCreateTarget.type === 'category') {
        const res = await apiFetch('/categories', {
          method: 'POST',
          body: JSON.stringify({
            name,
            multiplier: 1,
            parent_id: lib.libraryCreateTarget.parentCategoryID ?? 0,
          }),
        });
        const created = res.category?.id;
        lib.setExpandedLibraryCategories((c) => ({
          ...c,
          ...(lib.libraryCreateTarget.parentCategoryID ? { [lib.libraryCreateTarget.parentCategoryID]: true } : {}),
          ...(created ? { [created]: true } : {}),
        }));
      } else {
        await apiFetch('/activities', {
          method: 'POST',
          body: JSON.stringify({
            category_id: lib.libraryCreateTarget.categoryID,
            name,
            multiplier: 1,
            minimum_minutes: 0,
          }),
        });
        lib.setExpandedLibraryCategories((c) => ({
          ...c,
          [lib.libraryCreateTarget.categoryID]: true,
        }));
      }
      await sessionAPI.loadOwnedData();
      lib.closeLibraryCreate();
      setStatusMessage('Added.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }, [lib, sessionAPI]);
  const createCategoryInLibrary = useCallback((parent = null) => {
    const target = parent ?? rootCategory;
    lib.openLibraryCreate({
      type: 'category',
      parentCategoryID: target?.id ?? 0,
      parentLabel: target ? displayCategoryPath(target, derived.categoryById) : 'Library',
    });
  }, [lib, rootCategory, derived.categoryById]);
  const createActivityInLibrary = useCallback((cat) => {
    lib.openLibraryCreate({
      type: 'activity',
      categoryID: cat.id,
      parentLabel: displayCategoryPath(cat, derived.categoryById),
    });
  }, [lib, derived.categoryById]);
  const isTimingRecommendation = Boolean(
    recommendation && timer.timerState.startedAt && timer.timerState.activityId === recommendation.activity_id,
  );
  if (auth.bootstrapping) {
    return (
      <div className="app-shell">
        <header className="card top-bar">
          <button className="theme-button" onClick={() => setTheme((current) => (current === 'dark' ? 'light' : 'dark'))} type="button" aria-label="Toggle theme">
            <span aria-hidden="true">{theme === 'dark' ? '☼' : '☾'}</span>
          </button>
          <h1 className="top-bar-title">idletime</h1>
          <button className="account-button" type="button" aria-label="Account"><span aria-hidden="true">👤</span></button>
        </header>
      </div>
    );
  }
  return (
    <div className="app-shell">
      <header className="card top-bar">
        <button className="theme-button" onClick={() => setTheme((current) => (current === 'dark' ? 'light' : 'dark'))} type="button" aria-label="Toggle theme">
          <span aria-hidden="true">{theme === 'dark' ? '☼' : '☾'}</span>
        </button>
        <h1 className="top-bar-title">idletime</h1>
        <button className={auth.accountPanelOpen || !auth.user ? 'account-button active' : 'account-button'} onClick={() => auth.setAccountPanelOpen((o) => !o)} type="button" aria-label="Account">
          <span aria-hidden="true">👤</span>
        </button>
      </header>
      {statusMessage ? <div className="alert success">{statusMessage}</div> : null}
      {errorMessage ? <div className="alert error">{errorMessage}</div> : null}
      {!auth.user || auth.accountPanelOpen ? (
        <section className="card stack account-panel">
          {auth.user ? (
            <>
              <div className="section-heading"><h2>Account</h2></div>
              <p className="muted-text"><strong>{auth.user.username}</strong> · {auth.user.email}</p>
              <div className="row gap wrap-row">
                <button type="button" onClick={() => void sessionAPI.bootstrapSession()}>Refresh</button>
                <button type="button" onClick={() => void auth.handleLogout()}>Log out</button>
              </div>
            </>
          ) : (
            <>
              <div className="tabs">
                <button className={auth.authMode === 'login' ? 'tab active' : 'tab'} onClick={() => auth.setAuthMode('login')} type="button">Log in</button>
                <button className={auth.authMode === 'register' ? 'tab active' : 'tab'} onClick={() => auth.setAuthMode('register')} type="button">Register</button>
              </div>
              {auth.authMode === 'login' ? (
                <form className="stack" onSubmit={auth.handleLogin}>
                  <label>Email or username<input value={auth.loginForm.identifier} onChange={(e) => auth.setLoginForm((c) => ({ ...c, identifier: e.target.value }))} placeholder="you@example.com" required /></label>
                  <label>Password<input type="password" value={auth.loginForm.password} onChange={(e) => auth.setLoginForm((c) => ({ ...c, password: e.target.value }))} placeholder="••••••••" required /></label>
                  <button className="primary-button" type="submit">Log in</button>
                </form>
              ) : (
                <form className="stack" onSubmit={auth.handleRegister}>
                  <label>Username<input value={auth.registerForm.username} onChange={(e) => auth.setRegisterForm((c) => ({ ...c, username: e.target.value }))} placeholder="alice" required /></label>
                  <label>Email<input type="email" value={auth.registerForm.email} onChange={(e) => auth.setRegisterForm((c) => ({ ...c, email: e.target.value }))} placeholder="you@example.com" required /></label>
                  <label>Password<input type="password" value={auth.registerForm.password} onChange={(e) => auth.setRegisterForm((c) => ({ ...c, password: e.target.value }))} placeholder="At least 8 characters" required /></label>
                  <button className="primary-button" type="submit">Create account</button>
                </form>
              )}
              {auth.verificationLink ? <div className="dev-note"><strong>Verification link</strong><a href={auth.verificationLink}>{auth.verificationLink}</a></div> : null}
            </>
          )}
        </section>
      ) : null}
      {auth.user ? (
        <>
          <nav className="card view-switcher" aria-label="Sections">
            <button className={activeView === 'use' ? 'nav-tab active' : 'nav-tab'} onClick={() => setActiveView('use')} type="button">Activity</button>
            <button className={activeView === 'progress' ? 'nav-tab active' : 'nav-tab'} onClick={() => setActiveView('progress')} type="button">Stats</button>
            <button className={activeView === 'edit' ? 'nav-tab active' : 'nav-tab'} onClick={() => setActiveView('edit')} type="button">Edit</button>
          </nav>
          {activeView === 'use' && <UseView categories={data.categories} selectedRecommendationCategoryIdSet={derived.selectedRecommendationCategoryIdSet} recommendation={recommendation} hasRequestedRecommendation={hasRequestedRecommendation} isTimingRecommendation={isTimingRecommendation} timerState={timer.timerState} timerNow={timer.timerNow} recommendationCategoryPickerOpen={lib.recommendationCategoryPickerOpen} categoriesByParent={derived.categoriesByParent} expandedRecommendationCategories={lib.expandedRecommendationCategories} onToggleRecommendationCategory={handleToggleRecommendationCategory} onToggleRecommendationCategoryExpansion={lib.toggleRecommendationCategoryExpansion} onExpandAllRecommendationCategories={() => lib.expandAllRecommendationCategories(data.categories)} onCollapseAllRecommendationCategories={() => lib.collapseAllRecommendationCategories(data.categories)} onRecommendActivity={handleRecommendActivity} onRecommendationTimer={handleRecommendationTimer} onSkipRecommendation={handleSkipRecommendation} onSetRecommendationCategoryPickerOpen={lib.setRecommendationCategoryPickerOpen} rootCategory={rootCategory} />}
          {activeView === 'progress' && <StatsView activityPieChart={activityPieChart} categoryProgress={derived.categoryProgress} categoriesByParent={derived.categoriesByParent} isDefaultCategory={isDefaultCategory} />}
          {activeView === 'edit' && <EditView categories={data.categories} categoryProgress={derived.categoryProgress} categoriesByParent={derived.categoriesByParent} activitiesByCategoryId={derived.activitiesByCategoryId} dragMultipliers={dragMultipliers} expandedLibraryCategories={lib.expandedLibraryCategories} rootCategory={rootCategory} onToggleLibraryCategory={lib.toggleLibraryCategory} onExpandAllLibraryCategories={() => lib.expandAllLibraryCategories(data.categories)} onCollapseAllLibraryCategories={lib.collapseAllLibraryCategories} onOpenLibrarySettings={lib.openLibrarySettings} onCreateCategoryInLibrary={createCategoryInLibrary} onCreateActivityInLibrary={createActivityInLibrary} onSaveMultiplier={saveMultiplier} onSaveCategoryMultiplier={saveCategoryMultiplier} onSetDragMultipliers={setDragMultipliers} />}
        </>
      ) : null}
      <SettingsModal target={lib.librarySettingsTarget} draft={lib.librarySettingsDraft} setDraft={lib.setLibrarySettingsDraft} categories={data.categories} categoryById={derived.categoryById} onClose={lib.closeLibrarySettings} onSave={handleSaveLibrarySettings} onDelete={handleDeleteLibraryTarget} isDefaultCategory={isDefaultCategory} displayCategoryPath={displayCategoryPath} />
      <CreateModal target={lib.libraryCreateTarget} draft={lib.libraryCreateDraft} setDraft={lib.setLibraryCreateDraft} onClose={lib.closeLibraryCreate} onSubmit={handleSubmitLibraryCreate} />
      <DeleteConfirmModal target={lib.deleteConfirmTarget} onCancel={() => lib.cancelDeleteLibraryTarget()} onConfirm={confirmDeleteLibraryTarget} />
    </div>
  );
}
