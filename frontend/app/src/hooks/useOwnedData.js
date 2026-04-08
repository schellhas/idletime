import { useState } from 'react';

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

export function useOwnedData() {
  const [categories, setCategories] = useState([]);
  const [activities, setActivities] = useState([]);
  const [timeEntries, setTimeEntries] = useState([]);
  const [errorMessage, setErrorMessage] = useState('');

  async function loadOwnedData(options = {}) {
    const { includeRecommendation, excludeActivityIds = [], categoryIds = [] } = options;

    try {
      const response = await apiFetch('/owned-data');
      setCategories(response.categories ?? []);
      setActivities(response.activities ?? []);
      setTimeEntries(response.time_entries ?? []);
    } catch (error) {
      setErrorMessage(error.message);
      throw error;
    }
  }

  function clearOwnedData() {
    setCategories([]);
    setActivities([]);
    setTimeEntries([]);
  }

  async function createCategory(name, multiplier, parentId) {
    setErrorMessage('');
    try {
      await apiFetch('/categories', {
        method: 'POST',
        body: JSON.stringify({ name, multiplier, parent_id: parentId ?? 0 }),
      });
      await loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
      throw error;
    }
  }

  async function updateCategory(categoryId, updates) {
    setErrorMessage('');
    try {
      await apiFetch(`/categories/${categoryId}`, {
        method: 'PATCH',
        body: JSON.stringify(updates),
      });
      await loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
      throw error;
    }
  }

  async function deleteCategory(categoryId) {
    setErrorMessage('');
    try {
      await apiFetch(`/categories/${categoryId}`, { method: 'DELETE' });
      await loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
      throw error;
    }
  }

  async function createActivity(categoryId, name, multiplier, minimumMinutes) {
    setErrorMessage('');
    try {
      await apiFetch('/activities', {
        method: 'POST',
        body: JSON.stringify({
          category_id: categoryId,
          name,
          multiplier,
          minimum_minutes: minimumMinutes,
        }),
      });
      await loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
      throw error;
    }
  }

  async function updateActivity(activityId, updates) {
    setErrorMessage('');
    try {
      await apiFetch(`/activities/${activityId}`, {
        method: 'PATCH',
        body: JSON.stringify(updates),
      });
      await loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
      throw error;
    }
  }

  async function deleteActivity(activityId) {
    setErrorMessage('');
    try {
      await apiFetch(`/activities/${activityId}`, { method: 'DELETE' });
      await loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
      throw error;
    }
  }

  async function createTimeEntry(activityId, minutes, note) {
    setErrorMessage('');
    try {
      await apiFetch('/time-entries', {
        method: 'POST',
        body: JSON.stringify({ activity_id: activityId, minutes, note }),
      });
      await loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
      throw error;
    }
  }

  async function updateTimeEntry(timeEntryId, minutes, note) {
    setErrorMessage('');
    try {
      await apiFetch(`/time-entries/${timeEntryId}`, {
        method: 'PATCH',
        body: JSON.stringify({ minutes, note }),
      });
      await loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
      throw error;
    }
  }

  async function deleteTimeEntry(timeEntryId) {
    setErrorMessage('');
    try {
      await apiFetch(`/time-entries/${timeEntryId}`, { method: 'DELETE' });
      await loadOwnedData();
    } catch (error) {
      setErrorMessage(error.message);
      throw error;
    }
  }

  return {
    categories,
    setCategories,
    activities,
    setActivities,
    timeEntries,
    setTimeEntries,
    errorMessage,
    setErrorMessage,
    loadOwnedData,
    clearOwnedData,
    createCategory,
    updateCategory,
    deleteCategory,
    createActivity,
    updateActivity,
    deleteActivity,
    createTimeEntry,
    updateTimeEntry,
    deleteTimeEntry,
  };
}
