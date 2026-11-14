import { useState } from "react";

import { Message } from "../../ui/Message";
import { useLibrary } from "../library/useLibrary";
import { DailyChart } from "./DailyChart";
import { RecentEntries } from "./RecentEntries";
import { ScopeSelect, scopeLabel } from "./ScopeSelect";
import type { ScopeKey } from "./scope";
import { ShareChart } from "./ShareChart";
import { SummaryTiles } from "./SummaryTiles";
import { useStats } from "./useStats";

const PERIODS = [7, 30, 90] as const;
const DEFAULT_PERIOD = 30;

export function StatsPage() {
  const [days, setDays] = useState<number>(DEFAULT_PERIOD);
  const [scope, setScope] = useState<ScopeKey>("");
  const { stats, loading, error, reload } = useStats(days, scope);
  const library = useLibrary();
  const scopeName = scopeLabel(scope, library.categories, library.activities);
  const [version, setVersion] = useState(0);

  function changed() {
    setVersion((n) => n + 1);
    void reload();
    void library.reload();
  }

  return (
    <div className="stack stack--wide">
      <div className="page-head">
        <h1>Progress</h1>
        <div className="chips" role="group" aria-label="Period">
          {PERIODS.map((period) => (
            <button
              key={period}
              type="button"
              className={`chip${days === period ? " chip--active" : ""}`}
              aria-pressed={days === period}
              onClick={() => setDays(period)}
            >
              {period} days
            </button>
          ))}
        </div>
      </div>

      <ScopeSelect value={scope} categories={library.categories} activities={library.activities} onChange={setScope} />

      {error && <Message kind="error">{error}</Message>}
      {loading && !stats && <Message>Loading…</Message>}

      {stats && (
        <>
          <SummaryTiles stats={stats} scopeName={scopeName} />

          <section className="card stack">
            <h2>{scopeName ? `Time per day · ${scopeName}` : "Time per day"}</h2>
            <DailyChart daily={stats.daily} />
          </section>

          <section className="card stack">
            <h2>Target vs. actual</h2>
            {stats.shares.length === 0 ? (
              <Message>Add activities and log some time to see how you are doing.</Message>
            ) : (
              <ShareChart shares={stats.shares} />
            )}
          </section>
        </>
      )}

      <RecentEntries activities={library.activities} version={version} onChanged={changed} />
    </div>
  );
}
