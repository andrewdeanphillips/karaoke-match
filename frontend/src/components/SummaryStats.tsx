import type { MatchResult } from "../types/match";

type SummaryStatsProps = {
  result: MatchResult;
};

function SummaryStats({ result }: SummaryStatsProps) {
  const checkedCount = result.results.length;
  const availableCount = result.results.filter((r) => r.songJoysoundUrl || r.artistJoysoundUrl).length;
  const coveragePercent = checkedCount === 0 ? 0 : Math.round((availableCount / checkedCount) * 100);
  const partial = checkedCount < result.totalTracks;

  return (
    <section className="stats-section">
      <div className="stats">
        <div className="stat">
          <span className="stat-value">{checkedCount}</span>
          <span className="stat-label">
            song{checkedCount === 1 ? "" : "s"} checked
            {partial && ` of ${result.totalTracks}`}
          </span>
        </div>
        <div className="stat">
          <span className="stat-value">{availableCount}</span>
          <span className="stat-label">available on JOYSOUND</span>
        </div>
        <div className="stat">
          <span className="stat-value">{coveragePercent}%</span>
          <span className="stat-label">coverage</span>
        </div>
      </div>
      {partial && (
        <p className="status">
          JOYSOUND lookups are limited per request, so only the first{" "}
          {checkedCount} of {result.totalTracks} songs could be checked this
          time.
        </p>
      )}
    </section>
  );
}

export default SummaryStats;
