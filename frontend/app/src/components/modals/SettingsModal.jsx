export default function SettingsModal({
  target,
  draft,
  setDraft,
  categories,
  categoryById,
  onClose,
  onSave,
  onDelete,
  isDefaultCategory,
  displayCategoryPath,
}) {
  if (!target) {
    return null;
  }

  return (
    <div
      className="library-settings-backdrop"
      role="dialog"
      aria-modal="true"
      onClick={onClose}
    >
      <div className="library-settings-modal card stack" onClick={(event) => event.stopPropagation()}>
        <div className="section-heading">
          <h2>
            {target.type === 'category' ? 'Category settings' : 'Activity settings'}
          </h2>
          <button type="button" onClick={onClose} aria-label="Close settings">
            ✕
          </button>
        </div>

        <label>
          Name
          <input
            value={String(draft.name ?? '')}
            onChange={(event) => setDraft((current) => ({
              ...current,
              name: event.target.value,
            }))}
          />
        </label>

        <label>
          Multiplier
          <input
            className="no-spin"
            type="number"
            min="0"
            step="0.1"
            value={String(draft.multiplier ?? '1')}
            onChange={(event) => setDraft((current) => ({
              ...current,
              multiplier: event.target.value,
            }))}
          />
        </label>

        {target.type === 'category' ? (
          <label>
            Parent category
            <select
              value={String(draft.parentId ?? '')}
              onChange={(event) => setDraft((current) => ({
                ...current,
                parentId: event.target.value,
              }))}
            >
              <option value="">Library root</option>
              {categories
                .filter((category) => category.id !== target.item.id)
                .map((category) => (
                  <option key={category.id} value={String(category.id)}>
                    {displayCategoryPath(category, categoryById)}
                  </option>
                ))}
            </select>
          </label>
        ) : (
          <>
            <label>
              Minimum minutes
              <input
                className="no-spin"
                type="number"
                min="0"
                value={String(draft.minimumMinutes ?? '0')}
                onChange={(event) => setDraft((current) => ({
                  ...current,
                  minimumMinutes: event.target.value,
                }))}
              />
            </label>

            <label>
              Tracked minutes
              <input
                className="no-spin"
                type="number"
                min="0"
                value={String(draft.trackedMinutes ?? '0')}
                onChange={(event) => setDraft((current) => ({
                  ...current,
                  trackedMinutes: event.target.value,
                }))}
              />
            </label>

            <label>
              Category
              <select
                value={String(draft.categoryId ?? '')}
                onChange={(event) => setDraft((current) => ({
                  ...current,
                  categoryId: event.target.value,
                }))}
              >
                {categories.map((category) => (
                  <option key={category.id} value={String(category.id)}>
                    {displayCategoryPath(category, categoryById)}
                  </option>
                ))}
              </select>
            </label>
          </>
        )}

        <div className="row gap wrap-row">
          <button className="primary-button" type="button" onClick={onSave}>
            Save
          </button>
          <button
            className="destructive-button"
            type="button"
            onClick={onDelete}
            disabled={target.type === 'category' && isDefaultCategory(target.item)}
            title={target.type === 'category' && isDefaultCategory(target.item)
              ? 'Root category cannot be deleted'
              : ''}
          >
            {target.type === 'category' ? 'Delete category' : 'Delete activity'}
          </button>
          <button type="button" onClick={onClose}>Cancel</button>
        </div>
      </div>
    </div>
  );
}
