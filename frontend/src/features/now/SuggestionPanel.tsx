import { useState } from "react";

import type { Category } from "../../api/categories";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { formatMinutes } from "../../lib/format";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";
import { categoryOptions } from "../library/libraryTree";
import { LogTimeForm } from "./LogTimeForm";
import { MAX_SLOT_MINUTES, parseSlotMinutes } from "./slot";
import { useRecommendation } from "./useRecommendation";

/** Time slots offered as one-tap choices. 0 means "no limit". */
const PRESETS = [15, 30, 45, 60, 90, 0] as const;
const DEFAULT_SLOT_MINUTES = 30;

interface SuggestionPanelProps {
  categories: Category[];
  /** Starts the timer for the chosen activity. */
  onStart: (activityId: number) => Promise<void>;
  /** Called after time was logged manually, so other views can refresh. */
  onLogged: (minutes: number, activityName: string) => void;
  /** Bumped by the page whenever totals changed, to fetch a fresh suggestion. */
  refreshSignal: number;
}

export function SuggestionPanel({ categories, onStart, onLogged, refreshSignal }: SuggestionPanelProps) {
  const [availableMinutes, setAvailableMinutes] = useState(DEFAULT_SLOT_MINUTES);
  // What the user typed into the free input; empty while a preset is chosen.
  const [customMinutes, setCustomMinutes] = useState("");
  const [categoryId, setCategoryId] = useState<number | undefined>(undefined);
  const [logging, setLogging] = useState(false);

  const { state, skip, resetSkips, skippedCount, refresh } = useRecommendation({ availableMinutes, categoryId });
  const start = useAsyncAction();

  function choosePreset(minutes: number) {
    setCustomMinutes("");
    setAvailableMinutes(minutes);
  }

  function typeCustom(value: string) {
    setCustomMinutes(value);
    const minutes = parseSlotMinutes(value);
    if (minutes !== null) {
      setAvailableMinutes(minutes);
    }
  }

  // Re-ask after the page logged time elsewhere.
  const [seenSignal, setSeenSignal] = useState(refreshSignal);
  if (seenSignal !== refreshSignal) {
    setSeenSignal(refreshSignal);
    refresh();
  }

  return (
    <section className="card stack">
      <h2>What should I do?</h2>

      <fieldset className="chips">
        <legend>How much time do you have?</legend>
        {PRESETS.map((minutes) => (
          <button
            key={minutes}
            type="button"
            className={`chip${customMinutes === "" && availableMinutes === minutes ? " chip--active" : ""}`}
            aria-pressed={customMinutes === "" && availableMinutes === minutes}
            onClick={() => choosePreset(minutes)}
          >
            {minutes === 0 ? "Any" : formatMinutes(minutes)}
          </button>
        ))}
      </fieldset>

      <Field
        label="Or enter your own time (minutes)"
        type="number"
        inputMode="numeric"
        min="1"
        max={MAX_SLOT_MINUTES}
        step="1"
        placeholder="e.g. 25"
        value={customMinutes}
        onChange={(e) => typeCustom(e.target.value)}
      />

      {categories.length > 0 && (
        <label className="field">
          <span>Only from</span>
          <select
            value={categoryId ?? ""}
            onChange={(e) => setCategoryId(e.target.value === "" ? undefined : Number(e.target.value))}
          >
            <option value="">Everything</option>
            {categoryOptions(categories).map((option) => (
              <option key={option.id} value={option.id}>
                {option.label}
              </option>
            ))}
          </select>
        </label>
      )}

      <hr />

      {state.status === "loading" && <Message>Thinking…</Message>}
      {state.status === "error" && <Message kind="error">{state.message}</Message>}

      {state.status === "none" && (
        <div className="stack">
          <Message>
            {skippedCount > 0
              ? "You skipped everything that fits."
              : "Nothing fits right now. Add activities in the library, or choose a longer time slot."}
          </Message>
          {skippedCount > 0 && (
            <button type="button" className="button button--secondary" onClick={resetSkips}>
              Show skipped again
            </button>
          )}
        </div>
      )}

      {state.status === "found" && (
        <div className="stack suggestion">
          {state.recommendation.path.length > 0 && (
            <p className="muted">{state.recommendation.path.join(" › ")}</p>
          )}
          <p className="suggestion__name">{state.recommendation.activity.name}</p>
          <p className="muted">
            {state.recommendation.activity.minimumMinutes > 0 &&
              `Needs at least ${formatMinutes(state.recommendation.activity.minimumMinutes)} · `}
            {formatMinutes(state.recommendation.activity.trackedMinutes)} tracked so far
          </p>

          {start.error && <Message kind="error">{start.error}</Message>}

          {logging ? (
            <LogTimeForm
              activityId={state.recommendation.activity.id}
              defaultMinutes={availableMinutes > 0 ? availableMinutes : Math.max(state.recommendation.activity.minimumMinutes, 30)}
              onCancel={() => setLogging(false)}
              onLogged={(minutes) => {
                setLogging(false);
                onLogged(minutes, state.recommendation.activity.name);
              }}
            />
          ) : (
            <div className="stack">
              <button
                type="button"
                className="button"
                disabled={start.pending}
                onClick={() => void start.run(() => onStart(state.recommendation.activity.id))}
              >
                {start.pending ? "Starting…" : "Start timer"}
              </button>
              <div className="button-row">
                <button type="button" className="button button--secondary" onClick={() => skip(state.recommendation.activity.id)}>
                  Skip
                </button>
                <button type="button" className="button button--secondary" onClick={() => setLogging(true)}>
                  Log time
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
