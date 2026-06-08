import type { MatchResult } from "../types/match";

type SummaryStatsProps = {
  result: MatchResult;
};

function SummaryStats({ result }: SummaryStatsProps) {
  const checkedCount = result.results.length;
  const availableCount = result.results.filter((r) => r.available).length;
  const coveragePercent = checkedCount === 0 ? 0 : Math.round((availableCount / checkedCount) * 100);
  const partial = checkedCount < result.totalArtists;

  return (
    <section>
      <p>
        {checkedCount} artist{checkedCount === 1 ? "" : "s"} checked
        {partial && ` (out of ${result.totalArtists} found)`}
      </p>
      <p>{availableCount} available on JOYSOUND</p>
      <p>{coveragePercent}% coverage</p>
      {partial && (
        <p>
          JOYSOUND lookups are limited per request, so only the first{" "}
          {checkedCount} of {result.totalArtists} artists could be checked
          this time.
        </p>
      )}
    </section>
  );
}

export default SummaryStats;
