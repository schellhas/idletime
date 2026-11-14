import { useState } from "react";
import type { FormEvent } from "react";

import type { Category, NewCategory } from "../../api/categories";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";

interface CategoryFormProps {
  categories: Category[];
  onSubmit: (input: NewCategory) => Promise<void>;
}

export function CategoryForm({ categories, onSubmit }: CategoryFormProps) {
  const [name, setName] = useState("");
  const [multiplier, setMultiplier] = useState("1");
  const [parentId, setParentId] = useState("");
  const { pending, error, run } = useAsyncAction();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    const saved = await run(() =>
      onSubmit({
        name,
        multiplier: Number(multiplier),
        parentId: parentId === "" ? null : Number(parentId),
      }),
    );
    if (saved) {
      setName("");
    }
  }

  return (
    <form className="card stack" onSubmit={handleSubmit}>
      <h2>New category</h2>

      <Field label="Name" value={name} onChange={(e) => setName(e.target.value)} maxLength={100} required />

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

      {error && <Message kind="error">{error}</Message>}

      <button type="submit" className="button" disabled={pending}>
        {pending ? "Saving…" : "Add category"}
      </button>
    </form>
  );
}
