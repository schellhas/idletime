import {
  displayCategoryName,
  formatTimerDuration,
} from '../../utils/appUtils';

export function UseView({
  categories,
  selectedRecommendationCategoryIdSet,
  recommendation,
  hasRequestedRecommendation,
  isTimingRecommendation,
  timerState,
  timerNow,
  recommendationCategoryPickerOpen,
  categoriesByParent,
  expandedRecommendationCategories,
  onToggleRecommendationCategory,
  onToggleRecommendationCategoryExpansion,
  onExpandAllRecommendationCategories,
  onCollapseAllRecommendationCategories,
  onRecommendActivity,
  onRecommendationTimer,
  onSkipRecommendation,
  onSetRecommendationCategoryPickerOpen,
  rootCategory,
}) {
  function renderRecommendationCategoryTreeNode(category, depth = 0) {
    const childCategories = categoriesByParent[String(category.id)] ?? [];
    const isExpanded = expandedRecommendationCategories[category.id] ?? depth === 0;
    const isSelected = selectedRecommendationCategoryIdSet.has(String(category.id));
    const hasChildren = childCategories.length > 0;

    return (
      <div className="recommendation-tree-node" key={category.id}>
        <div
          className={isSelected ? 'recommendation-tree-item active' : 'recommendation-tree-item'}
          onClick={() => onToggleRecommendationCategory(category.id)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault();
              onToggleRecommendationCategory(category.id);
            }
          }}
          role="button"
          tabIndex={0}
        >
          <span
            className={hasChildren ? 'library-arrow recommendation-inline-toggle' : 'library-arrow recommendation-inline-toggle muted'}
            aria-hidden="true"
            onClick={(event) => {
              event.stopPropagation();
              if (hasChildren) {
                onToggleRecommendationCategoryExpansion(category.id);
              }
            }}
          >
            {hasChildren ? (isExpanded ? '▾' : '▸') : '•'}
          </span>
          <span className="library-icon" aria-hidden="true">📁</span>
          <span className="library-label">{displayCategoryName(category.name)}</span>
        </div>

        {isExpanded && hasChildren ? (
          <div className="recommendation-tree-children" role="group">
            {childCategories.map((childCategory) => renderRecommendationCategoryTreeNode(childCategory, depth + 1))}
          </div>
        ) : null}
      </div>
    );
  }

  return (
    <section className="stack">
      <article className="card stack">
        <div className="section-heading">
          <h2>Choose from categories</h2>
          <span className="pill">
            {selectedRecommendationCategoryIdSet.size} selected
          </span>
        </div>

        {categories.length === 0 ? (
          <p className="empty">Add a category first.</p>
        ) : (
          <div className="recommendation-picker stack">
            <button
              className="recommendation-picker-trigger"
              type="button"
              onClick={() => onSetRecommendationCategoryPickerOpen(!recommendationCategoryPickerOpen)}
              aria-expanded={recommendationCategoryPickerOpen}
            >
              Choose from categories
              <span aria-hidden="true">{recommendationCategoryPickerOpen ? '▴' : '▾'}</span>
            </button>

            {recommendationCategoryPickerOpen ? (
              <div className="recommendation-picker-panel stack" role="region" aria-label="Category picker">
                <div className="row gap small-gap wrap-row">
                  <button type="button" onClick={onExpandAllRecommendationCategories}>Expand all</button>
                  <button type="button" onClick={onCollapseAllRecommendationCategories}>Collapse all</button>
                </div>

                <div className="recommendation-tree" role="tree" aria-label="Categories">
                  {rootCategory
                    ? renderRecommendationCategoryTreeNode(rootCategory, 0)
                    : (categoriesByParent.root ?? []).map((category) => (
                      renderRecommendationCategoryTreeNode(category, 0)
                    ))}
                </div>
              </div>
            ) : null}
          </div>
        )}

        <div className="row gap wrap-row center-row">
          <button className="primary-button" type="button" onClick={onRecommendActivity}>
            Recommend activity
          </button>
        </div>
      </article>

      {hasRequestedRecommendation ? (
        <article className="card stack recommendation-card centered-card">
          <h2 className="center-text">{recommendation?.activity_name ?? 'No activity yet'}</h2>

          {recommendation ? (
            <div className="row gap wrap-row center-row">
              <button className="primary-button" type="button" onClick={onRecommendationTimer}>
                {isTimingRecommendation
                  ? `Stop timer (${formatTimerDuration(timerState.startedAt, timerNow)})`
                  : 'Start timer'}
              </button>
              <button type="button" onClick={onSkipRecommendation}>
                Skip
              </button>
            </div>
          ) : null}
        </article>
      ) : null}
    </section>
  );
}
