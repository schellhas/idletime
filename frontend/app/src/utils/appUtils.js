const DEFAULT_CATEGORY_NAMES = new Set(['none', 'root']);

export function formatTimestamp(value) {
  if (!value) {
    return '—';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

export function formatMinutes(value) {
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

export function formatHoursFromMinutes(value) {
  const roundedHours = Math.round((Number(value ?? 0) / 60) * 10) / 10;
  if (roundedHours === 0) {
    return '0';
  }
  return Number.isInteger(roundedHours) ? `${roundedHours.toFixed(0)}h` : `${roundedHours.toFixed(1)}h`;
}

export function currentValue(value) {
  return value ?? '';
}

export function displayCategoryName(name) {
  const normalized = String(name ?? '').trim().toLowerCase();
  if (normalized === 'root' || normalized === 'none') {
    return 'Activities';
  }
  return String(name ?? 'Activities');
}

export function displayCategoryPath(category, categoryById) {
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

export function isDefaultCategory(category) {
  return DEFAULT_CATEGORY_NAMES.has(String(category?.name ?? '').trim().toLowerCase());
}

export function clampPercent(value, maxValue) {
  const safeValue = Math.max(0, Number(value ?? 0));
  if (safeValue === 0) {
    return 0;
  }

  const safeMax = Math.max(1, Number(maxValue ?? 0));
  return Math.min(100, Math.max(8, Math.round((safeValue / safeMax) * 100)));
}

export function sameIdList(left, right) {
  if (left.length !== right.length) {
    return false;
  }

  return left.every((value, index) => value === right[index]);
}

export function formatTimerDuration(startedAt, now) {
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

export function buildPieData(entries, options = {}) {
  const normalizedEntries = entries
    .map((entry) => ({
      ...entry,
      minutes: Math.max(0, Number(entry.minutes ?? 0)),
    }))
    .sort((left, right) => right.minutes - left.minutes || left.name.localeCompare(right.name));

  const includeZeroEntries = options.includeZeroEntries === true;
  const relevantEntries = includeZeroEntries
    ? normalizedEntries
    : normalizedEntries.filter((entry) => entry.minutes > 0);

  const totalMinutes = relevantEntries.reduce((sum, entry) => sum + entry.minutes, 0);
  const positiveTotalMinutes = relevantEntries.reduce(
    (sum, entry) => sum + (entry.minutes > 0 ? entry.minutes : 0),
    0,
  );

  let currentAngle = 0;
  const slices = relevantEntries.map((entry, index) => {
    const start = currentAngle;
    let end = currentAngle;
    if (positiveTotalMinutes > 0 && entry.minutes > 0) {
      const angle = (entry.minutes / positiveTotalMinutes) * 360;
      end = currentAngle + angle;
      currentAngle = end;
    }
    const color = PIE_COLORS[index % PIE_COLORS.length];
    const percent = positiveTotalMinutes > 0
      ? Math.round((entry.minutes / positiveTotalMinutes) * 100)
      : 0;

    return {
      ...entry,
      color,
      percent,
      gradientStop: `${color} ${start}deg ${end}deg`,
    };
  });

  const positiveSlices = slices.filter((slice) => slice.minutes > 0);

  return {
    totalMinutes,
    gradient: positiveSlices.length > 0
      ? `conic-gradient(${positiveSlices.map((slice) => slice.gradientStop).join(', ')})`
      : 'conic-gradient(#dbe6f3 0deg 360deg)',
    slices,
  };
}

export function buildActivityPieData(activityList) {
  return buildPieData(
    activityList.map((activity) => ({
      id: `activity-${activity.id}`,
      name: activity.name,
      minutes: Math.max(0, Number(activity.tracked_minutes ?? activity.trackedDisplayMinutes ?? 0)),
    })),
    { includeZeroEntries: true },
  );
}
