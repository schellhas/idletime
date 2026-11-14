/** The longest time slot that can be entered: one day. */
export const MAX_SLOT_MINUTES = 24 * 60;

/** Parses a free-form time slot. Returns null unless it is a whole number of minutes from 1 to a day. */
export function parseSlotMinutes(input: string): number | null {
  const trimmed = input.trim();
  if (!/^\d+$/.test(trimmed)) {
    return null;
  }
  const minutes = Number(trimmed);
  return minutes >= 1 && minutes <= MAX_SLOT_MINUTES ? minutes : null;
}
