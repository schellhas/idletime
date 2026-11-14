import { useState } from "react";
import type { FormEvent } from "react";

import { createTimeEntry } from "../../api/timeEntries";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";

interface LogTimeFormProps {
  activityId: number;
  defaultMinutes: number;
  onLogged: (minutes: number) => void;
  onCancel: () => void;
}

/** Logs time for an activity after the fact, without using the timer. */
export function LogTimeForm({ activityId, defaultMinutes, onLogged, onCancel }: LogTimeFormProps) {
  const [minutes, setMinutes] = useState(String(defaultMinutes));
  const [note, setNote] = useState("");
  const { pending, error, run } = useAsyncAction();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    const amount = Number(minutes);
    if (await run(() => createTimeEntry({ activityId, minutes: amount, note }))) {
      onLogged(amount);
    }
  }

  return (
    <form className="inline-form stack" onSubmit={handleSubmit}>
      <Field
        label="Minutes"
        type="number"
        inputMode="numeric"
        min="1"
        max="1440"
        step="1"
        value={minutes}
        onChange={(e) => setMinutes(e.target.value)}
        required
      />
      <Field label="Note (optional)" value={note} onChange={(e) => setNote(e.target.value)} maxLength={500} />
      {error && <Message kind="error">{error}</Message>}
      <div className="button-row">
        <button type="button" className="button button--secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={pending}>
          {pending ? "Saving…" : "Log time"}
        </button>
      </div>
    </form>
  );
}
