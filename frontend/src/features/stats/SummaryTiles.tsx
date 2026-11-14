import type { Stats } from "../../api/progress";
import { formatMinutes } from "../../lib/format";

interface SummaryTilesProps {
  stats: Stats;
}

export function SummaryTiles({ stats }: SummaryTilesProps) {
  const activeDays = stats.daily.filter((day) => day.minutes > 0).length;

  return (
    <div className="tiles">
      <Tile label={`Last ${stats.days} days`} value={formatMinutes(stats.periodMinutes)} />
      <Tile label="Active days" value={`${activeDays} of ${stats.days}`} />
      <Tile label="All time" value={formatMinutes(stats.totalMinutes)} />
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
