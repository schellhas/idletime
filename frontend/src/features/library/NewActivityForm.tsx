import { useState } from "react";
import type { FormEvent } from "react";

import type { NewActivity } from "../../api/activities";
import type { Category } from "../../api/categories";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";
import { categoryOptions } from "./libraryTree";

interface NewActivityFormProps {
  categories: Category[];
  onSubmit: (input: NewActivity) => Promise<void>;
}

export function NewActivityForm({ categories, onSubmit }: NewActivityFormProps) {
  const [name, setName] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [multiplier, setMultiplier] = useState("1");
  const [minimumMinutes, setMinimumMinutes] = useState("0");
  const { pending, error, run } = useAsyncAction();

  if (categories.length === 0) {
    return (
      <section className="card stack">
        <h2>New activity</h2>
        <Message>Create a category first, then add activities to it.</Message>
      </section>
    );
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    const saved = await run(() =>
      onSubmit({
        name,
        categoryId: Number(categoryId),
        multiplier: Number(multiplier),
        minimumMinutes: Number(minimumMinutes),
      }),
    );
    if (saved) {
      setName("");
    }
  }

  return (
    <form className="card stack" onSubmit={handleSubmit}>
      <h2>New activity</h2>
      <Field label="Name" value={name} onChange={(e) => setName(e.target.value)} maxLength={100} required />
      <label className="field">
        <span>Category</span>
        <select value={categoryId} onChange={(e) => setCategoryId(e.target.value)} required>
          <option value="" disabled>
            Choose a category
          </option>
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
        hint="How much time it should get compared to its siblings."
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
        hint="Shortest time slot in which this makes sense, including getting ready."
        required
      />
      {error && <Message kind="error">{error}</Message>}
      <button type="submit" className="button" disabled={pending}>
        {pending ? "Saving…" : "Add activity"}
      </button>
    </form>
  );
}
