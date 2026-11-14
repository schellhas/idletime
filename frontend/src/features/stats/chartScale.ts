/** Axis maxima that read well as hours/minutes, ascending. */
const NICE_MAXIMA = [15, 30, 60, 90, 120, 180, 240, 300, 360, 480, 600, 720, 960, 1440];
const LARGE_STEP = 240;

/** The smallest "round" axis maximum (in minutes) that fits the given value. */
export function niceMax(maxMinutes: number): number {
  const fitting = NICE_MAXIMA.find((candidate) => candidate >= maxMinutes);
  return fitting ?? Math.ceil(maxMinutes / LARGE_STEP) * LARGE_STEP;
}

/** Gridline values for an axis from 0 to max: bottom, middle and top. */
export function axisTicks(max: number): number[] {
  return [0, max / 2, max];
}

/** Bar length in percent of the axis, clamped to 0-100. */
export function percentOf(value: number, max: number): number {
  return max <= 0 ? 0 : Math.min(100, Math.max(0, (value / max) * 100));
}
