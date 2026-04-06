import { useEffect, useMemo, useState } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080';

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

export default function App() {
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
      setStatusMessage(`Welcome back, ${response.user.username}.`);
    } catch {
      clearOwnedData();
      setUser(null);
    } finally {
      setBootstrapping(false);
    }
  }

  async function loadOwnedData(options = {}) {
    const recommendationParams = new URLSearchParams();
    if (options.excludeActivityId) {
      recommendationParams.set('exclude_activity_id', String(options.excludeActivityId));
    }
    const recommendationQuery = recommendationParams.toString();
    const recommendationPath = recommendationQuery
      ? `/recommendations?${recommendationQuery}`
      : '/recommendations';

    const [categoryResponse, activityResponse, timeEntryResponse, recommendationResponse] = await Promise.all([
      apiFetch('/categories'),
      apiFetch('/activities'),
      apiFetch('/time-entries'),
      apiFetch(recommendationPath),
    ]);

    setCategories(categoryResponse.categories ?? []);
    setActivities(activityResponse.activities ?? []);
    setTimeEntries(timeEntryResponse.time_entries ?? []);
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
      setStatusMessage(`Signed in as ${response.user.username}.`);
      setLoginForm({ identifier: '', password: '' });
      setVerificationLink('');
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

    try {
      const nextRecommendation = await loadOwnedData({ excludeActivityId: recommendation.activity_id });
      if (nextRecommendation?.activity_id) {
        setTimeEntryForm((current) => ({
          ...current,
          activityId: String(nextRecommendation.activity_id),
        }));
        setStatusMessage(`Skipped ${recommendation.activity_name}. ${nextRecommendation.activity_name} is next.`);
        return;
      }

      setStatusMessage(`Skipped ${recommendation.activity_name}. No alternative recommendation is available yet.`);
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function editCategory(category) {
    if (category.name === 'root') {
      setStatusMessage('The default root category stays available for uncategorized activities.');
      return;
    }

    const name = window.prompt('Category name', category.name);
    if (name === null) {
      return;
    }

    const multiplier = window.prompt('Multiplier', String(category.multiplier));
    if (multiplier === null) {
      return;
    }

    setErrorMessage('');

    try {
      await apiFetch(`/categories/${category.id}`, {
        method: 'PATCH',
        body: JSON.stringify({
          name,
          multiplier: Number(multiplier),
        }),
      });
      await loadOwnedData();
      setStatusMessage('Category updated.');
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function deleteCategory(category) {
    if (category.name === 'root') {
      setStatusMessage('The default root category cannot be deleted.');
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

  async function editActivity(activity) {
    const name = window.prompt('Activity name', activity.name);
    if (name === null) {
      return;
    }

    const multiplier = window.prompt('Multiplier', String(activity.multiplier));
    if (multiplier === null) {
      return;
    }

    const minimumMinutes = window.prompt('Minimum useful minutes', String(activity.minimum_minutes));
    if (minimumMinutes === null) {
      return;
    }

    setErrorMessage('');

    try {
      await apiFetch(`/activities/${activity.id}`, {
        method: 'PATCH',
        body: JSON.stringify({
          name,
          multiplier: Number(multiplier),
          minimum_minutes: Number(minimumMinutes),
        }),
      });
      await loadOwnedData();
      setStatusMessage('Activity updated.');
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

  if (bootstrapping) {
    return (
      <div className="app-shell">
        <section className="card hero-card">
          <h1>idletime</h1>
          <p>Loading your session…</p>
        </section>
      </div>
    );
  }

  return (
    <div className="app-shell">
      <header className="hero-card card">
        <div>
          <p className="eyebrow">React frontend</p>
          <h1>idletime</h1>
          <p>
            Track the things you want to do, keep your time balanced, and see your progress.
          </p>
        </div>
        <div className="hero-meta">
          <span className="pill">API: {API_BASE_URL}</span>
          {user ? <span className="pill success">Signed in</span> : <span className="pill">Signed out</span>}
        </div>
      </header>

      {statusMessage ? <div className="alert success">{statusMessage}</div> : null}
      {errorMessage ? <div className="alert error">{errorMessage}</div> : null}

      {!user ? (
        <section className="auth-layout">
          <div className="card">
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
          </div>

          <div className="card stack">
            <h2>How verification works</h2>
            <ol className="steps">
              <li>Create your account with email + password.</li>
              <li>Use the verification link from the response while in development.</li>
              <li>Come back and log in to access your own data.</li>
            </ol>

            {verificationLink ? (
              <div className="dev-note">
                <strong>Development verification link</strong>
                <a href={verificationLink}>{verificationLink}</a>
              </div>
            ) : null}
          </div>
        </section>
      ) : (
        <>
          <section className="card session-card">
            <div>
              <h2>Your session</h2>
              <p>
                <strong>{user.username}</strong> · {user.email}
              </p>
              <p>Email verified: {user.email_verified ? 'yes' : 'no'}</p>
            </div>
            <div className="row gap">
              <button type="button" onClick={handleRefreshData}>
                Refresh data
              </button>
              <button type="button" onClick={handleLogout}>
                Log out
              </button>
            </div>
          </section>

          <section className="panel-grid">
            <article className="card stack recommendation-card">
              <div className="recommendation-header">
                <div>
                  <p className="eyebrow">Recommended right now</p>
                  <h2>{recommendation?.activity_name ?? 'No recommendation yet'}</h2>
                </div>
                <span className={recommendation ? 'pill success' : 'pill'}>
                  {recommendation ? 'Most behind' : 'Waiting for data'}
                </span>
              </div>

              {recommendation ? (
                <>
                  <p>{recommendation.reason}</p>
                  <div className="recommendation-meta">
                    <span className="pill">Category: {recommendation.category_name}</span>
                    <span className="pill">Tracked: {recommendation.tracked_minutes} min</span>
                    <span className="pill">Minimum: {recommendation.minimum_minutes} min</span>
                    <span className="pill">Weight: {recommendation.combined_weight.toFixed(2)}</span>
                  </div>
                  <p className="muted-text">
                    Normalized progress: {recommendation.normalized_progress.toFixed(1)}
                  </p>
                  <div className="row gap">
                    <button className="primary-button" type="button" onClick={handleUseRecommendation}>
                      Use in time entry
                    </button>
                    <button type="button" onClick={handleSkipRecommendation}>
                      Skip
                    </button>
                  </div>
                </>
              ) : (
                <p className="empty">
                  {recommendationMessage || 'Add a category and at least one activity to get your first recommendation.'}
                </p>
              )}
            </article>

            <article className="card stack">
              <h2>Categories</h2>
              <form className="stack" onSubmit={handleCreateCategory}>
                <label>
                  Name
                  <input
                    value={categoryForm.name}
                    onChange={(event) => setCategoryForm((current) => ({ ...current, name: event.target.value }))}
                    placeholder="Sport"
                    required
                  />
                </label>
                <label>
                  Multiplier
                  <input
                    type="number"
                    min="0.1"
                    step="0.1"
                    value={categoryForm.multiplier}
                    onChange={(event) => setCategoryForm((current) => ({ ...current, multiplier: event.target.value }))}
                    required
                  />
                </label>
                <button className="primary-button" type="submit">Add category</button>
              </form>

              <ul className="item-list">
                {categories.length === 0 ? <li className="empty">No categories yet.</li> : null}
                {categories.map((category) => {
                  const isRootCategory = category.name === 'root';

                  return (
                    <li className="item-card" key={category.id}>
                      <div>
                        <strong>{category.name}</strong>
                        <p>Multiplier: {category.multiplier}</p>
                        {isRootCategory ? (
                          <p className="muted-text">Default category for activities that do not need a custom category.</p>
                        ) : null}
                        <small>Created {formatTimestamp(category.created_at)}</small>
                      </div>
                      {isRootCategory ? (
                        <div className="row gap small-gap">
                          <span className="pill">Default</span>
                        </div>
                      ) : (
                        <div className="row gap small-gap">
                          <button type="button" onClick={() => editCategory(category)}>Edit</button>
                          <button type="button" onClick={() => deleteCategory(category)}>Delete</button>
                        </div>
                      )}
                    </li>
                  );
                })}
              </ul>
            </article>

            <article className="card stack">
              <h2>Activities</h2>
              <form className="stack" onSubmit={handleCreateActivity}>
                <label>
                  Category
                  <select
                    value={activityForm.categoryId}
                    onChange={(event) => setActivityForm((current) => ({ ...current, categoryId: event.target.value }))}
                    required
                  >
                    {categories.length === 0 ? <option value="">Create a category first</option> : null}
                    {categories.map((category) => (
                      <option key={category.id} value={category.id}>
                        {category.name}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  Name
                  <input
                    value={activityForm.name}
                    onChange={(event) => setActivityForm((current) => ({ ...current, name: event.target.value }))}
                    placeholder="Swimming"
                    required
                  />
                </label>
                <div className="row split-row">
                  <label>
                    Multiplier
                    <input
                      type="number"
                      min="0.1"
                      step="0.1"
                      value={activityForm.multiplier}
                      onChange={(event) => setActivityForm((current) => ({ ...current, multiplier: event.target.value }))}
                      required
                    />
                  </label>
                  <label>
                    Minimum minutes
                    <input
                      type="number"
                      min="0"
                      step="1"
                      value={activityForm.minimumMinutes}
                      onChange={(event) => setActivityForm((current) => ({ ...current, minimumMinutes: event.target.value }))}
                      required
                    />
                  </label>
                </div>
                <button className="primary-button" type="submit" disabled={categories.length === 0}>Add activity</button>
              </form>

              <ul className="item-list">
                {activities.length === 0 ? <li className="empty">No activities yet.</li> : null}
                {activities.map((activity) => (
                  <li className="item-card" key={activity.id}>
                    <div>
                      <strong>{activity.name}</strong>
                      <p>
                        {categoryById[activity.category_id]?.name ?? 'Unknown category'} · multiplier {activity.multiplier}
                      </p>
                      <p>
                        Minimum {activity.minimum_minutes} min · tracked {activity.tracked_minutes} min
                      </p>
                    </div>
                    <div className="row gap small-gap">
                      <button type="button" onClick={() => editActivity(activity)}>Edit</button>
                      <button type="button" onClick={() => deleteActivity(activity)}>Delete</button>
                    </div>
                  </li>
                ))}
              </ul>
            </article>

            <article className="card stack">
              <h2>Time entries</h2>
              <form className="stack" onSubmit={handleCreateTimeEntry}>
                <label>
                  Activity
                  <select
                    value={timeEntryForm.activityId}
                    onChange={(event) => setTimeEntryForm((current) => ({ ...current, activityId: event.target.value }))}
                    required
                  >
                    {activities.length === 0 ? <option value="">Create an activity first</option> : null}
                    {activities.map((activity) => (
                      <option key={activity.id} value={activity.id}>
                        {activity.name}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  Minutes
                  <input
                    type="number"
                    min="1"
                    step="1"
                    value={timeEntryForm.minutes}
                    onChange={(event) => setTimeEntryForm((current) => ({ ...current, minutes: event.target.value }))}
                    required
                  />
                </label>
                <label>
                  Note
                  <textarea
                    rows="3"
                    value={timeEntryForm.note}
                    onChange={(event) => setTimeEntryForm((current) => ({ ...current, note: event.target.value }))}
                    placeholder="Pool session"
                  />
                </label>
                <button className="primary-button" type="submit" disabled={activities.length === 0}>Add time entry</button>
              </form>

              <ul className="item-list">
                {timeEntries.length === 0 ? <li className="empty">No time entries yet.</li> : null}
                {timeEntries.map((entry) => (
                  <li className="item-card" key={entry.id}>
                    <div>
                      <strong>{activityById[entry.activity_id]?.name ?? 'Unknown activity'}</strong>
                      <p>{entry.minutes} minutes</p>
                      {entry.note ? <p>{entry.note}</p> : null}
                      <small>{formatTimestamp(entry.created_at)}</small>
                    </div>
                    <div className="row gap small-gap">
                      <button type="button" onClick={() => editTimeEntry(entry)}>Edit</button>
                      <button type="button" onClick={() => deleteTimeEntry(entry)}>Delete</button>
                    </div>
                  </li>
                ))}
              </ul>
            </article>
          </section>
        </>
      )}
    </div>
  );
}

function currentValue(value) {
  return value ?? '';
}
