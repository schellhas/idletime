import { useMemo, useState } from "react";
import type { FormEvent } from "react";

import type { Category, CategoryPatch } from "../../api/categories";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";
import { categoryOptions, subtreeIds } from "./libraryTree";

interface EditCategoryFormProps {
  category: Category;
  categories: Category[];
  onSave: (patch: CategoryPatch) => Promise<void>;
  onCancel: () => void;
}

export function EditCategoryForm({ category, categories, onSave, onCancel }: EditCategoryFormProps) {
  const [name, setName] = useState(category.name);
  const [multiplier, setMultiplier] = useState(String(category.multiplier));
  const [parentId, setParentId] = useState(category.parentId === null ? "" : String(category.parentId));
  const { pending, error, run } = useAsyncAction();

  // A category cannot become a child of itself or of one of its own subcategories.
  const parentChoices = useMemo(
    () => categoryOptions(categories, subtreeIds(categories, category.id)),
    [categories, category.id],
  );

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    const saved = await run(() =>
      onSave({ name, multiplier: Number(multiplier), parentId: parentId === "" ? null : Number(parentId) }),
    );
    if (saved) {
      onCancel();
    }
  }

  return (
    <form className="inline-form stack" onSubmit={handleSubmit}>
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
          {parentChoices.map((option) => (
            <option key={option.id} value={option.id}>
              {option.label}
            </option>
          ))}
        </select>
      </label>
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
