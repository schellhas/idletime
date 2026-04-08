export default function CreateModal({ target, draft, setDraft, onClose, onSubmit }) {
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
      <form className="library-settings-modal card stack" onClick={(event) => event.stopPropagation()} onSubmit={onSubmit}>
        <div className="section-heading">
          <h2>{target.type === 'category' ? 'New category' : 'New activity'}</h2>
          <button type="button" onClick={onClose} aria-label="Close create dialog">
            ✕
          </button>
        </div>

        <p className="muted-text">
          Inside {target.parentLabel}
        </p>

        <label>
          Name
          <input
            autoFocus
            value={String(draft.name ?? '')}
            onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))}
          />
        </label>

        <label>
          Multiplier
          <input
            type="number"
            min="0"
            step="0.1"
            value={String(draft.multiplier ?? '1')}
            onChange={(event) => setDraft((current) => ({ ...current, multiplier: event.target.value }))}
          />
        </label>

        {target.type === 'activity' && (
          <>
            <label>
              Minimum minutes
              <input
                type="number"
                min="0"
                step="1"
                value={String(draft.minimumMinutes ?? '0')}
                onChange={(event) => setDraft((current) => ({ ...current, minimumMinutes: event.target.value }))}
              />
            </label>
          </>
        )}

        <div className="row gap wrap-row">
          <button className="primary-button" type="submit">Create</button>
          <button type="button" onClick={onClose}>Cancel</button>
        </div>
      </form>
    </div>
  );
}
