import { useState } from "react";
import type { TrackAvailabilityResult } from "../types/match";

type ResultsListProps = {
  results: TrackAvailabilityResult[];
};

// Each row shows the song title and artist name, linked to their JOYSOUND
// pages when available — there's no separate "match type" indicator, the
// presence of a link on the title or the artist name says it all. Rows with
// at least one link sort first, so the songs worth singing surface above the
// ones JOYSOUND doesn't have at all.
function ResultsList({ results }: ResultsListProps) {
  const [query, setQuery] = useState("");

  const needle = query.trim().toLowerCase();
  const matching = needle
    ? results.filter(
        (result) =>
          result.title.toLowerCase().includes(needle) ||
          result.artist.toLowerCase().includes(needle),
      )
    : results;

  const sorted = [...matching].sort((a, b) => matchRank(a) - matchRank(b));

  return (
    <div className="results">
      <input
        type="search"
        className="results-search"
        placeholder="Search songs or artists…"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        aria-label="Search songs or artists"
      />
      {sorted.length > 0 ? (
        <table className="results-table">
          <thead>
            <tr>
              <th>Song</th>
              <th>Artist</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((result) => (
              <tr key={`${result.artist}::${result.title}`}>
                <td>{linkOrText(result.title, result.songJoysoundUrl)}</td>
                <td>{linkOrText(result.artist, result.artistJoysoundUrl)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <p className="results-empty">No matches</p>
      )}
    </div>
  );
}

function matchRank(result: TrackAvailabilityResult): number {
  return result.songJoysoundUrl || result.artistJoysoundUrl ? 0 : 1;
}

function linkOrText(text: string, url?: string) {
  if (!url) return text;
  return (
    <a href={url} target="_blank" rel="noopener noreferrer">
      {text}
    </a>
  );
}

export default ResultsList;
