import {
  displayCategoryName,
  isDefaultCategory,
} from '../../utils/appUtils';

export function EditView({
  categories,
  categoryProgress,
  categoriesByParent,
  activitiesByCategoryId,
  dragMultipliers,
  expandedLibraryCategories,
  rootCategory,
  onToggleLibraryCategory,
  onExpandAllLibraryCategories,
  onCollapseAllLibraryCategories,
  onOpenLibrarySettings,
  onCreateCategoryInLibrary,
  onCreateActivityInLibrary,
  onSaveMultiplier,
  onSaveCategoryMultiplier,
  onSetDragMultipliers,
}) {
  function renderLibraryCategory(category, depth = 0) {
    const childCategories = categoriesByParent[String(category.id)] ?? [];
    const categoryActivities = activitiesByCategoryId[String(category.id)] ?? [];
    const isExpanded = expandedLibraryCategories[category.id] ?? depth === 0;

    return (
      <div className="library-node" key={category.id}>
        <div className="library-row">
          <div
            className="library-row-main"
            onClick={() => onToggleLibraryCategory(category.id)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                onToggleLibraryCategory(category.id);
              }
            }}
            role="button"
            tabIndex={0}
            aria-expanded={isExpanded}
          >
            <span className="library-arrow" aria-hidden="true">{isExpanded ? '▾' : '▸'}</span>
            <span className="library-icon" aria-hidden="true">{isExpanded ? '📂' : '📁'}</span>
            <span className="library-label">{displayCategoryName(category.name)}</span>
            <button
              className="library-settings-btn"
              type="button"
              aria-label={`Settings for ${displayCategoryName(category.name)}`}
              onClick={(event) => {
                event.stopPropagation();
                onOpenLibrarySettings('category', category);
              }}
            >
              ⚙︎
            </button>
          </div>
        </div>

        {isExpanded ? (
          <div className="library-children" role="group">
            {childCategories.map((childCategory) => renderLibraryCategory(childCategory, depth + 1))}
            {categoryActivities.map((activity) => (
              <div className="library-child" key={activity.id}>
                <span className="library-arrow placeholder" aria-hidden="true">•</span>
                <span className="library-icon" aria-hidden="true">📄</span>
                <span className="library-label">{activity.name}</span>
                <button
                  className="library-settings-btn"
                  type="button"
                  aria-label={`Settings for ${activity.name}`}
                  onClick={() => onOpenLibrarySettings('activity', activity)}
                >
                  ⚙︎
                </button>
              </div>
            ))}

            <button
              className="library-ghost-row"
              onClick={() => void onCreateCategoryInLibrary(category)}
              type="button"
            >
              <span className="library-arrow placeholder" aria-hidden="true">+</span>
              <span className="library-icon" aria-hidden="true">📁</span>
              <span className="library-label">New Category</span>
            </button>

            <button
              className="library-ghost-row"
              onClick={() => void onCreateActivityInLibrary(category)}
              type="button"
            >
              <span className="library-arrow placeholder" aria-hidden="true">+</span>
              <span className="library-icon" aria-hidden="true">📄</span>
              <span className="library-label">New Activity</span>
            </button>
          </div>
        ) : null}
      </div>
    );
  }

  return (
    <section className="stack">
      <article className="card stack">
        <div className="section-heading">
          <h2>Category tree</h2>
          <div className="row gap small-gap wrap-row">
            <button type="button" onClick={onExpandAllLibraryCategories}>Expand all</button>
            <button type="button" onClick={onCollapseAllLibraryCategories}>Collapse all</button>
          </div>
        </div>

        <div className="library-tree" role="tree" aria-label="Category library">
          {rootCategory ? renderLibraryCategory(rootCategory, 0) : (
            <button className="library-ghost-row" onClick={() => void onCreateCategoryInLibrary()} type="button">
              <span className="library-arrow placeholder" aria-hidden="true">+</span>
              <span className="library-icon" aria-hidden="true">📁</span>
              <span className="library-label">New Category</span>
            </button>
          )}
        </div>
      </article>

      {(() => {
        const progressById = Object.fromEntries(categoryProgress.map((g) => [g.id, g]));
        function renderEditGroup(group) {
          const childCategories = categoriesByParent[String(group.id)] ?? [];
          const childGroups = childCategories.map((c) => progressById[c.id]).filter(Boolean);
          const selfDragKey = `category-${group.id}`;
          const selfMultiplier = dragMultipliers[selfDragKey] ?? Number(group.multiplier ?? 1);
          const faderItems = [
            ...childGroups.map((childGroup) => ({
              type: 'category',
              id: childGroup.id,
              name: displayCategoryName(childGroup.name),
              multiplier: Number(childGroup.multiplier ?? 1),
            })),
            ...group.activities.map((activity) => ({
              type: 'activity',
              id: activity.id,
              name: activity.name,
              multiplier: Number(activity.multiplier ?? 1),
            })),
          ];
          const rawMax = Math.max(1, selfMultiplier, ...faderItems.map((item) => item.multiplier));
          const maxScale = Math.max(rawMax, 5);
          const sorted = [...faderItems].sort(
            (a, b) => b.multiplier - a.multiplier || a.name.localeCompare(b.name),
          );
          const selfPct = Math.min(100, Math.round((selfMultiplier / maxScale) * 100));
          return (
            <article className="card stack" key={group.id}>
              <div className="section-heading">
                <h2>{displayCategoryName(group.name)}</h2>
                {isDefaultCategory(group) ? <span className="pill">Default</span> : null}
              </div>
              <div className="edit-multiplier-row">
                <span className="edit-multiplier-name">📁 {displayCategoryName(group.name)}</span>
                <div
                  className="edit-multiplier-track"
                  onPointerDown={(e) => {
                    e.currentTarget.setPointerCapture(e.pointerId);
                    const rect = e.currentTarget.getBoundingClientRect();
                    const val = Math.max(0, Math.round(((e.clientX - rect.left) / rect.width) * maxScale * 10) / 10);
                    onSetDragMultipliers((m) => ({ ...m, [selfDragKey]: val }));
                  }}
                  onPointerMove={(e) => {
                    if (!e.currentTarget.hasPointerCapture(e.pointerId)) return;
                    const rect = e.currentTarget.getBoundingClientRect();
                    const val = Math.max(0, Math.round(((e.clientX - rect.left) / rect.width) * maxScale * 10) / 10);
                    onSetDragMultipliers((m) => ({ ...m, [selfDragKey]: val }));
                  }}
                  onPointerUp={(e) => {
                    if (!e.currentTarget.hasPointerCapture(e.pointerId)) return;
                    e.currentTarget.releasePointerCapture(e.pointerId);
                    const rect = e.currentTarget.getBoundingClientRect();
                    const val = Math.max(0, Math.round(((e.clientX - rect.left) / rect.width) * maxScale * 10) / 10);
                    void onSaveCategoryMultiplier(group.id, val);
                  }}
                  onPointerCancel={() => {
                    onSetDragMultipliers((m) => { const next = { ...m }; delete next[selfDragKey]; return next; });
                  }}
                >
                  <div className="edit-multiplier-fill" style={{ width: `${selfPct}%` }} />
                </div>
                <input
                  className="edit-multiplier-value no-spin"
                  type="number"
                  min="0"
                  step="0.1"
                  value={dragMultipliers[selfDragKey] !== undefined ? selfMultiplier.toFixed(1) : selfMultiplier}
                  onChange={(e) => {
                    onSetDragMultipliers((m) => ({ ...m, [selfDragKey]: Number(e.target.value) }));
                  }}
                  onBlur={(e) => {
                    const val = Math.max(0, Math.round(Number(e.target.value) * 10) / 10);
                    void onSaveCategoryMultiplier(group.id, val);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') e.currentTarget.blur();
                    if (e.key === 'Escape') {
                      onSetDragMultipliers((m) => { const next = { ...m }; delete next[selfDragKey]; return next; });
                      e.currentTarget.blur();
                    }
                  }}
                />
              </div>
              {sorted.length === 0 ? (
                <p className="empty">No activities in this category yet.</p>
              ) : (
                <div className="item-list">
                  {sorted.map((activity) => {
                    const dragKey = `${activity.type}-${activity.id}`;
                    const activeMultiplier = dragMultipliers[dragKey] ?? Number(activity.multiplier ?? 1);
                    const pct = Math.min(100, Math.round((activeMultiplier / maxScale) * 100));
                    return (
                      <div className="edit-multiplier-row" key={dragKey}>
                        <span className="edit-multiplier-name">
                          {activity.type === 'category' ? '📁 ' : '📄 '}
                          {activity.name}
                        </span>
                        <div
                          className="edit-multiplier-track"
                          onPointerDown={(e) => {
                            e.currentTarget.setPointerCapture(e.pointerId);
                            const rect = e.currentTarget.getBoundingClientRect();
                            const val = Math.max(0, Math.round(((e.clientX - rect.left) / rect.width) * maxScale * 10) / 10);
                            onSetDragMultipliers((m) => ({ ...m, [dragKey]: val }));
                          }}
                          onPointerMove={(e) => {
                            if (!e.currentTarget.hasPointerCapture(e.pointerId)) return;
                            const rect = e.currentTarget.getBoundingClientRect();
                            const val = Math.max(0, Math.round(((e.clientX - rect.left) / rect.width) * maxScale * 10) / 10);
                            onSetDragMultipliers((m) => ({ ...m, [dragKey]: val }));
                          }}
                          onPointerUp={(e) => {
                            if (!e.currentTarget.hasPointerCapture(e.pointerId)) return;
                            e.currentTarget.releasePointerCapture(e.pointerId);
                            const rect = e.currentTarget.getBoundingClientRect();
                            const val = Math.max(0, Math.round(((e.clientX - rect.left) / rect.width) * maxScale * 10) / 10);
                            if (activity.type === 'category') {
                              void onSaveCategoryMultiplier(activity.id, val);
                            } else {
                              void onSaveMultiplier(activity.id, val);
                            }
                          }}
                          onPointerCancel={(e) => {
                            onSetDragMultipliers((m) => { const next = { ...m }; delete next[dragKey]; return next; });
                          }}
                        >
                          <div className="edit-multiplier-fill" style={{ width: `${pct}%` }} />
                        </div>
                        <input
                          className="edit-multiplier-value no-spin"
                          type="number"
                          min="0"
                          step="0.1"
                          value={dragMultipliers[dragKey] !== undefined ? activeMultiplier.toFixed(1) : activeMultiplier}
                          onChange={(e) => {
                            onSetDragMultipliers((m) => ({ ...m, [dragKey]: Number(e.target.value) }));
                          }}
                          onBlur={(e) => {
                            const val = Math.max(0, Math.round(Number(e.target.value) * 10) / 10);
                            if (activity.type === 'category') {
                              void onSaveCategoryMultiplier(activity.id, val);
                            } else {
                              void onSaveMultiplier(activity.id, val);
                            }
                          }}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') e.currentTarget.blur();
                            if (e.key === 'Escape') {
                              onSetDragMultipliers((m) => { const next = { ...m }; delete next[dragKey]; return next; });
                              e.currentTarget.blur();
                            }
                          }}
                        />
                      </div>
                    );
                  })}
                </div>
              )}

              {childGroups.length > 0 && (
                <div className="nested-category-cards">
                  {childGroups.map(renderEditGroup)}
                </div>
              )}
            </article>
          );
        }
        const topLevel = (categoriesByParent.root ?? [])
          .map((c) => progressById[c.id]).filter(Boolean);
        if (topLevel.length === 0) {
          return (
            <article className="card">
              <p className="empty">Add a category and activities to see multiplier charts here.</p>
            </article>
          );
        }
        return <div className="progress-grid">{topLevel.map(renderEditGroup)}</div>;
      })()}
    </section>
  );
}
