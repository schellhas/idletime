import {
  buildPieData,
  displayCategoryName,
  formatHoursFromMinutes,
  formatMinutes,
  isDefaultCategory,
} from '../../utils/appUtils';

export function StatsView({
  activityPieChart,
  categoryProgress,
  categoriesByParent,
  isDefaultCategory: isDefCat,
}) {
  return (
    <section className="stack">
      <article className="card stack">
        <div className="section-heading">
          <h2>All activities</h2>
          <span className="pill">{formatMinutes(activityPieChart.totalMinutes)} tracked</span>
        </div>

        {activityPieChart.slices.length > 0 ? (
          <div className="activity-pie-layout">
            <div
              className="activity-pie"
              aria-label="Activity time distribution pie chart"
              role="img"
              style={{ background: activityPieChart.gradient }}
            />

            <div className="activity-pie-legend">
              {activityPieChart.slices.map((slice) => (
                <div className="activity-pie-legend-item" key={slice.id}>
                  <span
                    className="activity-pie-color"
                    aria-hidden="true"
                    style={{ background: slice.color }}
                  />
                  <span className="activity-pie-label">{slice.name}</span>
                  <span className="activity-pie-value">{formatHoursFromMinutes(slice.minutes)}</span>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <p className="empty">No activities yet.</p>
        )}
      </article>

      {(() => {
        const progressById = Object.fromEntries(categoryProgress.map((g) => [g.id, g]));
        function renderStatsGroup(group) {
          const childCategories = categoriesByParent[String(group.id)] ?? [];
          const childGroups = childCategories.map((c) => progressById[c.id]).filter(Boolean);
          const groupPieChart = buildPieData([
            ...group.activities.map((activity) => ({
              id: `activity-${activity.id}`,
              name: activity.name,
              minutes: Number(activity.tracked_minutes ?? activity.trackedDisplayMinutes ?? 0),
            })),
            ...childGroups.map((childGroup) => ({
              id: `category-${childGroup.id}`,
              name: displayCategoryName(childGroup.name),
              minutes: Number(childGroup.totalTrackedMinutes ?? 0),
            })),
          ], { includeZeroEntries: true });
          return (
            <article className="card stack" key={group.id}>
              <div className="section-heading">
                <h2>{displayCategoryName(group.name)}</h2>
                <div className="row gap small-gap wrap-row">
                  <span className="pill success">{formatMinutes(group.totalTrackedMinutes)} tracked</span>
                  {isDefCat(group) ? <span className="pill">Default</span> : null}
                </div>
              </div>

              {groupPieChart.slices.length > 0 ? (
                <div className="category-pie-layout">
                  <div
                    className="category-pie"
                    aria-label={`Activity distribution for ${displayCategoryName(group.name)}`}
                    role="img"
                    style={{ background: groupPieChart.gradient }}
                  />

                  <div className="category-pie-legend">
                    {groupPieChart.slices.map((slice) => (
                      <div className="category-pie-legend-item" key={slice.id}>
                        <span
                          className="category-pie-color"
                          aria-hidden="true"
                          style={{ background: slice.color }}
                        />
                        <span className="category-pie-label">{slice.name}</span>
                        <span className="category-pie-value">{formatHoursFromMinutes(slice.minutes)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <p className="empty">No activities or child categories in this category yet.</p>
              )}

              {childGroups.length > 0 && (
                <div className="nested-category-cards">
                  {childGroups.map(renderStatsGroup)}
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
              <p className="empty">Add a category and an activity to start seeing progress here.</p>
            </article>
          );
        }
        return <div className="progress-grid">{topLevel.map(renderStatsGroup)}</div>;
      })()}
    </section>
  );
}
