import { useState } from "react";
import type { FormEvent } from "react";

import type { Activity, ActivityPatch } from "../../api/activities";
import type { Category } from "../../api/categories";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";
import { categoryOptions } from "./libraryTree";

interface EditActivityFormProps {
  activity: Activity;
  categories: Category[];
  onSave: (patch: ActivityPatch) => Promise<void>;
  onCancel: () => void;
}

export function EditActivityForm({ activity, categories, onSave, onCancel }: EditActivityFormProps) {
  const [name, setName] = useState(activity.name);
  const [categoryId, setCategoryId] = useState(String(activity.categoryId));
  const [multiplier, setMultiplier] = useState(String(activity.multiplier));
  const [minimumMinutes, setMinimumMinutes] = useState(String(activity.minimumMinutes));
  const { pending, error, run } = useAsyncAction();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    const saved = await run(() =>
      onSave({
        name,
        categoryId: Number(categoryId),
        multiplier: Number(multiplier),
        minimumMinutes: Number(minimumMinutes),
      }),
    );
    if (saved) {
      onCancel();
    }
  }

  return (
    <form className="inline-form stack" onSubmit={handleSubmit}>
      <Field label="Name" value={name} onChange={(e) => setName(e.target.value)} maxLength={100} required />
      <label className="field">
        <span>Category</span>
        <select value={categoryId} onChange={(e) => setCategoryId(e.target.value)} required>
          {categoryOptions(categories).map((option) => (
            <option key={option.id} value={option.id}>
              {option.label}
            </option>
          ))}
        </select>
      </label>
      <Field
        label="Multiplier"
        type="number"
        inputMode="decimal"
        min="0.01"
        step="0.01"
        value={multiplier}
        onChange={(e) => setMultiplier(e.target.value)}
        required
      />
      <Field
        label="Minimum time (minutes)"
        type="number"
        inputMode="numeric"
        min="0"
        max="1440"
        step="1"
        value={minimumMinutes}
        onChange={(e) => setMinimumMinutes(e.target.value)}
        required
      />
      {error && <Message kind="error">{error}</Message>}
      <div className="button-row">
        <button type="button" className="button button--secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={pending}>
          {pending ? "Saving…" : "Save"}
        </button>
      </div>
    </form>
  );
}
