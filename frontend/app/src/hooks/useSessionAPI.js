import { useState, useCallback } from 'react';

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

export { apiFetch };

export function useSessionAPI(
  dataState,
  libState,
  recommendationState,
  setStatusMessage,
  setErrorMessage,
) {
  const [verificationHandled, setVerificationHandled] = useState(false);

  const loadOwnedData = useCallback(async (options = {}) => {
    const categoryIds = (options.categoryIds ?? libState.selectedRecommendationCategoryIds).map(String);
    const shouldLoad = options.includeRecommendation ?? recommendationState.hasRequested;
    try {
      const [cats, acts, entries] = await Promise.all([
        apiFetch('/categories'),
        apiFetch('/activities'),
        apiFetch('/time-entries'),
      ]);
      dataState.setCategories(cats.categories ?? []);
      dataState.setActivities(acts.activities ?? []);
      dataState.setTimeEntries(entries.time_entries ?? []);

      const validIds = new Set((cats.categories ?? []).map((c) => String(c.id)));
      const filtered = categoryIds.filter((v) => validIds.has(v));
      if (filtered.length !== libState.selectedRecommendationCategoryIds.length) {
        libState.setSelectedRecommendationCategoryIds(filtered);
      }

      if (!shouldLoad) {
        recommendationState.setRecommendation(null);
        recommendationState.setMessage('');
        return null;
      }

      const excluded = (options.excludeActivityIds ?? recommendationState.skipped).map(String);
      const params = new URLSearchParams();
      excluded.forEach((id) => params.append('exclude_activity_id', id));
      filtered.forEach((id) => params.append('category_id', id));
      const query = params.toString();
      const path = query ? `/recommendations?${query}` : '/recommendations';
      const rec = await apiFetch(path);
      recommendationState.setRecommendation(rec.recommendation ?? null);
      recommendationState.setMessage(rec.message ?? '');
      return rec.recommendation ?? null;
    } catch (error) {
      setErrorMessage(error.message);
      throw error;
    }
  }, [dataState, libState, recommendationState, setErrorMessage]);

  const bootstrapSession = useCallback(async () => {
    try {
      await apiFetch('/auth/user');
      await loadOwnedData();
      setStatusMessage('');
    } catch {
      dataState.clearOwnedData();
    }
  }, [dataState, loadOwnedData, setStatusMessage]);

  return {
    bootstrapSession,
    loadOwnedData,
    verificationHandled,
    setVerificationHandled,
    apiFetch,
  };
}
