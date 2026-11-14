import type { DayMinutes } from "../../api/progress";
import { formatDay, formatMinutes, formatMinutesShort } from "../../lib/format";
import { axisTicks, niceMax, percentOf } from "./chartScale";

interface DailyChartProps {
  daily: DayMinutes[];
}

/** Time per day as columns. One series, so the card title names it and no legend is needed. */
export function DailyChart({ daily }: DailyChartProps) {
  const max = niceMax(Math.max(0, ...daily.map((d) => d.minutes)));
  const first = daily[0];
  const middle = daily[Math.floor(daily.length / 2)];
  const last = daily[daily.length - 1];

  return (
    <figure className="chart">
      <div
        className="chart__plot"
        role="img"
        aria-label={`Minutes per day over the last ${daily.length} days. A table follows.`}
      >
        {axisTicks(max).map((tick) => (
          <div key={tick} className="chart__gridline" style={{ bottom: `${percentOf(tick, max)}%` }}>
            <span className="chart__tick">{formatMinutesShort(tick)}</span>
          </div>
        ))}
        <div className="chart__cols" style={{ gridTemplateColumns: `repeat(${daily.length}, minmax(0, 1fr))` }}>
          {daily.map((day) => (
            <div key={day.date} className="chart__col" data-tip={`${formatDay(day.date)} · ${formatMinutes(day.minutes)}`}>
              {day.minutes > 0 && <div className="chart__bar" style={{ height: `${percentOf(day.minutes, max)}%` }} />}
            </div>
          ))}
        </div>
      </div>

      {first && middle && last && (
        <div className="chart__xlabels" aria-hidden="true">
          <span>{formatDay(first.date)}</span>
          <span>{formatDay(middle.date)}</span>
          <span>{formatDay(last.date)}</span>
        </div>
      )}

      <details className="table-view">
        <summary>Show as table</summary>
        <table>
          <thead>
            <tr>
              <th scope="col">Day</th>
              <th scope="col">Time</th>
            </tr>
          </thead>
          <tbody>
            {[...daily].reverse().map((day) => (
              <tr key={day.date}>
                <th scope="row">{formatDay(day.date)}</th>
                <td>{formatMinutes(day.minutes)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </details>
    </figure>
  );
}
