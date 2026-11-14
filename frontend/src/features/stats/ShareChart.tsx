import type { Share } from "../../api/progress";
import { formatMinutes, formatPercent } from "../../lib/format";
import { percentOf } from "./chartScale";

interface ShareChartProps {
  shares: Share[];
}

/**
 * Actual vs. target share of the total time for every category and activity.
 * The bar is the actual share, the tick marks the target. Two encodings, so a legend is shown.
 */
export function ShareChart({ shares }: ShareChartProps) {
  const scale = Math.max(0.01, ...shares.flatMap((s) => [s.actualShare, s.targetShare]));

  return (
    <figure className="chart">
      <ul className="legend" aria-hidden="true">
        <li>
          <span className="legend__swatch" /> Actual
        </li>
        <li>
          <span className="legend__tick" /> Target
        </li>
      </ul>

      <ul className="shares">
        {shares.map((share) => (
          <li
            key={`${share.kind}-${share.id}`}
            className={`shares__row shares__row--${share.kind}`}
            data-tip={`${share.name} · ${formatPercent(share.actualShare)} actual, ${formatPercent(share.targetShare)} target`}
          >
            <span className="shares__name" style={{ paddingLeft: `${share.depth * 14}px` }}>
              {share.name}
            </span>
            <span className="shares__track" aria-hidden="true">
              <span className="shares__bar" style={{ width: `${percentOf(share.actualShare, scale)}%` }} />
              <span className="shares__target" style={{ left: `${percentOf(share.targetShare, scale)}%` }} />
            </span>
            <span className="shares__value">
              {formatPercent(share.actualShare)} <span className="muted">of {formatPercent(share.targetShare)}</span>
            </span>
          </li>
        ))}
      </ul>
      <figcaption className="muted">
        Share of all tracked time. Bar lengths are scaled to the largest value.
      </figcaption>

      <details className="table-view">
        <summary>Show as table</summary>
        <table>
          <thead>
            <tr>
              <th scope="col">Name</th>
              <th scope="col">Time</th>
              <th scope="col">Actual</th>
              <th scope="col">Target</th>
            </tr>
          </thead>
          <tbody>
            {shares.map((share) => (
              <tr key={`${share.kind}-${share.id}`}>
                <th scope="row" style={{ paddingLeft: `${share.depth * 14 + 8}px` }}>
                  {share.name}
                </th>
                <td>{formatMinutes(share.minutes)}</td>
                <td>{formatPercent(share.actualShare)}</td>
                <td>{formatPercent(share.targetShare)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </details>
    </figure>
  );
}
