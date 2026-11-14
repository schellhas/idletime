/** 45 -> "45 min", 95 -> "1 h 35 min", 120 -> "2 h". */
export function formatMinutes(minutes: number): string {
  const rounded = Math.round(minutes);
  if (rounded < 60) {
    return `${rounded} min`;
  }
  const hours = Math.floor(rounded / 60);
  const rest = rounded % 60;
  return rest === 0 ? `${hours} h` : `${hours} h ${rest} min`;
}

/** Compact variant for chart axes: 30 -> "30m", 90 -> "1.5h", 120 -> "2h". */
export function formatMinutesShort(minutes: number): string {
  if (minutes < 60) {
    return `${minutes}m`;
  }
  const hours = minutes / 60;
  return `${Number.isInteger(hours) ? hours : hours.toFixed(1)}h`;
}

/** Elapsed seconds as a clock: 75 -> "1:15", 3725 -> "1:02:05". */
export function formatClock(totalSeconds: number): string {
  const seconds = Math.max(0, Math.floor(totalSeconds));
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  const pad = (n: number) => String(n).padStart(2, "0");
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`;
}

/** A share between 0 and 1 as a whole percentage: 0.456 -> "46%". */
export function formatPercent(share: number): string {
  return `${Math.round(share * 100)}%`;
}

/** "2026-06-07" -> "Jun 7" (no time zone shifting: the date is already a calendar day). */
export function formatDay(isoDate: string): string {
  const [year, month, day] = isoDate.split("-").map(Number);
  return new Date(year ?? 1970, (month ?? 1) - 1, day ?? 1).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

/** Timestamp as local date and time. */
export function formatDateTime(isoTimestamp: string): string {
  return new Date(isoTimestamp).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
