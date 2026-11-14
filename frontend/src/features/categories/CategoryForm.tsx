import { useState } from "react";
import type { FormEvent } from "react";

import type { Category, NewCategory } from "../../api/categories";

interface CategoryFormProps {
  categories: Category[];
  onSubmit: (input: NewCategory) => Promise<void>;
}

export function CategoryForm({ categories, onSubmit }: CategoryFormProps) {
  const [name, setName] = useState("");
  const [multiplier, setMultiplier] = useState("1");
  const [parentId, setParentId] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await onSubmit({
        name,
        multiplier: Number(multiplier),
        parentId: parentId === "" ? null : Number(parentId),
      });
      setName("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Something went wrong.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form className="card form" onSubmit={handleSubmit}>
      <h2>New category</h2>

      <label className="field">
        <span>Name</span>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          maxLength={100}
          required
        />
      </label>

      <label className="field">
        <span>Multiplier</span>
        <input
          type="number"
          inputMode="decimal"
          min="0.01"
          step="0.01"
          value={multiplier}
          onChange={(e) => setMultiplier(e.target.value)}
          required
        />
      </label>

      <label className="field">
        <span>Parent</span>
        <select value={parentId} onChange={(e) => setParentId(e.target.value)}>
          <option value="">None (top level)</option>
          {categories.map((category) => (
            <option key={category.id} value={category.id}>
              {category.name}
            </option>
          ))}
        </select>
      </label>

      {error && <p role="alert" className="message message--error">{error}</p>}

      <button type="submit" className="button" disabled={submitting}>
        {submitting ? "Saving…" : "Add category"}
      </button>
    </form>
  );
}
