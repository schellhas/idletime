export default function DeleteConfirmModal({ target, onCancel, onConfirm }) {
  if (!target) {
    return null;
  }

  return (
    <div
      className="library-settings-backdrop"
      role="dialog"
      aria-modal="true"
      onClick={onCancel}
    >
      <div className="library-settings-modal card stack" onClick={(event) => event.stopPropagation()}>
        <div className="section-heading">
          <h2>Confirm delete</h2>
          <button type="button" onClick={onCancel} aria-label="Close confirmation">
            ✕
          </button>
        </div>

        <p className="muted-text">
          {target.type === 'category'
            ? `Delete category "${target.item.name}" and everything inside it?`
            : `Delete activity "${target.item.name}"?`}
        </p>

        <div className="row gap wrap-row">
          <button className="destructive-button" type="button" onClick={onConfirm}>
            {target.type === 'category' ? 'Delete category' : 'Delete activity'}
          </button>
          <button type="button" onClick={onCancel}>Cancel</button>
        </div>
      </div>
    </div>
  );
}
