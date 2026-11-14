import type { Stats } from "../../api/progress";
import { formatMinutes } from "../../lib/format";

interface SummaryTilesProps {
  stats: Stats;
  /** Name of the selected category or activity; empty when everything is shown. */
  scopeName: string;
}

export function SummaryTiles({ stats, scopeName }: SummaryTilesProps) {
  const activeDays = stats.daily.filter((day) => day.minutes > 0).length;
  const suffix = scopeName ? ` · ${scopeName}` : "";

  return (
    <div className="tiles">
      <Tile label={`Last ${stats.days} days${suffix}`} value={formatMinutes(stats.periodMinutes)} />
      <Tile label={`Active days${suffix}`} value={`${activeDays} of ${stats.days}`} />
      {/* The all-time total always covers everything, so say so when a filter is active. */}
      <Tile label={scopeName ? "All time · everything" : "All time"} value={formatMinutes(stats.totalMinutes)} />
    </div>
  );
}

function Tile({ label, value }: { label: string; value: string }) {
  return (
    <div className="card tile">
      <p className="tile__label">{label}</p>
      <p className="tile__value">{value}</p>
    </div>
  );
}
